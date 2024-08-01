package handler

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SversusN/gophermart/internal/controller/http/handlers/mock"
	"github.com/SversusN/gophermart/internal/model"
	storage "github.com/SversusN/gophermart/internal/repository"
	"github.com/SversusN/gophermart/internal/service"
	errs "github.com/SversusN/gophermart/pkg/errors"
	"github.com/SversusN/gophermart/pkg/logger"
)

func parceUint(value string) uint64 {
	num, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}
	return num
}

const (
	secretKey = "be55d1079e6c6167118ac91318fe"
)

func generatePasswordHash(password string) string {
	hash := sha1.New()
	hash.Write([]byte(password))
	return fmt.Sprintf("%x", hash.Sum([]byte(secretKey)))
}

func TestRegisterUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log, _ := logger.InitLogger()
	auth := http_mocks.NewMockAuthRepoInterface(ctrl)
	acc := http_mocks.NewMockAccrualOrderInterface(ctrl)
	withdraw := http_mocks.NewMockWithdrawOrderRepoInterface(ctrl)
	var rep = storage.Repository{Auth: auth,
		Accrual:  acc,
		Withdraw: withdraw}
	services := service.NewService(&rep, log)
	h := NewHandler(services, log)
	r := h.CreateRouter()

	type want struct {
		statusCode int
	}
	type request struct {
		body        string
		contentType string
	}
	type storageRes struct {
		userID int
		err    error
	}

	tests := []struct {
		name       string
		request    request
		want       want
		storageRes *storageRes
	}{
		{
			name: "good request",
			request: request{
				body:        `{"login":"user","password":"1"}`,
				contentType: "application/json",
			},
			want: want{
				statusCode: http.StatusOK,
			},
			storageRes: &storageRes{
				userID: 0,
				err:    nil,
			},
		},
		{
			name: "Bad content type",
			request: request{
				body:        `{"login":"user","password":"1"}`,
				contentType: "text",
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
			storageRes: nil,
		},
		{
			name: "Bad model",
			request: request{
				body:        `{,"password":"1"}`,
				contentType: "application/json",
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
			storageRes: nil,
		},
		{
			name: "Login conflict",
			request: request{
				body:        `{"login":"user","password":"1"}`,
				contentType: "application/json",
			},
			want: want{
				statusCode: http.StatusConflict,
			},
			storageRes: &storageRes{
				userID: 0,
				err:    errs.ConflictLoginError{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(tt.request.body))
			req.Header.Set("Content-Type", tt.request.contentType)
			w := httptest.NewRecorder()

			if tt.storageRes != nil {
				auth.EXPECT().
					CreateUser(gomock.Any(), &model.User{ID: 0, Login: "user", Password: generatePasswordHash("1")}).Times(1).
					Return(tt.storageRes.userID, tt.storageRes.err)
			} else {
				auth.EXPECT().CreateUser(gomock.Any(), &model.User{ID: 0, Password: generatePasswordHash("1")}).Times(0)
			}
			r.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
		})
	}
}

func TestLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log, _ := logger.InitLogger()
	auth := http_mocks.NewMockAuthRepoInterface(ctrl)
	acc := http_mocks.NewMockAccrualOrderInterface(ctrl)
	withdraw := http_mocks.NewMockWithdrawOrderRepoInterface(ctrl)
	var rep = storage.Repository{Auth: auth,
		Accrual:  acc,
		Withdraw: withdraw}
	services := service.NewService(&rep, log)
	h := NewHandler(services, log)
	r := h.CreateRouter()

	type want struct {
		statusCode int
	}
	type request struct {
		body        string
		contentType string
	}
	type storageRes struct {
		userID int
		err    error
	}

	tests := []struct {
		name       string
		request    request
		want       want
		storageRes *storageRes
	}{
		{
			name: "Good  login",
			request: request{
				body:        `{"login":"user","password":"1"}`,
				contentType: "application/json",
			},
			want: want{
				statusCode: http.StatusOK,
			},
			storageRes: &storageRes{
				userID: 0,
				err:    nil,
			},
		},
		{
			name: "Bad model",
			request: request{
				body:        `{"login":"",}`,
				contentType: "application/json",
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
			storageRes: nil,
		},
		{
			name: "BAD password",
			request: request{
				body:        `{"login":"user","password":"123"}`,
				contentType: "application/json",
			},
			want: want{
				statusCode: http.StatusUnauthorized,
			},
			storageRes: &storageRes{
				userID: 0,
				err:    errs.AuthenticationError{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(tt.request.body))
			req.Header.Set("Content-Type", tt.request.contentType)
			w := httptest.NewRecorder()
			if tt.name == "BAD password" {
				auth.EXPECT().GetUserID(gomock.Any(), &model.User{ID: tt.storageRes.userID, Login: "user", Password: generatePasswordHash("123")}).
					Return(0, errs.AuthenticationError{})
			} else if tt.storageRes != nil {
				auth.EXPECT().GetUserID(gomock.Any(), &model.User{ID: tt.storageRes.userID, Login: "user", Password: generatePasswordHash("1")}).
					Return(tt.storageRes.userID, nil).Times(1)
			} else {
				auth.EXPECT().GetUserID(gomock.Any(), &model.User{ID: 0, Login: "user", Password: generatePasswordHash("1")}).Times(0)
			}

			r.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
		})
	}
}

func TestCreateOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log, _ := logger.InitLogger()
	auth := http_mocks.NewMockAuthRepoInterface(ctrl)
	acc := http_mocks.NewMockAccrualOrderInterface(ctrl)
	withdraw := http_mocks.NewMockWithdrawOrderRepoInterface(ctrl)
	var rep = storage.Repository{Auth: auth,
		Accrual:  acc,
		Withdraw: withdraw}
	services := service.NewService(&rep, log)
	h := NewHandler(services, log)
	r := h.CreateRouter()

	type want struct {
		statusCode int
	}
	type request struct {
		orderNum    string
		contentType string
		isAuth      bool
	}
	type storageRes struct {
		orderID int
		err     error
	}

	tests := []struct {
		name       string
		request    request
		want       want
		storageRes *storageRes
	}{
		{
			name: "Good request",
			request: request{
				orderNum:    "12345678903",
				contentType: "text/plain",
				isAuth:      true,
			},
			want: want{
				statusCode: http.StatusAccepted,
			},
			storageRes: &storageRes{
				orderID: 1,
				err:     nil,
			},
		},
		{
			name: "Double order detect (one user)",
			request: request{
				orderNum:    "12345678903",
				contentType: "text/plain",
				isAuth:      true,
			},
			want: want{
				statusCode: http.StatusOK,
			},
			storageRes: &storageRes{
				orderID: 1,
				err:     errs.OrderAlreadyUploadedCurrentUserError{},
			},
		},
		{
			name: "Double order detect (another user)",
			request: request{
				orderNum:    "12345678903",
				contentType: "text/plain",
				isAuth:      true,
			},
			want: want{
				statusCode: http.StatusConflict,
			},
			storageRes: &storageRes{
				orderID: 1,
				err:     errs.OrderAlreadyUploadedAnotherUserError{},
			},
		},
		{
			name: "No Auth user",
			request: request{
				orderNum:    "12345678903",
				contentType: "text/plain",
				isAuth:      false,
			},
			want: want{
				statusCode: http.StatusUnauthorized,
			},
			storageRes: nil,
		},
		{
			name: "Bad number",
			request: request{
				orderNum:    "123456",
				contentType: "text/plain",
				isAuth:      true,
			},
			want: want{
				statusCode: http.StatusUnprocessableEntity,
			},
			storageRes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader(tt.request.orderNum))
			if tt.request.isAuth {
				req.Header.Set("Content-Type", tt.request.contentType)
				token, _ := services.Auth.GenerateToken(&model.User{ID: 1, Login: "user", Password: "1"}, h.TokenAuth)
				req.Header.Set("Authorization", "Bearer "+token)
			}
			w := httptest.NewRecorder()
			num, err := strconv.ParseUint(tt.request.orderNum, 10, 64)
			if err != nil {
				return
			}

			if tt.storageRes != nil {

				acc.EXPECT().SaveOrder(gomock.Any(), &model.AccrualOrder{
					UserID:  1,
					Number:  num,
					Status:  model.StatusNEW,
					Accrual: 0,
				}).
					Return(tt.storageRes.err).
					Times(1)
			}
			r.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
		})
	}
}

func TestGetOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log, _ := logger.InitLogger()
	auth := http_mocks.NewMockAuthRepoInterface(ctrl)
	acc := http_mocks.NewMockAccrualOrderInterface(ctrl)
	withdraw := http_mocks.NewMockWithdrawOrderRepoInterface(ctrl)
	var rep = storage.Repository{Auth: auth,
		Accrual:  acc,
		Withdraw: withdraw}
	services := service.NewService(&rep, log)
	h := NewHandler(services, log)
	r := h.CreateRouter()

	type want struct {
		statusCode int
		body       string
	}
	type request struct {
		isAuth bool
	}
	type storageRes struct {
		orders []model.AccrualOrder
		err    error
	}

	tests := []struct {
		name       string
		request    request
		want       want
		storageRes *storageRes
	}{
		{
			name: "No Data",
			request: request{
				isAuth: true,
			},
			want: want{
				statusCode: http.StatusNoContent,
				body:       "",
			},
			storageRes: &storageRes{
				err:    nil,
				orders: []model.AccrualOrder{},
			},
		},
		{
			name: "Good request",
			request: request{
				isAuth: true,
			},
			want: want{
				statusCode: http.StatusOK,
				body: `
					[
						{
									"number": "9278923470",
									"status": "PROCESSED",
									"accrual": 500,
									"uploaded_at": "2024-01-01T01:01:01+03:00",
									"user_id":1
							},
							{
									"number": "346436439",
									"status": "INVALID",
									"uploaded_at": "2024-01-01T02:01:01+03:00",
									"user_id":2
							}
					]
				`,
			},
			storageRes: &storageRes{
				err: nil,
				orders: []model.AccrualOrder{
					{
						UserID:     1,
						Number:     parceUint("9278923470"),
						Status:     model.StatusPROCESSED,
						Accrual:    500,
						UploadedAt: time.Date(2024, 01, 01, 01, 01, 01, 0, time.Local),
					},
					{
						UserID:     2,
						Number:     parceUint("346436439"),
						Status:     model.StatusINVALID,
						UploadedAt: time.Date(2024, 01, 01, 02, 01, 01, 0, time.Local),
					},
				},
			},
		},

		{
			name: "No Auth user",
			request: request{
				isAuth: false,
			},
			want: want{
				statusCode: http.StatusUnauthorized,
				body:       "",
			},
			storageRes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)

			if tt.request.isAuth {
				token, _ := services.Auth.GenerateToken(&model.User{ID: 1, Login: "user", Password: "1"}, h.TokenAuth)
				req.Header.Set("Authorization", "Bearer "+token)
			}
			w := httptest.NewRecorder()

			if tt.storageRes != nil {
				acc.EXPECT().
					GetUploadedOrders(gomock.Any(), 1).
					Return(tt.storageRes.orders, tt.storageRes.err).
					Times(1)
			} else {
				acc.EXPECT().
					GetUploadedOrders(gomock.Any(), 1).Times(0)
			}
			r.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.want.statusCode, resp.StatusCode)

			if tt.want.body != "" {
				resBody, readErr := io.ReadAll(resp.Body)
				require.NoError(t, readErr)
				assert.JSONEq(t, tt.want.body, string(resBody))
			}
		})
	}
}

