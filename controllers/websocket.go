package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"fyp/config"
	"fyp/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	Conn   *websocket.Conn
	UserID uint
}

var clients = make(map[*websocket.Conn]Client)

func HandleWebSocket(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	userIDRaw, exists := c.Get("user_id")
	if !exists {
		conn.Close()
		return
	}

	var userID uint
	switch v := userIDRaw.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	case string:
		// Convert string to uint if necessary
		if id, err := strconv.ParseUint(v, 10, 32); err == nil {
			userID = uint(id)
		} else {
			conn.Close()
			return
		}
	default:
		conn.Close()
		return
	}

	clients[conn] = Client{
		Conn:   conn,
		UserID: userID,
	}
	defer func() {
		delete(clients, conn)
		conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// BroadcastUpdate sends sensor data updates to all WebSocket clients.
func BroadcastUpdate(data models.SensorData) {
	msg, _ := json.Marshal(data)
	for client := range clients {
		client.WriteMessage(websocket.TextMessage, msg)
	}
}
func BroadcastNotification(data models.SensorData) {
	for _, client := range clients {
		// Query only for this user's abnormal count
		var count int64
		config.DB.Model(&models.SensorData{}).
			Where("user_id = ? AND is_abnormal = ?", client.UserID, true).
			Count(&count)

		notification := map[string]interface{}{
			"message":        "Abnormal data detected!",
			"data":           data,
			"abnormal_count": count,
		}

		msg, _ := json.Marshal(notification)
		client.Conn.WriteMessage(websocket.TextMessage, msg)
	}
}
