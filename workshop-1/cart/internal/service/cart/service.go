package cart

import (
	"context"
	"route256/cart/internal/model"
)

type Repository interface {
	Add(ctx context.Context, userID int64, sku uint32, count uint16) error
	Delete(ctx context.Context, userID int64, sku uint32) error
	List(ctx context.Context, userID int64) []model.Item
	Clear(ctx context.Context, userID int64)
	GetCount(ctx context.Context, userID int64, sku uint32) uint16
}

type ProductClient interface {
	GetProduct(ctx context.Context, sku uint32) (name string, price uint32, err error)
}

type LOMSClient interface {
	StockInfo(ctx context.Context, sku uint32) (uint64, error)
	CreateOrder(ctx context.Context, userID int64, items []model.Item) (int64, error)
}

type Item struct {
	SKU   uint32 `json:"sku"`
	Count uint16 `json:"count"`
	Name  string `json:"name"`
	Price uint32 `json:"price"`
}

type Cart struct {
	Items      []Item `json:"items"`
	TotalPrice uint32 `json:"totalPrice"`
}

type Service struct {
	repository Repository
	products   ProductClient
	loms       LOMSClient
}

func New(repository Repository, products ProductClient, loms LOMSClient) *Service {
	return &Service{
		repository: repository,
		products:   products,
		loms:       loms,
	}
}

func (s *Service) AddItem(
	ctx context.Context,
	userID int64,
	sku uint32,
	count uint16,
) error {
	if userID <= 0 || sku == 0 || count == 0 {
		return model.ErrValidation
	}

	if _, _, err := s.products.GetProduct(ctx, sku); err != nil {
		return err
	}

	available, err := s.loms.StockInfo(ctx, sku)
	if err != nil {
		return err
	}
	existing := s.repository.GetCount(ctx, userID, sku)
	if uint64(existing)+uint64(count) > available {
		return model.ErrInsufficient
	}

	return s.repository.Add(ctx, userID, sku, count)
}

func (s *Service) DeleteItem(ctx context.Context, userID int64, sku uint32) error {
	if userID <= 0 || sku == 0 {
		return model.ErrValidation
	}
	return s.repository.Delete(ctx, userID, sku)
}

func (s *Service) ListCart(ctx context.Context, userID int64) (Cart, error) {
	if userID <= 0 {
		return Cart{}, model.ErrValidation
	}

	storedItems := s.repository.List(ctx, userID)
	result := Cart{Items: make([]Item, 0, len(storedItems))}
	for _, storedItem := range storedItems {
		name, price, err := s.products.GetProduct(ctx, storedItem.SKU)
		if err != nil {
			return Cart{}, err
		}
		result.Items = append(result.Items, Item{
			SKU:   storedItem.SKU,
			Count: storedItem.Count,
			Name:  name,
			Price: price,
		})
		result.TotalPrice += price * uint32(storedItem.Count)
	}
	return result, nil
}

func (s *Service) ClearCart(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return model.ErrValidation
	}
	s.repository.Clear(ctx, userID)
	return nil
}

func (s *Service) Checkout(ctx context.Context, userID int64) (int64, error) {
	if userID <= 0 {
		return 0, model.ErrValidation
	}

	items := s.repository.List(ctx, userID)
	orderID, err := s.loms.CreateOrder(ctx, userID, items)
	if err != nil {
		return 0, err
	}
	s.repository.Clear(ctx, userID)
	return orderID, nil
}