func TestWithdraw(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log, _ := logger.InitLogger()
	auth := http_mocks.NewMockAuthRepoInterface(ctrl)
	acc := http_mocks.NewMockAccrualOrderInterface(ctrl)
	withdraw := http_mocks.NewMockWithdrawOrderRepoInterface(ctrl)
	var rep = storage.Repository{Auth: auth,
		Accrual:  acc,
		Withdraw: withdraw}
	services := service.NewService(&rep, log)
	h := NewHandler(services, log)
	r := h.CreateRouter()

	type want struct {
		statusCode int
	}
	type request struct {
		body        string
		contentType string
		isAuth      bool
	}
	type repRes struct {
		err error
	}
	type repReq struct {
		Sum    float64
		Number string
	}

	tests := []struct {
		name    string
		request request
		want    want
		repRes  *repRes
		repReq  *repReq
	}{
		{
			name: "GOOD request",
			request: request{
				body:        `{"order":"12345678903","sum":100}`,
				contentType: "application/json",
				isAuth:      true,
			},
			want: want{
				statusCode: http.StatusOK,
			},
			repRes: &repRes{
				err: nil,
			},
			repReq: &repReq{
				Sum:    100,
				Number: "12345678903",
			},
		},
		{
			name: "NO honey",
			request: request{
				body:        `{"order":"12345678903","sum":100}`,
				contentType: "application/json",
				isAuth:      true,
			},
			want: want{
				statusCode: http.StatusPaymentRequired,
			},
			repRes: &repRes{
				err: errs.ShowMeTheMoney{},
			},
			repReq: &repReq{
				Sum:    100,
				Number: "12345678903",
			},
		},
		{
			name: "BAD NUMBER",
			request: request{
				body:        `{"order":"123456","sum":100}`,
				contentType: "application/json",
				isAuth:      true,
			},
			want: want{
				statusCode: http.StatusUnprocessableEntity,
			},
			repRes: nil,
			repReq: nil,
		},
		{
			name: "BAD SUM",
			request: request{
				body:        `{"order":"12345678903","sum":-100}`,
				contentType: "application/json",
				isAuth:      true,
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
			repRes: nil,
			repReq: nil,
		},
		{
			name: "BAD auth",
			request: request{
				body:        `{"order":"12345678903","sum":-100}`,
				contentType: "application/json",
				isAuth:      false,
			},
			want: want{
				statusCode: http.StatusUnauthorized,
			},
			repRes: nil,
			repReq: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader(tt.request.body))
			req.Header.Set("Content-Type", tt.request.contentType)
			if tt.request.isAuth {
				token, _ := services.Auth.GenerateToken(&model.User{ID: 1, Login: "user", Password: "1"}, h.TokenAuth)
				req.Header.Set("Authorization", "Bearer "+token)
			}

			w := httptest.NewRecorder()

			if tt.repRes != nil {
				withdraw.EXPECT().
					DeductPoints(gomock.Any(), &model.WithdrawOrder{
						UserID: 1,
						Order:  parceUint(tt.repReq.Number),
						Sum:    float32(tt.repReq.Sum),
					}).
					Return(tt.repRes.err).
					Times(1)
			}

			r.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
		})
	}
}

