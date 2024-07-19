package service

import (
	"context"
	"go.uber.org/zap"

	"github.com/SversusN/gophermart/internal/model"
	storage "github.com/SversusN/gophermart/internal/repository"
	errs "github.com/SversusN/gophermart/pkg/errors"
	"github.com/SversusN/gophermart/pkg/util"
)

type AccrualOrderService struct {
	repo storage.AccrualOrderRepoInterface
	log  *zap.Logger
}

func NewAccrualOrderService(repo storage.AccrualOrderRepoInterface, log *zap.Logger) *AccrualOrderService {
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

	err := a.repo.SaveOrder(ctx, &order)

	if err != nil {

		switch err.(type) {
		case errs.OrderAlreadyUploadedAnotherUserError:
			return errs.OrderAlreadyUploadedAnotherUserError{}
		case errs.OrderAlreadyUploadedCurrentUserError:
			return errs.OrderAlreadyUploadedCurrentUserError{}
		default:
			a.log.Error("AccrualOrderService.LoadOrder: SaveOrder db error")
			return err
		}
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
