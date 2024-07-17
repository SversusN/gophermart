package handler

import (
	"github.com/SversusN/gophermart/internal/controller/http/middlewares"
	"github.com/SversusN/gophermart/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"go.uber.org/zap"
)

const (
	signingKey = "JdjshJhdjnnd<SdjkaAkjd"
)

type Handler struct {
	Service   *service.ServiceCollection
	TokenAuth *jwtauth.JWTAuth
	log       *zap.Logger
}

func NewHandler(service *service.ServiceCollection, log *zap.Logger) *Handler {
	tokenAuth := jwtauth.New("HS256", []byte(signingKey), nil)

	return &Handler{
		Service:   service,
		TokenAuth: tokenAuth,
		log:       log,
	}
}

func (h *Handler) CreateRouter() *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Use(middlewares.GzipHandle)

	router.Group(func(router chi.Router) {
		router.Post("/api/user/register", h.registration)
		router.Post("/api/user/login", h.authentication)
	})

	router.Group(func(router chi.Router) {
		router.Use(jwtauth.Verifier(h.TokenAuth))
		router.Use(jwtauth.Authenticator(h.TokenAuth))

		router.Post("/api/user/orders", h.loadOrders)
		router.Get("/api/user/orders", h.getUploadedOrders)
		router.Post("/api/user/balance/withdraw", h.deductionOfPoints)
		router.Get("/api/user/withdrawals", h.getWithdrawalOfPoints)
		router.Get("/api/user/balance", h.getBalance)
	})

	return router
}
