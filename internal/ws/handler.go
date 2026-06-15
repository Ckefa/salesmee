package ws

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"salesmee/internal/db"
	"salesmee/internal/models"
	"salesmee/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ServeBusinessWS(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims, err := services.ValidateToken(token)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		bizID := claims.UserID
		if bid := c.Query("business_id"); bid != "" {
			if id, err := strconv.ParseUint(bid, 10, 64); err == nil {
				bizID = uint(id)
			}
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("ws upgrade error: %v", err)
			return
		}

		info := ClientInfo{
			UserID:     strconv.Itoa(int(claims.UserID)),
			UserType:   "business",
			BusinessID: strconv.Itoa(int(bizID)),
			ClientID:   "",
		}

		client := NewClient(hub, conn, info, claims)
		client.rooms = []string{"biz:" + info.BusinessID}

		hub.register <- client

		go client.WritePump()
		go client.ReadPump()
	}
}

func getClientBizIDs(clientID uint) []string {
	var conversations []models.Conversation
	db.DB.Where("client_id = ?", clientID).Find(&conversations)
	ids := make([]string, 0, len(conversations))
	for _, conv := range conversations {
		ids = append(ids, strconv.Itoa(int(conv.BusinessID)))
	}
	return ids
}

func broadcastClientPresence(hub *Hub, clientID string, isOnline bool, bizIDs []string) {
	now := time.Now().UnixMilli()
	for _, bizID := range bizIDs {
		BroadcastPresenceUpdate(hub, clientID, isOnline, now, bizID)
	}
}

func ServeClientWS(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims, err := services.ValidateToken(token)
		if err != nil || claims.Subject != "client" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		bizID := c.Query("business_id")

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("ws upgrade error: %v", err)
			return
		}

		info := ClientInfo{
			UserID:     strconv.Itoa(int(claims.UserID)),
			UserType:   "client",
			BusinessID: bizID,
			ClientID:   strconv.Itoa(int(claims.UserID)),
		}

		client := NewClient(hub, conn, info, claims)
		client.rooms = []string{"client:" + info.ClientID}

		// Determine which business rooms to notify for presence
		var presenceBizIDs []string
		if bizID != "" {
			presenceBizIDs = []string{bizID}
		} else {
			presenceBizIDs = getClientBizIDs(claims.UserID)
		}

		// Mark online and broadcast presence
		now := time.Now()
		db.DB.Model(&models.Client{}).Where("id = ?", claims.UserID).Updates(map[string]interface{}{
			"is_online":    true,
			"last_seen_at": &now,
		})
		broadcastClientPresence(hub, info.ClientID, true, presenceBizIDs)

		// On disconnect, mark offline and broadcast to all connected businesses
		client.onDisconnect = func() {
			now := time.Now()
			db.DB.Model(&models.Client{}).Where("id = ?", claims.UserID).Updates(map[string]interface{}{
				"is_online":    false,
				"last_seen_at": &now,
			})
			// Query fresh business IDs to ensure all connected businesses get the offline event
			freshBizIDs := getClientBizIDs(claims.UserID)
			broadcastClientPresence(hub, info.ClientID, false, freshBizIDs)
		}

		hub.register <- client

		go client.WritePump()
		go client.ReadPump()
	}
}
