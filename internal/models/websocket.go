package models

import "time"

// WebSocketMessage  represent a generic message sent over websocket

type WebSocketMessage struct {
	Type    string      `json:"type"`    // eg. "subscribe", "unsubscribe", "snapshot", "error"
	Payload interface{} `json:"payload"` // Dynamic payload based on message type
}

// SubscriptionRequest represents a client's request to subscribe to a trading pair
type SubscriptionRequest struct {
	Pair string `json:"pair"` // e.g. "BTC/USD"
}

// OrderBookSnapshot  represents a snapshot of the order book sent to clients
type OrderBookSnapshot struct {
	Pair      string    `json:"pair"`
	Bids      []Order   `json:"bids"`
	Asks      []Order   `json:"asks"`
	Timestamp time.Time `json:"timestamp"`
}

// OrderBookUpdate represents an incremental update to the order book
type OrderBookUpdate struct {
	Pair      string    `json:"pair"`
	BidUpdate *Order    `json:"bid_update,omitempty"` // Nil if no update
	AskUpdate *Order    `json:"ask_update,omitempty"` // Nil if no update
	Timestamp time.Time `json:"timestamp"`
}

// ErrorMessage represents an error sent to the client
type ErrorMessage struct {
	Code    int    `json:"code"`    // e.g. 400, 404, 500
	Message string `json:"message"` // e.g., "Invalid pair"
}
