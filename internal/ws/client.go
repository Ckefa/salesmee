package ws

import (
	"encoding/json"
	"log"
	"time"

	"salesmee/internal/chatpb"
	"salesmee/internal/services"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 65536
)

type ClientInfo struct {
	UserID     string
	UserType   string
	BusinessID string
	ClientID   string
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	rooms  []string
	info   ClientInfo
	claims *services.Claims
}

func NewClient(hub *Hub, conn *websocket.Conn, info ClientInfo, claims *services.Claims) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		info:   info,
		claims: claims,
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		msgType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("ws read error: %v", err)
			}
			break
		}

		switch msgType {
		case websocket.BinaryMessage:
			c.handleBinaryMessage(message)
		case websocket.TextMessage:
			c.handleTextMessage(message)
		}
	}
}

func (c *Client) handleBinaryMessage(data []byte) {
	var frame chatpb.WsFrame
	if err := proto.Unmarshal(data, &frame); err != nil {
		log.Printf("ws binary unmarshal error: %v", err)
		return
	}
	c.processFrame(&frame)
}

type incomingFrame struct {
	EventType      int             `json:"event_type"`
	ConversationID string          `json:"conversation_id"`
	SenderID       string          `json:"sender_id"`
	SenderType     string          `json:"sender_type"`
	Timestamp      int64           `json:"timestamp"`
	Typing         *incomingTyping `json:"typing,omitempty"`
}

type incomingTyping struct {
	UserID         string `json:"user_id"`
	UserType       string `json:"user_type"`
	ConversationID string `json:"conversation_id"`
	ClientID       string `json:"client_id"`
	BusinessID     string `json:"business_id"`
}

func (c *Client) handleTextMessage(data []byte) {
	var inc incomingFrame
	if err := json.Unmarshal(data, &inc); err != nil {
		log.Printf("ws json unmarshal error: %v", err)
		return
	}

	frame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType(inc.EventType),
		ConversationId: inc.ConversationID,
		SenderId:       inc.SenderID,
		SenderType:     inc.SenderType,
		Timestamp:      inc.Timestamp,
	}

	switch chatpb.WsEventType(inc.EventType) {
	case chatpb.WsEventType_PING:
	case chatpb.WsEventType_TYPING_START, chatpb.WsEventType_TYPING_STOP:
		if inc.Typing == nil {
			return
		}
		frame.Payload = &chatpb.WsFrame_Typing{
			Typing: &chatpb.TypingIndicator{
				UserId:         inc.Typing.UserID,
				UserType:       inc.Typing.UserType,
				ConversationId: inc.Typing.ConversationID,
				ClientId:       inc.Typing.ClientID,
				BusinessId:     inc.Typing.BusinessID,
			},
		}
	default:
		log.Printf("ws unknown event type from client: %d", inc.EventType)
		return
	}

	c.processFrame(frame)
}

