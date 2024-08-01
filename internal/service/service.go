package service

import (
	"context"
	"github.com/go-chi/jwtauth/v5"
	"go.uber.org/zap"

	"github.com/SversusN/gophermart/internal/model"
	"github.com/SversusN/gophermart/internal/repository"
)

type AuthServiceInterface interface {
	CreateUser(ctx context.Context, user *model.User) error
	AuthenticationUser(ctx context.Context, user *model.User) error
	GenerateToken(user *model.User, tokenAuth *jwtauth.JWTAuth) (string, error)
}

type AccrualOrderServiceInterface interface {
	LoadOrder(ctx context.Context, numOrder uint64, userID int) error
	GetUploadedOrders(ctx context.Context, userID int) ([]model.AccrualOrder, error)
}

type WithdrawOrderServiceInterface interface {
	DeductionOfPoints(ctx context.Context, order *model.WithdrawOrder) error
	GetBalance(ctx context.Context, userID int) (float32, float32)
	GetWithdrawalOfPoints(ctx context.Context, userID int) ([]model.WithdrawOrder, error)
}

type ServiceCollection struct {
	Auth     AuthServiceInterface
	Accrual  AccrualOrderServiceInterface
	Withdraw WithdrawOrderServiceInterface
}

func NewService(r *storage.Repository, log *zap.Logger) *ServiceCollection {
	return &ServiceCollection{
		Auth:     NewAuthService(r.Auth, log),
		Accrual:  NewAccrualOrderService(r.Accrual, log),
		Withdraw: NewWithdrawOrderService(r.Withdraw, log),
	}
}
