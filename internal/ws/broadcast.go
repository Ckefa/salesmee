package ws

import (
	"encoding/json"
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
	BroadcastOrderUpdateFull(hub, orderID, status, paidAmount, totalAmount, 0, false, 0, "", "", bizID, clientID)
}

func BroadcastOrderUpdateFull(hub *Hub, orderID, status string, paidAmount, totalAmount, pendingAmount float64, hasReview bool, reviewRating int32, bizCardHTML, clientCardHTML, bizID, clientID string) {
	bizFrame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_ORDER_UPDATE,
		ConversationId: "",
		SenderId:       "",
		SenderType:     "system",
		Timestamp:      time.Now().UnixMilli(),
		Payload: &chatpb.WsFrame_OrderUpdate{
			OrderUpdate: &chatpb.OrderUpdate{
				OrderId:       orderID,
				Status:        status,
				PaidAmount:    paidAmount,
				TotalAmount:   totalAmount,
				PendingAmount: pendingAmount,
				HasReview:     hasReview,
				ReviewRating:  reviewRating,
				CardHtml:      bizCardHTML,
			},
		},
	}
	clientFrame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_ORDER_UPDATE,
		ConversationId: "",
		SenderId:       "",
		SenderType:     "system",
		Timestamp:      time.Now().UnixMilli(),
		Payload: &chatpb.WsFrame_OrderUpdate{
			OrderUpdate: &chatpb.OrderUpdate{
				OrderId:       orderID,
				Status:        status,
				PaidAmount:    paidAmount,
				TotalAmount:   totalAmount,
				PendingAmount: pendingAmount,
				HasReview:     hasReview,
				ReviewRating:  reviewRating,
				CardHtml:      clientCardHTML,
			},
		},
	}
	hub.Broadcast("biz:"+bizID, bizFrame, nil)
	hub.Broadcast("client:"+clientID, clientFrame, nil)
}

func BroadcastBookingUpdate(hub *Hub, bookingID, status string, paidAmount, totalAmount float64, bizID, clientID string) {
	BroadcastBookingUpdateFull(hub, bookingID, status, paidAmount, totalAmount, 0, false, 0, "", "", bizID, clientID)
}

func BroadcastBookingUpdateFull(hub *Hub, bookingID, status string, paidAmount, totalAmount, pendingAmount float64, hasReview bool, reviewRating int32, bizCardHTML, clientCardHTML, bizID, clientID string) {
	bizFrame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_BOOKING_UPDATE,
		ConversationId: "",
		SenderId:       "",
		SenderType:     "system",
		Timestamp:      time.Now().UnixMilli(),
		Payload: &chatpb.WsFrame_BookingUpdate{
			BookingUpdate: &chatpb.BookingUpdate{
				BookingId:     bookingID,
				Status:        status,
				PaidAmount:    paidAmount,
				TotalAmount:   totalAmount,
				PendingAmount: pendingAmount,
				HasReview:     hasReview,
				ReviewRating:  reviewRating,
				CardHtml:      bizCardHTML,
			},
		},
	}
	clientFrame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_BOOKING_UPDATE,
		ConversationId: "",
		SenderId:       "",
		SenderType:     "system",
		Timestamp:      time.Now().UnixMilli(),
		Payload: &chatpb.WsFrame_BookingUpdate{
			BookingUpdate: &chatpb.BookingUpdate{
				BookingId:     bookingID,
				Status:        status,
				PaidAmount:    paidAmount,
				TotalAmount:   totalAmount,
				PendingAmount: pendingAmount,
				HasReview:     hasReview,
				ReviewRating:  reviewRating,
				CardHtml:      clientCardHTML,
			},
		},
	}
	hub.Broadcast("biz:"+bizID, bizFrame, nil)
	hub.Broadcast("client:"+clientID, clientFrame, nil)
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

