package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"route256/cart/internal/model"
	"route256/cart/internal/pkg/ctxkey"
	cartsvc "route256/cart/internal/service/cart"
	"strings"
	"time"
)

const requestTimeout = 500 * time.Millisecond

type Handler struct {
	service *cartsvc.Service
}

func New(service *cartsvc.Service) http.Handler {
	handler := &Handler{service: service}
	mux := http.NewServeMux()
	mux.HandleFunc("/cart/item/add", handler.addItem)
	mux.HandleFunc("/cart/item/delete", handler.deleteItem)
	mux.HandleFunc("/cart/list", handler.listCart)
	mux.HandleFunc("/cart/clear", handler.clearCart)
	mux.HandleFunc("/cart/checkout", handler.checkout)
	return authorization(mux)
}

func authorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		next.ServeHTTP(w, r.WithContext(ctxkey.WithToken(r.Context(), token)))
	})
}

func (h *Handler) addItem(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	var request struct {
		User  int64  `json:"user"`
		SKU   uint32 `json:"sku"`
		Count uint16 `json:"count"`
	}
	if !decode(w, r, &request) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	if err := h.service.AddItem(ctx, request.User, request.SKU, request.Count); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, struct{}{})
}

func (h *Handler) deleteItem(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	var request struct {
		User int64  `json:"user"`
		SKU  uint32 `json:"sku"`
	}
	if !decode(w, r, &request) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	if err := h.service.DeleteItem(ctx, request.User, request.SKU); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, struct{}{})
}

func (h *Handler) listCart(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	user, ok := decodeUser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	cart, err := h.service.ListCart(ctx, user)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, cart)
}

func (h *Handler) clearCart(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	user, ok := decodeUser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	if err := h.service.ClearCart(ctx, user); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, struct{}{})
}

func (h *Handler) checkout(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}
	user, ok := decodeUser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	orderID, err := h.service.Checkout(ctx, user)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, struct {
		OrderID int64 `json:"orderID"`
	}{OrderID: orderID})
}

func decodeUser(w http.ResponseWriter, r *http.Request) (int64, bool) {
	var request struct {
		User int64 `json:"user"`
	}
	if !decode(w, r, &request) {
		return 0, false
	}
	return request.User, true
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
	case errors.Is(err, model.ErrNotFound), errors.Is(err, model.ErrProductNotFound):
		status = http.StatusNotFound
	case errors.Is(err, model.ErrInsufficient):
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
