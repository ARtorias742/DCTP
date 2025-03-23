package models

import (
	"sort"
	"sync"
	"time"
)

// OrderBook represents the order book for a trading pair
type OrderBook struct {
	Pair       string             // Trading pair identifier (e.g., "BTC/USD")
	Bids       *OrderQueue        // Buy orders (highest price first)
	Asks       *OrderQueue        // Sell orders (lowest price first)
	LastUpdate time.Time          // Timestamp of last modification
	mu         sync.RWMutex       // Mutex for thread-safe access (if not handled by Engine)
	Volume     map[string]float64 // Tracks volume per price level (optional)
}

// OrderQueue is a custom structure for maintaining sorted orders
type OrderQueue struct {
	Orders     []Order           // List of orders
	PriceIndex map[float64][]int // Index of orders by price for quick lookup
	mu         sync.RWMutex      // Mutex for queue-specific operations
}

// NewOrderbook create a new order book for a trading pair
func NewOrderBook(pair string) *OrderBook {
	return &OrderBook{
		Pair:       pair,
		Bids:       &OrderQueue{Orders: make([]Order, 0), PriceIndex: make(map[float64][]int)},
		Asks:       &OrderQueue{Orders: make([]Order, 0), PriceIndex: make(map[float64][]int)},
		LastUpdate: time.Now(),
		Volume:     make(map[string]float64),
	}
}

// Lock aquires the order book's read-write mutex
func (ob *OrderBook) Lock() {
	ob.mu.Lock()
}

// Unlock release the order book's reaad-write mutex
func (ob *OrderBook) Unlock() {
	ob.mu.Unlock()
}

// AddOrder adds an order to the queue and updates indices

func (q *OrderQueue) AddOrder(order Order) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.Orders = append(q.Orders, order)
	idx := len(q.Orders) - 1
	q.PriceIndex[order.Price] = append(q.PriceIndex[order.Price], idx)

	// Sort by price and time (example for bids; reverse for asks)
	sort.Slice(q.Orders, func(i, j int) bool {
		if q.Orders[i].Price == q.Orders[j].Price {
			return q.Orders[i].CreatedAt.Before(q.Orders[j].CreatedAt)
		}
		return q.Orders[i].Price > q.Orders[j].Price // Highest first for bids
	})
}

// RemoveOrder removes an order by index
func (q *OrderQueue) RemoveOrder(idx int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	order := q.Orders[idx]
	q.Orders = append(q.Orders[:idx], q.Orders[idx+1:]...)

	// Updates price index
	indices := q.PriceIndex[order.Price]
	for i, v := range indices {
		if v == idx {
			q.PriceIndex[order.Price] = append(indices[:i], indices[idx+1:]...)
			break
		}
	}

	if len(q.PriceIndex[order.Price]) == 0 {
		delete(q.PriceIndex, order.Price)
	}
}
