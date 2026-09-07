package loms

import (
	"context"
	"errors"
	"fmt"
	"route256/loms/internal/model"
	"time"
)

type OrderRepository interface {
	Create(ctx context.Context, user int64, items []model.Item, status string) int64
	Get(ctx context.Context, orderID int64) (model.Order, error)
	SetStatus(ctx context.Context, orderID int64, status string) error
	CompareAndSwapStatus(ctx context.Context, orderID int64, from, to string) (model.Order, error)
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
			var rollbackErrs []error
			for _, reservedItem := range reserved {
				if releaseErr := s.stocks.Release(ctx, reservedItem.SKU, reservedItem.Count); releaseErr != nil {
					rollbackErrs = append(rollbackErrs, fmt.Errorf("release reserved stock for sku %d: %w", reservedItem.SKU, releaseErr))
				}
			}
			if statusErr := s.orders.SetStatus(ctx, orderID, model.StatusFailed); statusErr != nil {
				rollbackErrs = append(rollbackErrs, fmt.Errorf("set order %d status to failed: %w", orderID, statusErr))
			}
			if len(rollbackErrs) > 0 {
				return 0, errors.Join(rollbackErrs...)
			}
			return 0, err
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
	order, err := s.orders.CompareAndSwapStatus(ctx, orderID, model.StatusAwaitingPayment, model.StatusPayed)
	if err != nil {
		return err
	}
	for _, item := range order.Items {
		if err := s.stocks.RemoveReserved(ctx, item.SKU, item.Count); err != nil {
			return s.rollbackClaim(ctx, orderID, model.StatusPayed, err)
		}
	}
	return nil
}

func (s *Service) CancelOrder(ctx context.Context, orderID int64) error {
	order, err := s.orders.CompareAndSwapStatus(ctx, orderID, model.StatusAwaitingPayment, model.StatusCancelled)
	if err != nil {
		return err
	}
	for _, item := range order.Items {
		if err := s.stocks.Release(ctx, item.SKU, item.Count); err != nil {
			return s.rollbackClaim(ctx, orderID, model.StatusCancelled, err)
		}
	}
	return nil
}

func (s *Service) rollbackClaim(ctx context.Context, orderID int64, claimedStatus string, operationErr error) error {
	if _, err := s.orders.CompareAndSwapStatus(ctx, orderID, claimedStatus, model.StatusAwaitingPayment); err != nil {
		return errors.Join(operationErr, fmt.Errorf("roll back order %d status: %w", orderID, err))
	}
	return operationErr
}

func (s *Service) StockInfo(ctx context.Context, sku uint32) uint64 {
	return s.stocks.GetAvailable(ctx, sku)
}
