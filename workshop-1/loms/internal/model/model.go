package model

import "errors"

const (
	StatusNew             = "new"
	StatusAwaitingPayment = "awaiting payment"
	StatusFailed          = "failed"
	StatusPayed           = "payed"
	StatusCancelled       = "cancelled"
)

type Item struct {
	SKU   uint32 `json:"sku"`
	Count uint16 `json:"count"`
}

type Order struct {
	OrderID int64
	User    int64
	Items   []Item
	Status  string
}

var (
	ErrNotFound      = errors.New("not found")
	ErrInsufficient  = errors.New("insufficient stocks")
	ErrInvalidStatus = errors.New("invalid status")
	ErrValidation    = errors.New("validation error")
)