func TestGetWithdraws(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log, _ := logger.InitLogger()
	auth := http_mocks.NewMockAuthRepoInterface(ctrl)
	acc := http_mocks.NewMockAccrualOrderInterface(ctrl)
	withdraw := http_mocks.NewMockWithdrawOrderRepoInterface(ctrl)
	var rep = storage.Repository{Auth: auth,
		Accrual:  acc,
		Withdraw: withdraw}
	services := service.NewService(&rep, log)
	h := NewHandler(services, log)
	r := h.CreateRouter()

	type want struct {
		statusCode int
		body       string
	}
	type request struct {
		isAuth bool
	}
	type storageRes struct {
		withdraws []model.WithdrawOrder
		err       error
	}

	tests := []struct {
		name       string
		request    request
		want       want
		storageRes *storageRes
	}{
		{
			name: "Good request",
			request: request{
				isAuth: true,
			},
			want: want{
				statusCode: http.StatusOK,
				body: `
					[
							{
									"order": "2377225624",
									"sum": 500.00,
									"processed_at": "2024-11-09T16:09:57+03:00"
							}
					]
				`,
			},
			storageRes: &storageRes{
				err: nil,
				withdraws: []model.WithdrawOrder{
					{UserID: 1,
						Order:       parceUint("2377225624"),
						Sum:         500.00,
						ProcessedAt: time.Date(2024, 11, 9, 16, 9, 57, 0, time.Local),
					},
				},
			},
		},
		{
			name: "NO DATA",
			request: request{
				isAuth: true,
			},
			want: want{
				statusCode: http.StatusNoContent,
				body:       "",
			},
			storageRes: &storageRes{
				err:       nil,
				withdraws: []model.WithdrawOrder{},
			},
		},
		{
			name: "NO AUTH",
			request: request{
				isAuth: false,
			},
			want: want{
				statusCode: http.StatusUnauthorized,
				body:       "",
			},
			storageRes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)

			if tt.request.isAuth {
				token, _ := services.Auth.GenerateToken(&model.User{ID: 1, Login: "user", Password: "1"}, h.TokenAuth)
				req.Header.Set("Authorization", "Bearer "+token)
			}

			w := httptest.NewRecorder()

			if tt.storageRes != nil {
				withdraw.EXPECT().
					GetWithdrawalOfPoints(gomock.Any(), 1).
					Return(tt.storageRes.withdraws, tt.storageRes.err).
					Times(1)
			}
			r.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.want.statusCode, resp.StatusCode)

			if tt.want.body != "" {
				resBody, readErr := io.ReadAll(resp.Body)
				require.NoError(t, readErr)
				assert.JSONEq(t, tt.want.body, string(resBody))
			}
		})
	}
}
