package handlers

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	projectClients   = make(map[string]map[*websocket.Conn]bool)
	projectClientsMu sync.RWMutex
)

func BroadCastRefresh(projectID string) {
	projectClientsMu.RLock()
	clients, exists := projectClients[projectID]
	if !exists || len(clients) == 0 {
		projectClientsMu.RUnlock()
		return
	}

	// Create a copy of the clients map to avoid holding the lock during message sending
	clientsCopy := make([]*websocket.Conn, 0, len(clients))
	for conn := range clients {
		clientsCopy = append(clientsCopy, conn)
	}
	projectClientsMu.RUnlock()

	// Send refresh message to all clients
	for _, conn := range clientsCopy {
		err := conn.WriteJSON(map[string]string{
			"type":       "refresh",
			"message":    "Dashboard data updated",
			"project_id": projectID,
		})

		if err != nil {
			// Remove failed connection
			projectClientsMu.Lock()
			if clients, exists := projectClients[projectID]; exists {
				delete(clients, conn)
				if len(clients) == 0 {
					delete(projectClients, projectID)
				}
			}
			projectClientsMu.Unlock()
			conn.Close()
		}
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
			return true
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// Register the connection to the project
	projectClientsMu.Lock()
	if projectClients[projectID] == nil {
		projectClients[projectID] = make(map[*websocket.Conn]bool)
	}
	projectClients[projectID][conn] = true
	projectClientsMu.Unlock()

	defer func() {
		projectClientsMu.Lock()
		if clients, exists := projectClients[projectID]; exists {
			delete(clients, conn)
			if len(clients) == 0 {
				delete(projectClients, projectID)
			}
		}
		projectClientsMu.Unlock()
		conn.Close()
	}()

	err = conn.WriteJSON(map[string]string{
		"type":       "connected",
		"message":    "WebSocket connection established",
		"project_id": projectID,
	})

	if err != nil {
		return
	}

	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
