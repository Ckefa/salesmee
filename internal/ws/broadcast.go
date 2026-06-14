package ws

import (
	"salesmee/internal/chatpb"
	"time"
)

func BroadcastNewMessage(hub *Hub, conversationID, senderID, senderType, messageID, content, mediaURL, mediaType, msgType string, dataJSON []byte, createdAt time.Time, bizID, clientID string) {
	frame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_NEW_MESSAGE,
		ConversationId: conversationID,
		SenderId:       senderID,
		SenderType:     senderType,
		Timestamp:      createdAt.UnixMilli(),
		Payload: &chatpb.WsFrame_NewMessage{
			NewMessage: &chatpb.NewMessage{
				Id:        messageID,
				Content:   content,
				Sender:    senderType,
				MediaUrl:  mediaURL,
				MediaType: mediaType,
				MsgType:   msgType,
				CreatedAt: createdAt.UnixMilli(),
				DataJson:  dataJSON,
			},
		},
	}
	hub.Broadcast("biz:"+bizID, frame, nil)
	hub.Broadcast("client:"+clientID, frame, nil)
}

func BroadcastReadReceipt(hub *Hub, conversationID, readerID, readerType, messageID, bizID, clientID string) {
	frame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_READ_RECEIPT,
		ConversationId: conversationID,
		SenderId:       readerID,
		SenderType:     readerType,
		Timestamp:      time.Now().UnixMilli(),
		Payload: &chatpb.WsFrame_ReadReceipt{
			ReadReceipt: &chatpb.ReadReceipt{
				MessageId:      messageID,
				ReaderId:       readerID,
				ReaderType:     readerType,
				ConversationId: conversationID,
			},
		},
	}
	hub.Broadcast("biz:"+bizID, frame, nil)
	if clientID != "" {
		hub.Broadcast("client:"+clientID, frame, nil)
	}
}

func BroadcastOrderUpdate(hub *Hub, orderID, status string, paidAmount, totalAmount float64, bizID, clientID string) {
	frame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_ORDER_UPDATE,
		ConversationId: "",
		SenderId:       "",
		SenderType:     "system",
		Timestamp:      time.Now().UnixMilli(),
		Payload: &chatpb.WsFrame_OrderUpdate{
			OrderUpdate: &chatpb.OrderUpdate{
				OrderId:     orderID,
				Status:      status,
				PaidAmount:  paidAmount,
				TotalAmount: totalAmount,
			},
		},
	}
	hub.Broadcast("biz:"+bizID, frame, nil)
	hub.Broadcast("client:"+clientID, frame, nil)
}

func BroadcastBookingUpdate(hub *Hub, bookingID, status string, paidAmount, totalAmount float64, bizID, clientID string) {
	frame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_BOOKING_UPDATE,
		ConversationId: "",
		SenderId:       "",
		SenderType:     "system",
		Timestamp:      time.Now().UnixMilli(),
		Payload: &chatpb.WsFrame_BookingUpdate{
			BookingUpdate: &chatpb.BookingUpdate{
				BookingId:   bookingID,
				Status:      status,
				PaidAmount:  paidAmount,
				TotalAmount: totalAmount,
			},
		},
	}
	hub.Broadcast("biz:"+bizID, frame, nil)
	hub.Broadcast("client:"+clientID, frame, nil)
}

func BroadcastPresenceUpdate(hub *Hub, clientID string, isOnline bool, lastSeen int64, bizID string) {
	frame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_PRESENCE_UPDATE,
		SenderId:       clientID,
		SenderType:     "client",
		Timestamp:      time.Now().UnixMilli(),
		Payload: &chatpb.WsFrame_Presence{
			Presence: &chatpb.PresenceUpdate{
				ClientId: clientID,
				IsOnline: isOnline,
				LastSeen: lastSeen,
			},
		},
	}
	hub.Broadcast("biz:"+bizID, frame, nil)
}

func BroadcastUnreadCount(hub *Hub, conversationID string, count int32, bizID string) {
	frame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_UNREAD_COUNT,
		ConversationId: conversationID,
		SenderId:       "",
		SenderType:     "system",
		Timestamp:      time.Now().UnixMilli(),
		Payload: &chatpb.WsFrame_UnreadCount{
			UnreadCount: &chatpb.UnreadCount{
				ConversationId: conversationID,
				Count:          count,
			},
		},
	}
	hub.Broadcast("biz:"+bizID, frame, nil)
}
