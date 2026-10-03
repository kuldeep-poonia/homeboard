package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Strict origin check can be enforced or customized per deployment
		return true
	},
}

type Event struct {
	Type    string      `json:"type"` // item.created, item.updated, item.deleted, device.revoked
	Payload interface{} `json:"payload"`
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	boardID  string
	deviceID string
	isTV     bool
}

type Hub struct {
	mu           sync.RWMutex
	boardClients map[string]map[*Client]bool
	register     chan *Client
	unregister   chan *Client
	broadcast    chan boardMessage
	maxPerBoard  int
}

type boardMessage struct {
	boardID string
	data    []byte
}

func NewHub(maxPerBoard int) *Hub {
	if maxPerBoard <= 0 {
		maxPerBoard = 15
	}
	return &Hub{
		boardClients: make(map[string]map[*Client]bool),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		broadcast:    make(chan boardMessage, 256),
		maxPerBoard:  maxPerBoard,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			clients, ok := h.boardClients[client.boardID]
			if !ok {
				clients = make(map[*Client]bool)
				h.boardClients[client.boardID] = clients
			}
			if len(clients) >= h.maxPerBoard {
				// Board limit reached: close new connection
				h.mu.Unlock()
				close(client.send)
				client.conn.Close()
				continue
			}
			clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.boardClients[client.boardID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.boardClients, client.boardID)
					}
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			if clients, ok := h.boardClients[msg.boardID]; ok {
				for client := range clients {
					select {
					case client.send <- msg.data:
					default:
						// Buffer full: drop slow client
						close(client.send)
						delete(clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastEvent(boardID string, eventType string, payload interface{}) {
	evt := Event{
		Type:    eventType,
		Payload: payload,
	}
	data, err := json.Marshal(evt)
	if err != nil {
		log.Printf("error marshaling websocket event: %v", err)
		return
	}
	h.broadcast <- boardMessage{boardID: boardID, data: data}
}

func (h *Hub) DisconnectDevice(boardID, deviceID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.boardClients[boardID]; ok {
		for client := range clients {
			if client.deviceID == deviceID {
				client.conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "device revoked"),
					time.Now().Add(time.Second),
				)
				delete(clients, client)
				close(client.send)
				client.conn.Close()
			}
		}
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		// Read messages; client-to-server WS messages are validated or discarded (we are mostly server-push)
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func UpgradeConnection(hub *Hub, w http.ResponseWriter, r *http.Request, boardID, deviceID string, isTV bool) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 64),
		boardID:  boardID,
		deviceID: deviceID,
		isTV:     isTV,
	}

	hub.register <- client

	go client.WritePump()
	go client.ReadPump()

	return nil
}
