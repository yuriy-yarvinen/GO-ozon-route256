package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"route256/loms/internal/model"
	lomssvc "route256/loms/internal/service/loms"
)

const requestTimeout = 500 * time.Millisecond

type Handler struct {
	service *lomssvc.Service
}

func New(service *lomssvc.Service) http.Handler {
	handler := &Handler{service: service}
	mux := http.NewServeMux()
	mux.HandleFunc("/order/create", handler.createOrder)
	mux.HandleFunc("/order/info", handler.orderInfo)
	mux.HandleFunc("/order/pay", handler.payOrder)
	mux.HandleFunc("/order/cancel", handler.cancelOrder)
	mux.HandleFunc("/stock/info", handler.stockInfo)
	return mux
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	var request struct {
		User  int64        `json:"user"`
		Items []model.Item `json:"items"`
	}
	if !decode(w, r, &request) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	orderID, err := h.service.CreateOrder(ctx, request.User, request.Items)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, struct {
		OrderID int64 `json:"orderID"`
	}{OrderID: orderID})
}

func (h *Handler) orderInfo(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	var request struct {
		OrderID int64 `json:"orderID"`
	}
	if !decode(w, r, &request) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	order, err := h.service.OrderInfo(ctx, request.OrderID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, struct {
		Status string       `json:"status"`
		User   int64        `json:"user"`
		Items  []model.Item `json:"items"`
	}{
		Status: order.Status,
		User:   order.User,
		Items:  order.Items,
	})
}

func (h *Handler) payOrder(w http.ResponseWriter, r *http.Request) {
	h.changeOrder(w, r, h.service.PayOrder)
}

func (h *Handler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	h.changeOrder(w, r, h.service.CancelOrder)
}

func (h *Handler) changeOrder(
	w http.ResponseWriter,
	r *http.Request,
	change func(context.Context, int64) error,
) {
	if !requirePost(w, r) {
		return
	}
	var request struct {
		OrderID int64 `json:"orderID"`
	}
	if !decode(w, r, &request) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	if err := change(ctx, request.OrderID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, struct{}{})
}

func (h *Handler) stockInfo(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	var request struct {
		SKU uint32 `json:"sku"`
	}
	if !decode(w, r, &request) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	writeJSON(w, struct {
		Count uint64 `json:"count"`
	}{Count: h.service.StockInfo(ctx, request.SKU)})
}

func requirePost(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodPost {
		return true
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
	return false
}

func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	if err := json.NewDecoder(r.Body).Decode(value); err != nil {
		http.Error(w, model.ErrValidation.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, model.ErrValidation):
		status = http.StatusBadRequest
	case errors.Is(err, model.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, model.ErrInsufficient), errors.Is(err, model.ErrInvalidStatus):
		status = http.StatusPreconditionFailed
	}
	http.Error(w, err.Error(), status)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
