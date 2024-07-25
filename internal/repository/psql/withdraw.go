package postgres

import (
	"context"
	"database/sql"
	"fmt"
	errs "github.com/SversusN/gophermart/pkg/errors"
	"go.uber.org/zap"
	"time"

	"github.com/SversusN/gophermart/internal/model"
)

type WithdrawOrderRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewWithdrawOrderPostgres(db *sql.DB, log *zap.Logger) *WithdrawOrderRepository {
	return &WithdrawOrderRepository{
		db:  db,
		log: log,
	}
}

func (w *WithdrawOrderRepository) GetAccruals(ctx context.Context, UserID int) float32 {
	row := w.db.QueryRowContext(ctx, "SELECT SUM(amount) FROM public.accruals WHERE user_id=$1", UserID)
	var accruals float32
	_ = row.Scan(&accruals)

	return accruals
}

func (w *WithdrawOrderRepository) GetWithdrawals(ctx context.Context, UserID int) float32 {
	row := w.db.QueryRowContext(ctx, "SELECT SUM(amount) FROM public.withdrawals WHERE user_id=$1", UserID)
	var withdrawals float32
	_ = row.Scan(&withdrawals)

	return withdrawals
}

func (w *WithdrawOrderRepository) DeductPoints(ctx context.Context, order *model.WithdrawOrder) (err error) {
	order.ProcessedAt = time.Now()
	tx, err := w.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			txError := tx.Rollback()
			if txError != nil {
				err = fmt.Errorf("balance DeductPoints rollback error %s: %s", txError.Error(), err.Error())
				w.log.Error(err.Error())
			}
		}
	}()
	var canDeduct int
	wasUsed := tx.QueryRowContext(ctx, `SELECT user_id FROM  public.withdrawals WHERE order_num = $1 LIMIT 1`, order.Order)
	err = wasUsed.Scan(canDeduct)
	if err == nil {
		if canDeduct == order.UserID {
			return errs.OrderAlreadyUploadedCurrentUserError{}
		} else {
			return errs.OrderAlreadyUploadedAnotherUserError{}
		}
	}

	//https://t.me/bushigo/36
	sumAcc := 0.0
	sumWd := 0.0
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(a.amount),0) as TotalBalance FROM (SELECT * FROM public.accruals FOR UPDATE) a WHERE a.user_id=$1`, order.UserID).Scan(&sumAcc)
	if err != nil {
		w.log.Error(err.Error())
		return err
	}
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(a.amount),0) as TotalBalance FROM (SELECT * FROM public.withdrawals FOR UPDATE) a WHERE a.user_id=$1`, order.UserID).Scan(&sumWd)
	if err != nil {
		w.log.Error(err.Error())
		return err
	}
	if sumAcc-sumWd <= 0 {
		return errs.ShowMeTheMoney{}
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO public.withdrawals(order_num, user_id, amount, processed_at) VALUES ($1,$2,$3,$4)",
		order.Order, order.UserID, order.Sum, order.ProcessedAt)

	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "UPDATE users SET current = current - $1, withdrawal = withdrawal + $1 WHERE id = $2", order.Sum, order.UserID)
	if err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func (w *WithdrawOrderRepository) GetWithdrawalOfPoints(ctx context.Context, userID int) ([]model.WithdrawOrder, error) {
	rows, err := w.db.QueryContext(ctx, "SELECT order_num, amount, processed_at FROM public.withdrawals WHERE user_id =$1 ORDER BY processed_at", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []model.WithdrawOrder
	for rows.Next() {
		var order model.WithdrawOrder
		err = rows.Scan(&order.Order, &order.Sum, &order.ProcessedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}
