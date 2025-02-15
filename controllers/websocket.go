package controllers

import (
	"encoding/json"
	"net/http"

	"fyp/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)

// HandleWebSocket upgrades the connection to a WebSocket.
func HandleWebSocket(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	clients[conn] = true
}

// BroadcastUpdate sends sensor data updates to all WebSocket clients.
func BroadcastUpdate(data models.SensorData) {
	msg, _ := json.Marshal(data)
	for client := range clients {
		client.WriteMessage(websocket.TextMessage, msg)
	}
}

// BroadcastNotification sends an abnormal data notification to all WebSocket clients.
func BroadcastNotification(data models.SensorData) {
	notification := map[string]interface{}{
		"message": "Abnormal data detected!",
		"data":    data,
	}
	msg, _ := json.Marshal(notification)
	for client := range clients {
		client.WriteMessage(websocket.TextMessage, msg)
	}
}
