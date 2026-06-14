package ws

import (
	"log"
	"net/http"
	"strconv"

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

		hub.register <- client

		go client.WritePump()
		go client.ReadPump()
	}
}
