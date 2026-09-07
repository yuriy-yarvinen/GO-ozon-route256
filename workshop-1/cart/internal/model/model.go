package model

import "errors"

type Item struct {
	SKU   uint32 `json:"sku"`
	Count uint16 `json:"count"`
}

var (
	ErrValidation      = errors.New("validation error")
	ErrNotFound        = errors.New("not found")
	ErrInsufficient    = errors.New("insufficient stocks")
	ErrProductNotFound = errors.New("product not found")
)
