package presentation

import (
	"net/http"
	"core-backend/config"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		cfg := config.GetAppConfig().WebSocket
		// Allow all origins for now, restrict in production
		return true
	},
}

func WebSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upgrade to websocket"})
		return
	}
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		// Echo message back
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}
