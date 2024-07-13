package storage

import (
	"context"
	"database/sql"

	"go.uber.org/zap"

	modelBroker "github.com/SversusN/gophermart/internal/accrualagent/model"
	psql "github.com/SversusN/gophermart/internal/repository/psql"
)

type AgentRepoInterface interface {
	GetOrders(ctx context.Context, limit int) ([]modelBroker.Order, error)
	UpdateOrderAccruals(ctx context.Context, orderAccruals []modelBroker.OrderAccrual) error
}

type AgentRepository struct {
	AgentRepoInterface
}

func NewAgentRepository(db *sql.DB, log *zap.Logger) *AgentRepository {
	return &AgentRepository{
		AgentRepoInterface: psql.NewAgentPostgres(db, log),
	}
}
