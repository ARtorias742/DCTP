package matching

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ARtorias742/DCTP/internal/config"
	"github.com/ARtorias742/DCTP/internal/models"
)

type Engine struct {
	cfg       *config.Config
	orderBook map[string]*models.OrderBook // Changed to pointer
	mu        sync.RWMutex
	orderChan chan *models.Order
	shutdown  chan struct{}
	wg        sync.WaitGroup
}

func NewEngine(cfg *config.Config) *Engine {
	return &Engine{
		cfg:       cfg,
		orderBook: make(map[string]*models.OrderBook), // Updated to pointer type
		orderChan: make(chan *models.Order, 1000),
		shutdown:  make(chan struct{}),
	}
}

func (e *Engine) Start() error {
	if e.cfg == nil {
		return errors.New("nil configuration provided")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const numWorkers = 4
	e.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go e.worker(ctx, i)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Matching engine started")

	select {
	case <-sigChan:
		log.Println("Received shutdown signal")
	case <-e.shutdown:
		log.Println("Received internal shutdown")
	}

	close(e.shutdown)
	cancel()
	e.wg.Wait()
	log.Println("Matching engine stopped")
	return nil
}

func (e *Engine) GetOrderBook(pair string) (*models.OrderBook, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	book, ok := e.orderBook[pair]
	return book, ok
}

func (e *Engine) worker(ctx context.Context, id int) {
	defer e.wg.Done()
	log.Printf("Worker %d started", id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d shutting down", id)
			return
		case order := <-e.orderChan:
			if err := e.executeOrder(order); err != nil {
				log.Printf("Worker %d error processing order %s: %v", id, order.ID, err)
			}
		}
	}
}

func (e *Engine) ProcessOrder(order *models.Order) error {
	select {
	case e.orderChan <- order:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("timeout adding order to processing queue")
	}
}

func (e *Engine) executeOrder(order *models.Order) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	book, ok := e.orderBook[order.Pair]
	if !ok {
		book = models.NewOrderBook(order.Pair) // Already returns a pointer
		e.orderBook[order.Pair] = book
	}

	book.Lock()
	defer book.Unlock()

	if order.Quantity <= 0 || (order.Type == models.Limit && order.Price <= 0) {
		return errors.New("invalid order: quantity and price must be positive")
	}

	switch order.Side {
	case "buy":
		if e.matchBuyOrder(order, book) {
			book.LastUpdate = time.Now()
			return nil
		}
		book.Bids.AddOrder(*order)
	case "sell":
		if e.matchSellOrder(order, book) {
			book.LastUpdate = time.Now()
			return nil
		}
		book.Asks.AddOrder(*order)
	default:
		return errors.New("invalid order side")
	}

	order.Status = "open"
	book.LastUpdate = time.Now()
	return nil
}

func (e *Engine) matchBuyOrder(order *models.Order, book *models.OrderBook) bool {
	book.Lock()
	defer book.Unlock()

	remainingQty := order.Quantity

	for i := 0; i < len(book.Asks.Orders) && remainingQty > 0; i++ {
		ask := &book.Asks.Orders[i]
		if order.Type == models.Limit && order.Price < ask.Price {
			continue
		}

		matchedQty := min(remainingQty, ask.Quantity)
		remainingQty -= matchedQty
		ask.Quantity -= matchedQty

		if ask.Quantity <= 0 {
			book.Asks.RemoveOrder(i)
			i--
		}
	}

	order.Quantity = remainingQty
	if remainingQty <= 0 {
		order.Status = "filled"
		return true
	}
	return false
}

func (e *Engine) matchSellOrder(order *models.Order, book *models.OrderBook) bool {
	book.Lock()
	defer book.Unlock()

	remainingQty := order.Quantity

	for i := 0; i < len(book.Bids.Orders) && remainingQty > 0; i++ {
		bid := &book.Bids.Orders[i]
		if order.Type == models.Limit && order.Price > bid.Price {
			continue
		}

		matchedQty := min(remainingQty, bid.Quantity)
		remainingQty -= matchedQty
		bid.Quantity -= matchedQty

		if bid.Quantity <= 0 {
			book.Bids.RemoveOrder(i)
			i--
		}
	}

	order.Quantity = remainingQty
	if remainingQty <= 0 {
		order.Status = "filled"
		return true
	}
	return false
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
