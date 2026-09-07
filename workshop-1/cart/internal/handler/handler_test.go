package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"route256/cart/internal/model"
	"route256/cart/internal/pkg/ctxkey"
	cartrepo "route256/cart/internal/repository/cart"
	cartsvc "route256/cart/internal/service/cart"
	"testing"
	"time"
)

type productClient struct {
	err   error
	token string
}

func (c *productClient) GetProduct(ctx context.Context, _ uint32) (string, uint32, error) {
	c.token, _ = ctxkey.TokenFromContext(ctx)
	return "product", 10, c.err
}

type lomsClient struct {
	stock    uint64
	err      error
	orderID  int64
	deadline time.Time
}

func (c *lomsClient) StockInfo(ctx context.Context, _ uint32) (uint64, error) {
	c.deadline, _ = ctx.Deadline()
	return c.stock, c.err
}

func (c *lomsClient) CreateOrder(context.Context, int64, []model.Item) (int64, error) {
	return c.orderID, c.err
}

func newHandler(products cartsvc.ProductClient, loms cartsvc.LOMSClient) http.Handler {
	return New(cartsvc.New(cartrepo.New(), products, loms))
}

func request(t *testing.T, handler http.Handler, path, body, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Authorization", authorization)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func TestAddItemPassesStrippedBearerTokenAndDeadline(t *testing.T) {
	products := &productClient{}
	loms := &lomsClient{stock: 5}
	response := request(t, newHandler(products, loms), "/cart/item/add",
		`{"user":1,"sku":1001,"count":1}`, "Bearer secret")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if products.token != "secret" {
		t.Fatalf("token = %q, want %q", products.token, "secret")
	}
	remaining := time.Until(loms.deadline)
	if remaining <= 0 || remaining > 500*time.Millisecond {
		t.Fatalf("handler deadline remaining = %s, want (0, 500ms]", remaining)
	}
}

func TestHandlerRoutes(t *testing.T) {
	handler := newHandler(&productClient{}, &lomsClient{stock: 5, orderID: 42})
	tests := []struct {
		path string
		body string
	}{
		{"/cart/item/add", `{"user":1,"sku":1001,"count":1}`},
		{"/cart/item/delete", `{"user":1,"sku":1001}`},
		{"/cart/list", `{"user":1}`},
		{"/cart/clear", `{"user":1}`},
		{"/cart/checkout", `{"user":1}`},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			response := request(t, handler, test.path, test.body, "")
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
			}
		})
	}
}

func TestCheckoutResponse(t *testing.T) {
	response := request(t, newHandler(&productClient{}, &lomsClient{orderID: 42}),
		"/cart/checkout", `{"user":1}`, "")
	var body struct {
		OrderID int64 `json:"orderID"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.OrderID != 42 {
		t.Fatalf("orderID = %d, want 42", body.OrderID)
	}
}

func TestErrorMapping(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{"validation", model.ErrValidation, http.StatusBadRequest},
		{"not found", model.ErrNotFound, http.StatusNotFound},
		{"product not found", model.ErrProductNotFound, http.StatusNotFound},
		{"insufficient", model.ErrInsufficient, http.StatusPreconditionFailed},
		{"unexpected", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := request(t, newHandler(&productClient{err: test.err}, &lomsClient{stock: 5}),
				"/cart/item/add", `{"user":1,"sku":1001,"count":1}`, "")
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.status, response.Body.String())
			}
		})
	}
}

func TestRejectsInvalidJSONAndNonPost(t *testing.T) {
	handler := newHandler(&productClient{}, &lomsClient{stock: 5})
	if response := request(t, handler, "/cart/list", `{`, ""); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	req := httptest.NewRequest(http.MethodGet, "/cart/list", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
