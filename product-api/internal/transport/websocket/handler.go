package websocket

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/product-api/internal/events"
	"net/http"
)

type Handler struct {
	Upgrader websocket.Upgrader
	Log      hclog.Logger
	EventBus *events.EventBus[any]
}

type Message struct {
	EventType string      `json:"event-type"`
	Data      interface{} `json:"data"`
}

func NewHandler(log hclog.Logger, eventBus *events.EventBus[any]) *Handler {
	return &Handler{
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Implement origin checks if necessary
				return true
			},
		},
		Log:      log,
		EventBus: eventBus,
	}
}

func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.Log.Error("Unable to upgrade to WebSocket", "error", err)
		return
	}
	defer conn.Close()

	// Subscribe to events
	subscriber := h.EventBus.Subscribe()
	defer h.EventBus.Unsubscribe(subscriber)

	// Create a done channel to signal when to connection is closed
	done := make(chan struct{})

	// Handle incoming requests (if any)
	go h.readPump(conn, done)

	// Listen for events and send them to WebSocket client
	for {
		select {
		case event := <-subscriber:
			var message Message
			switch e := event.(type) {
			case events.PriceUpdate:
				message = Message{
					EventType: "price_update",
					Data:      e,
				}
			case events.ProductAdded:
				message = Message{
					EventType: "product_added",
					Data:      e,
				}
			case events.ProductUpdated:
				message = Message{
					EventType: "product_updated",
					Data:      e,
				}
			case events.ProductDeleted:
				message = Message{
					EventType: "product_deleted",
					Data:      e,
				}
			default:
				h.Log.Warn("Unknown event type", "event", e)
				continue
			}

			payload, err := json.Marshal(message)
			if err != nil {
				h.Log.Error("Error marshalling message", "error", err)
				continue
			}

			// Send the message over the WebSocket connection
			err = conn.WriteMessage(websocket.TextMessage, payload)
			if err != nil {
				h.Log.Error("Error writing message to WebSocket", "error", err)
				// Connection might be closed, exit the loop
				return
			}
		case <-done:
			// The connection has been closed
			h.Log.Info("WebSocket connection closed by the client")
			return
		}
	}
}

func (h *Handler) readPump(conn *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.Log.Error("Error reading message", "error", err)
			}
			break
		}
	}
}