func (c *Client) processFrame(frame *chatpb.WsFrame) {
	switch frame.GetEventType() {
	case chatpb.WsEventType_PING:
		pong := &chatpb.WsFrame{
			EventType: chatpb.WsEventType_PONG,
		}
		data, _ := proto.Marshal(pong)
		select {
		case c.send <- data:
		default:
		}

	case chatpb.WsEventType_TYPING_START, chatpb.WsEventType_TYPING_STOP:
		if typing := frame.GetTyping(); typing != nil {
			if typing.GetUserType() == "client" {
				c.hub.Broadcast("biz:"+typing.GetBusinessId(), frame, c)
			} else {
				c.hub.Broadcast("client:"+typing.GetClientId(), frame, c)
			}
		}

	default:
		log.Printf("ws unhandled event type: %v", frame.GetEventType())
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
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			var frame chatpb.WsFrame
			if err := proto.Unmarshal(message, &frame); err != nil {
				continue
			}

			jsonData, err := json.Marshal(jsonFromProto(&frame))
			if err != nil {
				continue
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
				log.Printf("ws write error: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			sendPing(c.send)
		}
	}
}

func sendPing(send chan<- []byte) {
	pong := &chatpb.WsFrame{
		EventType: chatpb.WsEventType_PING,
	}
	data, _ := proto.Marshal(pong)
	select {
	case send <- data:
	default:
	}
}

type jsonFrame struct {
	EventType      int                     `json:"event_type"`
	ConversationID string                  `json:"conversation_id,omitempty"`
	SenderID       string                  `json:"sender_id,omitempty"`
	SenderType     string                  `json:"sender_type,omitempty"`
	Timestamp      int64                   `json:"timestamp,omitempty"`
	NewMessage     *jsonNewMessage         `json:"new_message,omitempty"`
	ReadReceipt    *jsonReadReceipt        `json:"read_receipt,omitempty"`
	Typing         *jsonTyping             `json:"typing,omitempty"`
	OrderUpdate    *jsonOrderUpdate        `json:"order_update,omitempty"`
	BookingUpdate  *jsonBookingUpdate      `json:"booking_update,omitempty"`
	UnreadCount    *jsonUnreadCount        `json:"unread_count,omitempty"`
}

type jsonNewMessage struct {
	ID        string          `json:"id"`
	Content   string          `json:"content,omitempty"`
	Sender    string          `json:"sender,omitempty"`
	MediaURL  string          `json:"media_url,omitempty"`
	MediaType string          `json:"media_type,omitempty"`
	MsgType   string          `json:"msg_type,omitempty"`
	CreatedAt int64           `json:"created_at,omitempty"`
	DataJSON  json.RawMessage `json:"data_json,omitempty"`
}

type jsonReadReceipt struct {
	MessageID      string `json:"message_id"`
	ReaderID       string `json:"reader_id"`
	ReaderType     string `json:"reader_type"`
	ConversationID string `json:"conversation_id"`
}

type jsonTyping struct {
	UserID         string `json:"user_id"`
	UserType       string `json:"user_type"`
	ConversationID string `json:"conversation_id"`
	ClientID       string `json:"client_id"`
	BusinessID     string `json:"business_id"`
}

type jsonOrderUpdate struct {
	OrderID     string  `json:"order_id"`
	Status      string  `json:"status"`
	PaidAmount  float64 `json:"paid_amount"`
	TotalAmount float64 `json:"total_amount"`
}

type jsonBookingUpdate struct {
	BookingID   string  `json:"booking_id"`
	Status      string  `json:"status"`
	PaidAmount  float64 `json:"paid_amount"`
	TotalAmount float64 `json:"total_amount"`
}

type jsonUnreadCount struct {
	ConversationID string `json:"conversation_id"`
	Count          int32  `json:"count"`
}

func jsonFromProto(frame *chatpb.WsFrame) *jsonFrame {
	jf := &jsonFrame{
		EventType:      int(frame.GetEventType()),
		ConversationID: frame.GetConversationId(),
		SenderID:       frame.GetSenderId(),
		SenderType:     frame.GetSenderType(),
		Timestamp:      frame.GetTimestamp(),
	}

	switch p := frame.Payload.(type) {
	case *chatpb.WsFrame_NewMessage:
		m := p.NewMessage
		var dj json.RawMessage
		if len(m.GetDataJson()) > 0 {
			dj = json.RawMessage(m.GetDataJson())
		}
		jf.NewMessage = &jsonNewMessage{
			ID:        m.GetId(),
			Content:   m.GetContent(),
			Sender:    m.GetSender(),
			MediaURL:  m.GetMediaUrl(),
			MediaType: m.GetMediaType(),
			MsgType:   m.GetMsgType(),
			CreatedAt: m.GetCreatedAt(),
			DataJSON:  dj,
		}
	case *chatpb.WsFrame_ReadReceipt:
		r := p.ReadReceipt
		jf.ReadReceipt = &jsonReadReceipt{
			MessageID:      r.GetMessageId(),
			ReaderID:       r.GetReaderId(),
			ReaderType:     r.GetReaderType(),
			ConversationID: r.GetConversationId(),
		}
	case *chatpb.WsFrame_Typing:
		t := p.Typing
		jf.Typing = &jsonTyping{
			UserID:         t.GetUserId(),
			UserType:       t.GetUserType(),
			ConversationID: t.GetConversationId(),
			ClientID:       t.GetClientId(),
			BusinessID:     t.GetBusinessId(),
		}
	case *chatpb.WsFrame_OrderUpdate:
		o := p.OrderUpdate
		jf.OrderUpdate = &jsonOrderUpdate{
			OrderID:     o.GetOrderId(),
			Status:      o.GetStatus(),
			PaidAmount:  o.GetPaidAmount(),
			TotalAmount: o.GetTotalAmount(),
		}
	case *chatpb.WsFrame_BookingUpdate:
		b := p.BookingUpdate
		jf.BookingUpdate = &jsonBookingUpdate{
			BookingID:   b.GetBookingId(),
			Status:      b.GetStatus(),
			PaidAmount:  b.GetPaidAmount(),
			TotalAmount: b.GetTotalAmount(),
		}
	case *chatpb.WsFrame_UnreadCount:
		u := p.UnreadCount
		jf.UnreadCount = &jsonUnreadCount{
			ConversationID: u.GetConversationId(),
			Count:          u.GetCount(),
		}
	}

	return jf
}
