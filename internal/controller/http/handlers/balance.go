package handler

import (
	"encoding/json"
	errs "github.com/SversusN/gophermart/pkg/errors"
	"github.com/SversusN/gophermart/pkg/util"
	"io"
	"net/http"

	"github.com/SversusN/gophermart/internal/model"
)

// getCurrentBalance GET /api/user/balance - получение текущего баланса пользователя
func (h *Handler) getBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, err := h.getUserIDFromToken(w, r, "handler.getCurrentBalance")
	if err != nil {
		return
	}

	accruals, withdraws := h.Service.Withdraw.GetBalance(r.Context(), userID)

	balance := model.Balance{Current: accruals - withdraws, Withdrawn: withdraws}

	output, err := json.Marshal(balance)
	if err != nil {
		h.log.Error("Handler.getCurrentBalance: json write error")
		http.Error(w, errs.InternalServerError, http.StatusInternalServerError)
		return
	}
	w.Write(output)
}

// deductionOfPoints POST /api/user/balance/withdraw - запрос на списание средств
func (h *Handler) deductionOfPoints(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserIDFromToken(w, r, "handler.deductionOfPoints")
	if err != nil {
		return
	}
	//https://t.me/bushigo/21
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("Handler.deductionOfPoints: body read error")
		http.Error(w, "wrong input data", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var order *model.WithdrawOrder
	err = json.Unmarshal(body, &order)
	if err != nil {
		h.log.Error("Handler.deductionOfPoints: json read error")
		http.Error(w, errs.InternalServerError, http.StatusInternalServerError)
		return
	}
	if order.Sum < 0 {
		http.Error(w, errs.BadData, http.StatusBadRequest)
		return
	}
	if !util.ValidLuhn(order.Order) {
		http.Error(w, errs.BadData, http.StatusUnprocessableEntity)
		return
	}

	order.UserID = userID

	err = h.Service.Withdraw.DeductionOfPoints(r.Context(), order)

	switch err.(type) {
	case nil:
		w.WriteHeader(http.StatusOK)
	case errs.ShowMeTheMoney:
		http.Error(w, err.Error(), http.StatusPaymentRequired)
		return
	default:
		http.Error(w, errs.InternalServerError, http.StatusInternalServerError)
	}
}

func (h *Handler) getWithdrawalOfPoints(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, err := h.getUserIDFromToken(w, r, "handler.getCurrentBalance")
	if err != nil {
		return
	}

	orders, err := h.Service.Withdraw.GetWithdrawalOfPoints(r.Context(), userID)
	if err != nil {
		http.Error(w, errs.InternalServerError, http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	output, err := json.Marshal(orders)
	if err != nil {
		h.log.Error("Handler.getWithdrawalOfPoints: json marshal error")
		http.Error(w, errs.InternalServerError, http.StatusInternalServerError)
		return
	}
	w.Write(output)
}
