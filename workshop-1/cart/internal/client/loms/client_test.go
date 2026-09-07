package loms

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"route256/cart/internal/model"
	"testing"
	"time"
)

func TestNewUsesDefaultHTTPTimeout(t *testing.T) {
	t.Parallel()

	client := New("http://loms", nil)
	if client.httpClient.Timeout != requestTimeout {
		t.Fatalf("http timeout = %v, want %v", client.httpClient.Timeout, requestTimeout)
	}
}

func TestClientStockInfo(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/stock/info" {
			t.Errorf("request = %s %s, want POST /stock/info", r.Method, r.URL.Path)
		}

		var request struct {
			SKU uint32 `json:"sku"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request.SKU != 1001 {
			t.Errorf("sku = %d, want 1001", request.SKU)
		}
		_ = json.NewEncoder(w).Encode(struct {
			Count uint64 `json:"count"`
		}{Count: 7})
	}))
	defer server.Close()

	count, err := New(server.URL, server.Client()).StockInfo(context.Background(), 1001)
	if err != nil {
		t.Fatalf("StockInfo() error = %v", err)
	}
	if count != 7 {
		t.Fatalf("StockInfo() = %d, want 7", count)
	}
}

func TestClientCreateOrder(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/order/create" {
			t.Errorf("request = %s %s, want POST /order/create", r.Method, r.URL.Path)
		}

		var request struct {
			User  int64        `json:"user"`
			Items []model.Item `json:"items"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request.User != 42 || len(request.Items) != 1 || request.Items[0] != (model.Item{SKU: 1001, Count: 2}) {
			t.Errorf("request body = %+v", request)
		}
		_ = json.NewEncoder(w).Encode(struct {
			OrderID int64 `json:"orderID"`
		}{OrderID: 99})
	}))
	defer server.Close()

	orderID, err := New(server.URL, server.Client()).CreateOrder(
		context.Background(),
		42,
		[]model.Item{{SKU: 1001, Count: 2}},
	)
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if orderID != 99 {
		t.Fatalf("CreateOrder() = %d, want 99", orderID)
	}
}

func TestClientMapsStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantErr    error
	}{
		{name: "insufficient", statusCode: http.StatusPreconditionFailed, wantErr: model.ErrInsufficient},
		{name: "not found", statusCode: http.StatusNotFound, wantErr: model.ErrNotFound},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			_, err := New(server.URL, server.Client()).StockInfo(context.Background(), 1001)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("StockInfo() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientAddsRequestDeadline(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		assertRequestDeadline(t, r.Context())
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
			Header:     make(http.Header),
		}, nil
	})}

	_, _ = New("http://loms", httpClient).StockInfo(context.Background(), 1001)
}

func assertRequestDeadline(t *testing.T, ctx context.Context) {
	t.Helper()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("request context has no deadline")
		return
	}
	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > requestTimeout {
		t.Errorf("request deadline remaining = %v, want (0, %v]", remaining, requestTimeout)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := f(request)
	if response != nil && response.Body == nil {
		response.Body = http.NoBody
	}
	return response, err
}
