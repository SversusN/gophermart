package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/SversusN/gophermart/internal/model"
	"github.com/go-chi/jwtauth/v5"

	errs "github.com/SversusN/gophermart/pkg/errors"
)

func (h *Handler) writeToken(w http.ResponseWriter, user *model.User, nameFunc string) {
	token, err := h.Service.Auth.GenerateToken(user, h.TokenAuth)
	if err != nil {
		h.log.Error("writeToken: %s - token generate error")
		http.Error(w, errs.InternalServerError, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Authorization", "Bearer "+token)
}

func (h *Handler) readUserData(w http.ResponseWriter, r *http.Request, user *model.User, nameFunc string) error {
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		h.log.Error("Handler.readingUserData: %s - header read error")
		http.Error(w, errs.BadData, http.StatusBadRequest)
		return errs.CheckError{}
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("Handler.readingUserData: %s - body read error")
		http.Error(w, errs.BadData, http.StatusBadRequest)
		return err
	}

	err = json.Unmarshal(body, &user)
	if err != nil {
		h.log.Error("Handler.readingUserData: %s - json read error")
		http.Error(w, errs.BadData, http.StatusBadRequest)
		return err
	}

	if user.Login == "" || user.Password == "" {
		http.Error(w, "empty login or password", http.StatusBadRequest)
		return err
	}
	return nil
}

func (h *Handler) getUserIDFromToken(w http.ResponseWriter, r *http.Request, nameFunc string) (int, error) {
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		h.log.Error("Handler.getUserIDFromToken: %s - jwt claims error")
		http.Error(w, errs.InternalServerError, http.StatusInternalServerError)
		return 0, err
	}

	//https://github.com/go-chi/jwtauth/blob/master/_example/main.go
	userID, err := strconv.Atoi(fmt.Sprintf("%v", claims["user_id"]))
	if err != nil {
		h.log.Error("Handler.getUserIDFromToken: %s - conv string to int")
		http.Error(w, errs.InternalServerError, http.StatusInternalServerError)
		return 0, err
	}

	return userID, nil
}
