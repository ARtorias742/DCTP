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
	Pair      string    `json:"pair"`
	Type      string    `json:"type"`
	Price     string    `json:"price"`
	Quantity  string    `json:"quantity"`
	Side      string    `json:"side"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}
