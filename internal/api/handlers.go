package api

import (
	"log"
	"net/http"
	"time"

	"github.com/ARtorias742/DCTP/internal/config"
	"github.com/ARtorias742/DCTP/internal/matching"
	"github.com/ARtorias742/DCTP/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Handler struct {
	cfg    *config.Config
	engine *matching.Engine
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewHandler(cfg *config.Config) *gin.Engine {
	engine := matching.NewEngine(cfg)
	h := &Handler{
		cfg:    cfg,
		engine: engine,
	}

	go func() {
		if err := engine.Start(); err != nil {
			panic(err)
		}
	}()

	r := gin.Default()
	r.POST("/orders", h.createOrder)
	r.GET("/orderbook/:pair", h.getOrderBook)
	r.GET("/ws", h.websocketHandler)
	return r
}

func (h *Handler) createOrder(c *gin.Context) {
	var order models.Order
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.engine.ProcessOrder(&order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order queued successfully", "order": order})
}

func (h *Handler) getOrderBook(c *gin.Context) {
	pair := c.Param("pair")
	book, ok := h.engine.GetOrderBook(pair) // Use the new method
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order book not found"})
		return
	}

	c.JSON(http.StatusOK, book)
}

func (h *Handler) websocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to establish WebSocket connection"})
		return
	}
	defer conn.Close()

	var subscribedPair string

	go func() {
		for {
			var msg models.WebSocketMessage
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error: %v", err)
				}
				return
			}

			switch msg.Type {
			case "subscribe":
				if sub, ok := msg.Payload.(map[string]interface{}); ok {
					if pair, ok := sub["pair"].(string); ok {
						subscribedPair = pair
						conn.WriteJSON(models.WebSocketMessage{
							Type:    "subscribed",
							Payload: map[string]string{"pair": pair},
						})
					} else {
						conn.WriteJSON(models.WebSocketMessage{
							Type:    "error",
							Payload: models.ErrorMessage{Code: 400, Message: "Invalid pair in subscription"},
						})
					}
				}
			case "unsubscribe":
				subscribedPair = ""
				conn.WriteJSON(models.WebSocketMessage{
					Type:    "unsubscribed",
					Payload: nil,
				})
			default:
				conn.WriteJSON(models.WebSocketMessage{
					Type:    "error",
					Payload: models.ErrorMessage{Code: 400, Message: "Unknown message type"},
				})
			}
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if subscribedPair == "" {
				continue
			}

			book, ok := h.engine.GetOrderBook(subscribedPair) // Use the new method here too
			var snapshot models.OrderBookSnapshot
			if ok {
				snapshot = models.OrderBookSnapshot{
					Pair:      subscribedPair,
					Bids:      book.Bids.Orders,
					Asks:      book.Asks.Orders,
					Timestamp: time.Now(),
				}
			} else {
				snapshot = models.OrderBookSnapshot{
					Pair:      subscribedPair,
					Bids:      []models.Order{},
					Asks:      []models.Order{},
					Timestamp: time.Now(),
				}
			}

			if err := conn.WriteJSON(models.WebSocketMessage{
				Type:    "snapshot",
				Payload: snapshot,
			}); err != nil {
				log.Printf("Failed to send WebSocket message: %v", err)
				return
			}

		case <-c.Request.Context().Done():
			log.Println("Client disconnected")
			return
		}
	}
}
