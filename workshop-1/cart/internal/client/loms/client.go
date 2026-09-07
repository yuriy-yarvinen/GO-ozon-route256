package loms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"route256/cart/internal/model"
	"strings"
	"time"
)

const requestTimeout = 500 * time.Millisecond

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 500 * time.Millisecond}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *Client) StockInfo(ctx context.Context, sku uint32) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	body, err := json.Marshal(struct {
		SKU uint32 `json:"sku"`
	}{SKU: sku})
	if err != nil {
		return 0, fmt.Errorf("marshal stock info request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/stock/info",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, fmt.Errorf("create stock info request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("request stock info: %w", err)
	}
	defer response.Body.Close()

	if err := mapResponseError(response.StatusCode); err != nil {
		return 0, err
	}

	var result struct {
		Count uint64 `json:"count"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode stock info response: %w", err)
	}
	return result.Count, nil
}

func (c *Client) CreateOrder(ctx context.Context, user int64, items []model.Item) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	body, err := json.Marshal(struct {
		User  int64        `json:"user"`
		Items []model.Item `json:"items"`
	}{
		User:  user,
		Items: items,
	})
	if err != nil {
		return 0, fmt.Errorf("marshal create order request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/order/create",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, fmt.Errorf("create order request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("request create order: %w", err)
	}
	defer response.Body.Close()

	if err := mapResponseError(response.StatusCode); err != nil {
		return 0, err
	}

	var result struct {
		OrderID int64 `json:"orderID"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode create order response: %w", err)
	}
	return result.OrderID, nil
}

func mapResponseError(statusCode int) error {
	if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
		return nil
	}

	switch statusCode {
	case http.StatusBadRequest:
		return model.ErrValidation
	case http.StatusPreconditionFailed:
		return model.ErrInsufficient
	case http.StatusNotFound:
		return model.ErrNotFound
	default:
		return fmt.Errorf("loms response status %d: %w", statusCode, errors.New(http.StatusText(statusCode)))
	}
}
