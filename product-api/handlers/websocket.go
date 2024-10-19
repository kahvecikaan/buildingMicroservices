package handlers

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-hclog"
	"github.com/kahvecikaan/buildingMicroservices/product-api/events"
	"net/http"
)

type WebSocketHandler struct {
	upgrader websocket.Upgrader
	log      hclog.Logger
	eventBus *events.EventBus[any]
}

type WebSocketMessage struct {
	EventType string      `json:"event_type"`
	Data      interface{} `json:"data"`
}

func NewWebSocketHandler(log hclog.Logger, eventBus *events.EventBus[any]) *WebSocketHandler {
	return &WebSocketHandler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		log:      log,
		eventBus: eventBus,
	}
}

func (wsh *WebSocketHandler) HandleWebSocket(rw http.ResponseWriter, r *http.Request) {
	conn, err := wsh.upgrader.Upgrade(rw, r, nil)
	if err != nil {
		wsh.log.Error("Unable to upgrade to WebSocket", "error", err)
		return
	}

	defer conn.Close()

	// subscribe to events
	subscriber := wsh.eventBus.Subscribe()
	defer wsh.eventBus.Unsubscribe(subscriber)

	// create a done channel to signal when the connection is closed
	done := make(chan struct{})

	// handle incoming messages
	go wsh.readPump(conn, done)

	// listen for events coming through the subscriber channel
	for {
		// select statement blocks until one of its cases is ready
		select {
		case event := <-subscriber:
			// determine the type of the event
			var message WebSocketMessage

			switch e := event.(type) {
			case events.PriceUpdate:
				message = WebSocketMessage{
					EventType: "price_update",
					Data:      e,
				}
			// add more cases for other event types if needed
			default:
				wsh.log.Warn("Unknown event type", "event", e)
				continue
			}

			payload, err := json.Marshal(message)
			if err != nil {
				wsh.log.Error("Error marshaling message", "error", err)
				continue
			}

			// send the message over the WebSocket connection to the client
			err = conn.WriteMessage(websocket.TextMessage, payload)
			if err != nil {
				wsh.log.Error("Error writing message to WebSocket", "error", err)
				// connection might be closed, exit the loop
				return
			}
		case <-done:
			// the connection has been closed
			wsh.log.Info("Connection has been closed by client")
			return
		}
	}
}

func (wsh *WebSocketHandler) readPump(conn *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				wsh.log.Error("Error reading message", "error", err)
			}
			break
		}
	}
}
