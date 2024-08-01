package service

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/SversusN/gophermart/internal/model"
	storage "github.com/SversusN/gophermart/internal/repository"
	errs "github.com/SversusN/gophermart/pkg/errors"
)

type WithdrawOrderService struct {
	rep storage.WithdrawOrderRepoInterface
	log *zap.Logger
}

func NewWithdrawOrderService(rep storage.WithdrawOrderRepoInterface, log *zap.Logger) *WithdrawOrderService {
	return &WithdrawOrderService{
		rep: rep,
		log: log,
	}
}

func (w WithdrawOrderService) GetBalance(ctx context.Context, userID int) (float32, float32) {
	accruals := w.rep.GetAccruals(ctx, userID)
	withdrawn := w.rep.GetWithdrawals(ctx, userID)
	return accruals, withdrawn
}

func (w WithdrawOrderService) DeductionOfPoints(ctx context.Context, order *model.WithdrawOrder) error {
	err := w.rep.DeductPoints(ctx, order)
	if errors.Is(err, errs.ShowMeTheMoney{}) {
		w.log.Error("no more money")
		return err
	}
	if err != nil {
		w.log.Error("WithdrawOrderService.DeductionOfPoints: DeductPoints db error")
		return err
	}

	return nil
}

func (w *WithdrawOrderService) GetWithdrawalOfPoints(ctx context.Context, userID int) ([]model.WithdrawOrder, error) {
	orders, err := w.rep.GetWithdrawalOfPoints(ctx, userID)
	if err != nil {
		w.log.Error("WithdrawOrderService.GetWithdrawalOfPoints: GetWithdrawalOfPoints db error")
		return nil, err
	}
	return orders, nil
}
