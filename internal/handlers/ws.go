package handlers

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/monocle-dev/monocle/internal/types"
)

// wsClient wraps a websocket connection with a mutex to prevent concurrent writes
type wsClient struct {
	conn  *websocket.Conn
	mutex sync.Mutex
}

var (
	projectClients   = make(map[string]map[*wsClient]bool)
	projectClientsMu sync.RWMutex
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

func BroadCastRefresh(projectID string) {
	projectClientsMu.RLock()
	clients, exists := projectClients[projectID]
	if !exists || len(clients) == 0 {
		projectClientsMu.RUnlock()
		log.Printf("No clients connected for project %s", projectID)
		return
	}

	clientsCopy := make([]*wsClient, 0, len(clients))

	for client := range clients {
		clientsCopy = append(clientsCopy, client)
	}

	projectClientsMu.RUnlock()

	// Send refresh message to all clients
	for _, client := range clientsCopy {
		go func(client *wsClient) {
			// Lock this specific client connection to prevent concurrent writes
			client.mutex.Lock()
			defer client.mutex.Unlock()

			// Set write deadline
			if err := client.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("Failed to set write deadline for broadcast: %v", err)
				return
			}

			err := client.conn.WriteJSON(map[string]string{
				"type":       "refresh",
				"message":    "Dashboard data updated",
				"project_id": projectID,
			})

			if err != nil {
				log.Printf("Failed to broadcast refresh to client: %v", err)
				// Remove failed connection
				projectClientsMu.Lock()
				if clients, exists := projectClients[projectID]; exists {
					delete(clients, client)
					if len(clients) == 0 {
						delete(projectClients, projectID)
					}
				}
				projectClientsMu.Unlock()
				client.conn.Close()
			}
		}(client)
	}
}

func WebSocket(c *gin.Context) {
	projectID := c.Param("project_id")

	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			for _, allowed := range types.AllowedOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Set up connection parameters
	conn.SetReadLimit(maxMessageSize)
	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("Failed to set initial read deadline: %v", err)
		return
	}
	conn.SetPongHandler(func(string) error {
		if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			log.Printf("Failed to set read deadline in pong handler: %v", err)
		}
		return nil
	})

	// Register the connection to the project
	client := &wsClient{conn: conn}
	projectClientsMu.Lock()
	if projectClients[projectID] == nil {
		projectClients[projectID] = make(map[*wsClient]bool)
	}
	projectClients[projectID][client] = true
	projectClientsMu.Unlock()

	// Clean up when connection closes
	defer func() {
		projectClientsMu.Lock()

		if clients, exists := projectClients[projectID]; exists {
			delete(clients, client)

			if len(clients) == 0 {
				delete(projectClients, projectID)
			}
		}

		projectClientsMu.Unlock()
		conn.Close()

		log.Printf("WebSocket connection closed for project %s", projectID)
	}()

	// Send welcome message
	client.mutex.Lock()
	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		log.Printf("Failed to set write deadline for welcome message: %v", err)
		client.mutex.Unlock()
		return
	}

	err = conn.WriteJSON(map[string]string{
		"type":       "connected",
		"message":    "WebSocket connection established",
		"project_id": projectID,
	})
	client.mutex.Unlock()

	if err != nil {
		log.Printf("Failed to send welcome message: %v", err)
		return
	}

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				client.mutex.Lock()
				if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
					log.Printf("Failed to set write deadline for project %s: %v", projectID, err)
					client.mutex.Unlock()
					return
				}
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("Ping failed for project %s: %v", projectID, err)
					client.mutex.Unlock()
					return
				}
				client.mutex.Unlock()
			case <-done:
				return
			}
		}
	}()

	for {
		// Set read deadline for each message
		if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			log.Printf("Failed to set read deadline for project %s: %v", projectID, err)
			break
		}

		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for project %s: %v", projectID, err)
			}
			break
		}

		switch messageType {
		case websocket.TextMessage:
			log.Printf("Received message from client in project %s: %s", projectID, string(message))
		case websocket.PongMessage:
			log.Printf("Received pong from project %s", projectID)
		}
	}
}
