package matching

import (
	"testing"
	"time"

	"github.com/ARtorias742/DCTP/internal/config"
	"github.com/ARtorias742/DCTP/internal/models" // Ensure models package is imported
)

func TestEngine_ProcessOrder(t *testing.T) {
	cfg := &config.Config{}
	engine := NewEngine(cfg)

	// Start the engine in a goroutine
	errChan := make(chan error, 1) // Buffered channel for errors
	go func() {
		if err := engine.Start(); err != nil {
			errChan <- err
		}
	}()

	// Ensure engine starts properly
	select {
	case err := <-errChan:
		t.Fatalf("Engine start failed: %v", err)
	case <-time.After(100 * time.Millisecond): // Wait for startup
	}

	// Test buy order
	buyOrder := &models.Order{
		ID:        "1",
		UserID:    "user1",
		Pair:      "BTC/USD",
		Type:      models.Limit,
		Price:     50000.0,
		Quantity:  1.0,
		Side:      "buy",
		CreatedAt: time.Now(),
		Status:    "open",
	}

	if err := engine.ProcessOrder(buyOrder); err != nil {
		t.Errorf("Failed to process buy order: %v", err)
	}

	// Test sell order
	sellOrder := &models.Order{
		ID:        "2",
		UserID:    "user2",
		Pair:      "BTC/USD",
		Type:      models.Limit,
		Price:     50000.0,
		Quantity:  1.0,
		Side:      "sell",
		CreatedAt: time.Now(),
		Status:    "open",
	}

	if err := engine.ProcessOrder(sellOrder); err != nil {
		t.Errorf("Failed to process sell order: %v", err)
	}

	// Allow orders to be processed
	time.Sleep(200 * time.Millisecond)

	// Proper shutdown of the engine
	if engine.shutdown != nil {
		close(engine.shutdown)
	} else {
		t.Errorf("Engine shutdown channel is nil")
	}
}
