package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"route256/loms/internal/handler"
	"route256/loms/internal/repository/order"
	"route256/loms/internal/repository/stock"
	lomssvc "route256/loms/internal/service/loms"
)

func newHandler() http.Handler {
	return handler.New(lomssvc.New(
		order.New(),
		stock.New(map[uint32]uint64{1001: 10}),
	))
}

func post(t *testing.T, h http.Handler, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	return response
}

func TestOrderLifecycleAndStockInfo(t *testing.T) {
	h := newHandler()

	created := post(t, h, "/order/create", `{"user":7,"items":[{"sku":1001,"count":3}]}`)
	if created.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var createResponse struct {
		OrderID int64 `json:"orderID"`
	}
	if err := json.NewDecoder(created.Body).Decode(&createResponse); err != nil {
		t.Fatal(err)
	}

	info := post(t, h, "/order/info", `{"orderID":1}`)
	if info.Code != http.StatusOK {
		t.Fatalf("info status=%d body=%s", info.Code, info.Body.String())
	}
	var infoResponse struct {
		Status string `json:"status"`
		User   int64  `json:"user"`
		Items  []struct {
			SKU   uint32 `json:"sku"`
			Count uint16 `json:"count"`
		} `json:"items"`
	}
	if err := json.NewDecoder(info.Body).Decode(&infoResponse); err != nil {
		t.Fatal(err)
	}
	if createResponse.OrderID != 1 || infoResponse.Status != "awaiting payment" ||
		infoResponse.User != 7 || len(infoResponse.Items) != 1 {
		t.Fatalf("create=%+v info=%+v", createResponse, infoResponse)
	}

	stockInfo := post(t, h, "/stock/info", `{"sku":1001}`)
	if stockInfo.Code != http.StatusOK || stockInfo.Body.String() != "{\"count\":7}\n" {
		t.Fatalf("stock status=%d body=%s", stockInfo.Code, stockInfo.Body.String())
	}

	paid := post(t, h, "/order/pay", `{"orderID":1}`)
	if paid.Code != http.StatusOK || paid.Body.String() != "{}\n" {
		t.Fatalf("pay status=%d body=%s", paid.Code, paid.Body.String())
	}
}

func TestHandlerErrorMapping(t *testing.T) {
	h := newHandler()
	created := post(t, h, "/order/create", `{"user":7,"items":[{"sku":1001,"count":1}]}`)
	if created.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	canceled := post(t, h, "/order/cancel", `{"orderID":1}`)
	if canceled.Code != http.StatusOK {
		t.Fatalf("cancel status=%d body=%s", canceled.Code, canceled.Body.String())
	}

	tests := []struct {
		name string
		path string
		body string
		want int
	}{
		{name: "validation", path: "/order/create", body: `{"user":7,"items":[]}`, want: http.StatusBadRequest},
		{name: "malformed json", path: "/order/info", body: `{`, want: http.StatusBadRequest},
		{name: "not found", path: "/order/info", body: `{"orderID":99}`, want: http.StatusNotFound},
		{name: "insufficient", path: "/order/create", body: `{"user":7,"items":[{"sku":1001,"count":11}]}`, want: http.StatusPreconditionFailed},
		{name: "invalid status", path: "/order/cancel", body: `{"orderID":1}`, want: http.StatusPreconditionFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := post(t, h, tt.path, tt.body)
			if response.Code != tt.want {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestHandlerRejectsWrongMethod(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/stock/info", nil)
	response := httptest.NewRecorder()
	newHandler().ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", response.Code)
	}
}
