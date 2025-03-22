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
	orderBook map[string]*OrderBook
	mu        sync.RWMutex
	orderChan chan *models.Order // Channel for incoming ordera
	shutdown  chan struct{}      // Signal for shutdown
	wg        sync.WaitGroup     // Wait group for goroutines
}

type OrderBook struct {
	Bids []models.Order // Buy orders sorted by price  (highest first)
	Asks []models.Order // Sell orders sortes by price (lowest first)
}

func NewEngine(cfg *config.Config) *Engine {
	return &Engine{
		cfg:       cfg,
		orderBook: make(map[string]*OrderBook),
		orderChan: make(chan *models.Order, 1000), // Buffered channel for orders
		shutdown:  make(chan struct{}),
	}
}

func (e *Engine) start() error {

	// Validate configuration
	if e.cfg == nil {
		return errors.New("nil configuration provided")
	}

	// create context for shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start order processing workers
	const numWorkers = 4 // Number of concurrent workers
	e.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go e.worker(ctx, i)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Matching engine started")

	// Main loop
	select {
	case <-sigChan:
		log.Println("Received shutdown signal")

	case <-e.shutdown:
		log.Println("Received internal shutdown")
	}

	// Initiate shutdown
	close(e.shutdown)
	cancel()

	// Wait for workers to complete
	e.wg.Wait()
	log.Println("Matching engine stopped")
	return nil
}

// worker processes order from the channel
func (e *Engine) worker(ctx context.Context, id int) {
	defer e.wg.Done()
	log.Printf("Worker %d started", id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d shutting down", id)

			return

		case order := <-e.orderChan:
			if err := e.ProcessOrder(order); err != nil {
				log.Printf("Worker %d error processing order %s : %v", id, order.ID, err)
			}
		}
	}
}

// ProcessOrder adds an order to the processing queue
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

	book ,ok := e.orderBook[order.Pair]
	if !ok {
		book = &OrderBook {
			Bids: make([]models.Order, 0),
			Asks: make([]models.Order, 0),
		}
		e.orderBook[order.Pair] = book
	}

	if order.Quantity <= 0 || (order.Type == models.Limit && order.Price <= 0) {
		return errors.New("invalid order: quantity and price must be positive")
	}

	switch order.Side {
	case "buy":
		if e.matchBuyOrder(order, book) {
			return nil
		}
		book.Bids = append(book.Bids, *order)
		sort.Slice(book.Bids, func(i, j int) bool {
			if book.Bids[i].Price == book.Bids[j].Price {
				return book.Bids[i].CreatedAt.Before.(book.Bids[j].CreatedAt)
			}
			return book.Bids[i].Price > book.Bids[j].Price
		})

	case "sell":
		if e.matchSellOrder(order, book) {
			return nil
		}
		book.Asks = append(book.Asks, *order)
		sort.Slice(book.Asks, func(i,j int) bool {
			if book.Asks[i].Price == book.Asks[j].Price {
				return book.Asks[i].CreatedAt.Before(book.Asks[j].CreatedAt)
			}
			return book.Asks[i].Price < book.Asks[j].Price
		})

	default:
		return errors.New("invalid order side")
	}

	order.Status = "open"
	return nil
}

func (e *Engine) matchBuyOrder(order *models.Order, book *OrderBook) bool {
	remainingQty := order.Quantity

	for i := 0; i < len(book.Asks); i++ {
		if remainingQty <= 0 {
			break
		}

		ask := &book.Asks[i]

		if order.Type == models.Limit && order.Price < ask.Price {
			continue
		}

		matchedQty := min(remainingQty, ask.Quantity)
		remainingQty -= matchedQty
		ask.Quantity -= matchedQty

		if ask.Quantity <= 0 {
			book.Asks = append(book.Asks[:i], book.Asks[i+1:]...)
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


// matchSellOrder attempts to match a sell order against existing buy orders
func (e *Engine) matchSellOrder(order *models.Order, book *orderBook) bool {
	remainingQty := order.Quantity

	for i := 0; i < len(book.Bids); i++ {
		if remainingQty <= 0{
			break
		}

		bid := &book.Bids[i]

		if order.Type == models.Limit && order.Price > bid.Price {
			continue
		}

		matchedQty := min(remainingQty, bid.Quantity)
		remainingQty -= matchedQty
		bid.Quantity -= matchedQty

		if bid.Quantity <= 0 {
			book.Bids = append(book.Bids[:i], book.Bids[i+1:]...)
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