func BroadcastConversationUpdate(hub *Hub, conversationID, bizCardHTML, clientCardHTML, bizID, clientID string) {
	if bizCardHTML != "" {
		bizFrame := &chatpb.WsFrame{
			EventType:      chatpb.WsEventType_CONVERSATION_UPDATE,
			ConversationId: conversationID,
			SenderId:       "",
			SenderType:     "system",
			Timestamp:      time.Now().UnixMilli(),
			Payload: &chatpb.WsFrame_ConversationUpdate{
				ConversationUpdate: &chatpb.ConversationUpdate{
					ConversationId: conversationID,
					BizCardHtml:    bizCardHTML,
				},
			},
		}
		hub.Broadcast("biz:"+bizID, bizFrame, nil)
	}
	if clientCardHTML != "" {
		clientFrame := &chatpb.WsFrame{
			EventType:      chatpb.WsEventType_CONVERSATION_UPDATE,
			ConversationId: conversationID,
			SenderId:       "",
			SenderType:     "system",
			Timestamp:      time.Now().UnixMilli(),
			Payload: &chatpb.WsFrame_ConversationUpdate{
				ConversationUpdate: &chatpb.ConversationUpdate{
					ConversationId:   conversationID,
					ClientCardHtml:   clientCardHTML,
				},
			},
		}
		hub.Broadcast("client:"+clientID, clientFrame, nil)
	}
}

func BroadcastConversationRemovedToBiz(hub *Hub, conversationID, bizID, clientID string) {
	bizFrame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_CONVERSATION_UPDATE,
		ConversationId: conversationID,
		SenderId:       clientID,
		SenderType:     "system",
		Timestamp:      time.Now().UnixMilli(),
		Payload: &chatpb.WsFrame_ConversationUpdate{
			ConversationUpdate: &chatpb.ConversationUpdate{
				ConversationId: conversationID,
			},
		},
	}
	hub.Broadcast("biz:"+bizID, bizFrame, nil)
}

func BroadcastConversationRemovedToClient(hub *Hub, conversationID, bizID, clientID string) {
	clientFrame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_CONVERSATION_UPDATE,
		ConversationId: conversationID,
		SenderId:       bizID,
		SenderType:     "system",
		Timestamp:      time.Now().UnixMilli(),
		Payload: &chatpb.WsFrame_ConversationUpdate{
			ConversationUpdate: &chatpb.ConversationUpdate{
				ConversationId: conversationID,
			},
		},
	}
	hub.Broadcast("client:"+clientID, clientFrame, nil)
}

func BroadcastBusinessPresenceUpdate(hub *Hub, businessID string, isOnline bool, clientIDs []string) {
	frame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_PRESENCE_UPDATE,
		SenderId:       businessID,
		SenderType:     "business",
		Timestamp:      time.Now().UnixMilli(),
		Payload: &chatpb.WsFrame_Presence{
			Presence: &chatpb.PresenceUpdate{
				BusinessId: businessID,
				IsOnline:   isOnline,
				LastSeen:   time.Now().UnixMilli(),
			},
		},
	}
	for _, clientID := range clientIDs {
		hub.Broadcast("client:"+clientID, frame, nil)
	}
}

func BroadcastUnreadCount(hub *Hub, conversationID string, count int32, roomID, roomPrefix string) {
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
	hub.Broadcast(roomPrefix+":"+roomID, frame, nil)
}

func BroadcastOnboardingUpdate(hub *Hub, bizID string, step, totalSteps int, completed bool) {
	data, _ := json.Marshal(map[string]interface{}{
		"step":        step,
		"total_steps": totalSteps,
		"completed":   completed,
	})
	frame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_ONBOARDING_UPDATE,
		ConversationId: bizID,
		SenderId:       string(data),
		SenderType:     "system",
		Timestamp:      time.Now().UnixMilli(),
	}
	hub.Broadcast("biz:"+bizID, frame, nil)
}

func BroadcastPendingCount(hub *Hub, bizID string, orderCount, bookingCount, notifCount int) {
	data, _ := json.Marshal(map[string]int{
		"order_count":   orderCount,
		"booking_count": bookingCount,
		"notif_count":   notifCount,
	})
	frame := &chatpb.WsFrame{
		EventType:      chatpb.WsEventType_PENDING_COUNT,
		ConversationId: bizID,
		SenderId:       string(data),
		SenderType:     "system",
		Timestamp:      time.Now().UnixMilli(),
	}
	hub.Broadcast("biz:"+bizID, frame, nil)
}
