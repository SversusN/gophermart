package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/SversusN/gophermart/config"
	agent "github.com/SversusN/gophermart/internal/accrualagent/service"
	app "github.com/SversusN/gophermart/internal/app"
	handler "github.com/SversusN/gophermart/internal/controller/http/handlers"
	repository "github.com/SversusN/gophermart/internal/repository"
	psql "github.com/SversusN/gophermart/internal/repository/psql"
	"github.com/SversusN/gophermart/internal/service"
	"github.com/SversusN/gophermart/pkg/logger"
)

func main() {
	log, err := logger.InitLogger()
	if err != nil {
		log.Fatal("error init logger")
	}

	defer log.Sync()
	zp := log.Sugar()

	conf, err := config.NewConfig()
	if err != nil {
		zp.Fatalf("failed to retrieve env variables, %v", err)
	}

	db, err := psql.NewPsql(conf.DatabaseURI)
	if err != nil {
		zp.Fatalf("DB connection error %v", err)
	}

	err = db.Init(conf.DatabaseURI)
	if err != nil {
		zp.Fatalf("failed to create db table %v", err)
	}

	ctx, stopping := context.WithCancel(context.Background())
	defer stopping()

	repos := repository.NewRepository(db.DB, log)
	services := service.NewService(repos, log)
	handlers := handler.NewHandler(services, log)
	//настройка воркера
	agentRepo := repository.NewAgentRepository(db.DB, log)
	newAgent := agent.NewAgent(agentRepo, conf.AccrualSystemAddress, log)
	wg := sync.WaitGroup{}
	newAgent.Start(ctx, &wg)

	server := app.NewServer(conf, handlers.CreateRouter())

	//завершаемся по книжке...
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-termChan
		zp.Infof("server shutdown success")
		stopping()
		if err = server.Stop(ctx); err != nil {
			zp.Fatalf("server shutdown error %v", err)
		}
		wg.Wait()
	}()

	if err = server.Run(); err != nil && err != http.ErrServerClosed {
		zp.Fatalf("server run error %v", err)
	}

}
