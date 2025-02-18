package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/SversusN/gophermart/internal/accrualagent/model"
)

const (
	timeoutClient       = 5
	limitWorkers        = 3
	bufSizeOrdersRecord = 3
	limitQuery          = 10
	timeoutLoadOrdersDB = 3
)

type AgentInterface interface {
	GetOrders(ctx context.Context, lim int) ([]model.Order, error)
	UpdateOrderAccruals(ctx context.Context, orderAccruals []model.OrderAccrual) error
}

type Agent struct {
	r                              AgentInterface
	client                         *http.Client
	accrualURL                     string
	bufOrderForRecord              []model.OrderAccrual
	chOrdersForProcessing          chan model.Order
	chOrdersAccrual                chan model.OrderAccrual
	chSignalGetOrdersForProcessing chan struct{}
	chLimitWorkers                 chan int
	log                            *zap.Logger
}

func NewAgent(r AgentInterface, accrualURL string, log *zap.Logger) *Agent {
	return &Agent{
		r:                              r,
		client:                         &http.Client{Timeout: time.Second * timeoutClient},
		accrualURL:                     accrualURL,
		bufOrderForRecord:              make([]model.OrderAccrual, 0, bufSizeOrdersRecord),
		chOrdersForProcessing:          make(chan model.Order),
		chOrdersAccrual:                make(chan model.OrderAccrual),
		chSignalGetOrdersForProcessing: make(chan struct{}),
		chLimitWorkers:                 make(chan int, limitWorkers),
		log:                            log,
	}
}

func (a *Agent) Start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(3)
	defer wg.Done()
	go a.GetOrders(ctx)
	go a.GetOrdersAccrual(ctx)
	go a.LoadOrdersAccrual(ctx)
}

func (a *Agent) GetOrders(ctx context.Context) {

	ticker := time.NewTicker(timeoutLoadOrdersDB * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-a.chSignalGetOrdersForProcessing:
			a.runGetOrdersForProcessing(ctx)
			ticker.Reset(timeoutLoadOrdersDB * time.Second)
		case <-ticker.C:
			a.runGetOrdersForProcessing(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) runGetOrdersForProcessing(ctx context.Context) {
	orders, err := a.r.GetOrders(ctx, limitQuery)
	if err != nil {
		a.log.Error("Agent.runGetOrdersForProcessing: GetOrdersForProcessing db error")
	}

	for _, numOrder := range orders {
		a.chOrdersForProcessing <- numOrder
	}
}

func (a *Agent) GetOrdersAccrual(ctx context.Context) {
	for {
		select {
		case order := <-a.chOrdersForProcessing:
			a.chLimitWorkers <- 1
			go a.getOrdersAccrualWorker(order)
		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) getOrdersAccrualWorker(order model.Order) {
	var orderAccrual model.OrderAccrual
	uri := fmt.Sprintf("%s%s%d", a.accrualURL, "/api/orders/", order.Number)
	err := a.getOrderFromAccrual(uri, &orderAccrual)
	if err != nil {
		<-a.chLimitWorkers
		return
	}
	if order.Status != model.StatusUNKNOWN && order.Status != orderAccrual.Status {
		a.chOrdersAccrual <- orderAccrual
		<-a.chLimitWorkers
	}
}

func (a *Agent) getOrderFromAccrual(url string, orderAccrual *model.OrderAccrual) error {
	resp, err := a.client.Get(url)
	if err != nil {
		a.log.Error("Agent.getJSONOrderFromAccrual: Get url error")
		return err
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		secondsString := resp.Header.Get("Retry-After")
		timeWait, err := time.ParseDuration(strings.Join([]string{secondsString, "s"}, ""))
		if err != nil {
			return err
		}
		time.Sleep(timeWait)
		return nil
	}
	if resp.StatusCode == http.StatusInternalServerError {
		a.log.Error("Agent.getJSONOrderFromAccrual: Get url internal server error")
		return nil
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&orderAccrual)

	if err != nil {
		a.log.Error("Agent.getJSONOrderFromAccrual: json decode error")
		return err
	}
	return nil
}

func (a *Agent) LoadOrdersAccrual(ctx context.Context) {
	ticker := time.NewTicker(timeoutLoadOrdersDB * time.Second)
	defer ticker.Stop()
	for {
		select {
		case order := <-a.chOrdersAccrual:
			a.bufOrderForRecord = append(a.bufOrderForRecord, order)
			if len(a.bufOrderForRecord) >= bufSizeOrdersRecord {
				a.send(ctx)
			}
			ticker.Reset(timeoutLoadOrdersDB * time.Second)
		case <-ticker.C:
			if len(a.bufOrderForRecord) > 0 {
				a.send(ctx)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) send(ctx context.Context) {
	ordersUpdate := make([]model.OrderAccrual, len(a.bufOrderForRecord))
	copy(ordersUpdate, a.bufOrderForRecord)
	a.bufOrderForRecord = make([]model.OrderAccrual, 0)
	go func() {
		err := a.r.UpdateOrderAccruals(ctx, ordersUpdate)
		if err != nil {
			a.log.Error("Agent.send: UpdateOrderAccruals db error")
			return
		}
		a.chSignalGetOrdersForProcessing <- struct{}{}
	}()
}
