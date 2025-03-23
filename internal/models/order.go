package models

import "time"

type OrderType string

const (
	Limit  OrderType = "limit"
	Market OrderType = "market"
)

type Order struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Pair      string    `json:"pair"` // e.g., "BTC/USD"
	Type      OrderType `json:"type"`
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	Side      string    `json:"side"` // "buy" or "sell"
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}
