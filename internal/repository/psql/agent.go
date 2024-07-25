package postgres

import (
	"context"
	"database/sql"
	"go.uber.org/zap"

	"github.com/SversusN/gophermart/internal/accrualagent/model"
)

type AgentPG struct {
	db  *sql.DB
	log *zap.Logger
}

func NewAgentPostgres(db *sql.DB, log *zap.Logger) *AgentPG {
	return &AgentPG{
		db:  db,
		log: log,
	}
}

func (a *AgentPG) GetOrders(ctx context.Context, limit int) ([]model.Order, error) {
	rows, err := a.db.QueryContext(ctx, "SELECT order_num, status FROM public.accruals WHERE status=$1 OR status=$2 ORDER BY uploaded_at limit $3", model.StatusNEW.String(), model.StatusPROCESSING.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []model.Order
	for rows.Next() {
		var order model.Order
		var status string
		err = rows.Scan(&order.Number, &status)
		if err != nil {
			return nil, err
		}
		order.Status, err = model.GetStatus(status)
		if err != nil {
			a.log.Error("accrualagent db GetOrdersForProcessing")
			return nil, err
		}
		orders = append(orders, order)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (a *AgentPG) UpdateOrderAccruals(ctx context.Context, orderAccruals []model.OrderAccrual) error {

	stmt, err := a.db.PrepareContext(ctx,
		"UPDATE public.accruals SET status=$1, amount=$2 WHERE order_num=$3")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, order := range orderAccruals {
		_, err = stmt.ExecContext(ctx, order.Status, order.Accrual, order.Order)
		if err != nil {
			return err
		}
	}
	return nil
}
