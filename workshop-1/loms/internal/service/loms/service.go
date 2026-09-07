package loms

import (
	"context"
	"time"

	"route256/loms/internal/model"
)

type OrderRepository interface {
	Create(ctx context.Context, user int64, items []model.Item, status string) int64
	Get(ctx context.Context, orderID int64) (model.Order, error)
	SetStatus(ctx context.Context, orderID int64, status string) error
	ListAwaitingOlderThan(ctx context.Context, cutoff time.Time) []model.Order
}

type StockRepository interface {
	GetAvailable(ctx context.Context, sku uint32) uint64
	Reserve(ctx context.Context, sku uint32, count uint16) error
	Release(ctx context.Context, sku uint32, count uint16) error
	RemoveReserved(ctx context.Context, sku uint32, count uint16) error
}

type Service struct {
	orders OrderRepository
	stocks StockRepository
}

func New(orders OrderRepository, stocks StockRepository) *Service {
	return &Service{orders: orders, stocks: stocks}
}

func (s *Service) CreateOrder(ctx context.Context, user int64, items []model.Item) (int64, error) {
	if len(items) == 0 {
		return 0, model.ErrValidation
	}

	orderID := s.orders.Create(ctx, user, items, model.StatusNew)
	reserved := make([]model.Item, 0, len(items))
	for _, item := range items {
		if err := s.stocks.Reserve(ctx, item.SKU, item.Count); err != nil {
			for _, reservedItem := range reserved {
				_ = s.stocks.Release(ctx, reservedItem.SKU, reservedItem.Count)
			}
			_ = s.orders.SetStatus(ctx, orderID, model.StatusFailed)
			return 0, model.ErrInsufficient
		}
		reserved = append(reserved, item)
	}

	if err := s.orders.SetStatus(ctx, orderID, model.StatusAwaitingPayment); err != nil {
		return 0, err
	}
	return orderID, nil
}

func (s *Service) OrderInfo(ctx context.Context, orderID int64) (model.Order, error) {
	return s.orders.Get(ctx, orderID)
}

func (s *Service) PayOrder(ctx context.Context, orderID int64) error {
	order, err := s.orders.Get(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != model.StatusAwaitingPayment {
		return model.ErrInvalidStatus
	}
	for _, item := range order.Items {
		if err := s.stocks.RemoveReserved(ctx, item.SKU, item.Count); err != nil {
			return err
		}
	}
	return s.orders.SetStatus(ctx, orderID, model.StatusPayed)
}

func (s *Service) CancelOrder(ctx context.Context, orderID int64) error {
	order, err := s.orders.Get(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != model.StatusAwaitingPayment {
		return model.ErrInvalidStatus
	}
	for _, item := range order.Items {
		if err := s.stocks.Release(ctx, item.SKU, item.Count); err != nil {
			return err
		}
	}
	return s.orders.SetStatus(ctx, orderID, model.StatusCancelled)
}

func (s *Service) StockInfo(ctx context.Context, sku uint32) uint64 {
	return s.stocks.GetAvailable(ctx, sku)
}
