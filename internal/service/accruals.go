package service

import (
	"context"
	"go.uber.org/zap"

	"github.com/SversusN/gophermart/internal/model"
	errs "github.com/SversusN/gophermart/pkg/errors"
	"github.com/SversusN/gophermart/pkg/util"
)

type AccrualOrderRepoContract interface {
	SaveOrder(ctx context.Context, order *model.AccrualOrder) error
	GetUserIDByNumberOrder(ctx context.Context, number uint64) int
	GetUploadedOrders(ctx context.Context, userID int) ([]model.AccrualOrder, error)
}

type AccrualOrderService struct {
	repo AccrualOrderRepoContract
	log  *zap.Logger
}

func NewAccrualOrderService(repo AccrualOrderRepoContract, log *zap.Logger) *AccrualOrderService {
	return &AccrualOrderService{
		repo: repo,
		log:  log,
	}
}

func (a *AccrualOrderService) LoadOrder(ctx context.Context, numOrder uint64, userID int) error {

	if !util.ValidLuhn(numOrder) {
		return errs.CheckError{}
	}

	order := model.AccrualOrder{
		Number: numOrder,
		UserID: userID,
		Status: model.StatusNEW,
	}

	userIDinDB := a.repo.GetUserIDByNumberOrder(ctx, order.Number)
	if userIDinDB != 0 {
		if userIDinDB == order.UserID {
			return errs.OrderAlreadyUploadedCurrentUserError{}
		} else {
			return errs.OrderAlreadyUploadedAnotherUserError{}
		}
	}

	err := a.repo.SaveOrder(ctx, &order)
	if err != nil {
		a.log.Error("AccrualOrderService.LoadOrder: SaveOrder db error")
		return err
	}

	return nil
}

func (a *AccrualOrderService) GetUploadedOrders(ctx context.Context, userID int) ([]model.AccrualOrder, error) {
	orders, err := a.repo.GetUploadedOrders(ctx, userID)
	if err != nil {
		a.log.Error("AccrualOrderService.GetUploadedOrders: GetUploadedOrders db error")
		return nil, err
	}
	return orders, nil
}
