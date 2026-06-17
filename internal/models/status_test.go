package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderStatusConstants(t *testing.T) {
	assert.Equal(t, "draft", OrderDraft)
	assert.Equal(t, "pending", OrderPending)
	assert.Equal(t, "client_confirmed", OrderClientConfirmed)
	assert.Equal(t, "confirmed", OrderConfirmed)
	assert.Equal(t, "fulfilled", OrderFulfilled)
	assert.Equal(t, "completed", OrderCompleted)
	assert.Equal(t, "cancelled", OrderCancelled)
}

func TestBookingStatusConstants(t *testing.T) {
	assert.Equal(t, "pending", BookingPending)
	assert.Equal(t, "client_confirmed", BookingClientConfirmed)
	assert.Equal(t, "confirmed", BookingConfirmed)
	assert.Equal(t, "completed", BookingCompleted)
	assert.Equal(t, "cancelled", BookingCancelled)
}

func TestPaymentStatusConstants(t *testing.T) {
	assert.Equal(t, "pending", PaymentPending)
	assert.Equal(t, "completed", PaymentCompleted)
	assert.Equal(t, "failed", PaymentFailed)
}

func TestSenderConstants(t *testing.T) {
	assert.Equal(t, "business", SenderBusiness)
	assert.Equal(t, "client", SenderClient)
}

func TestPaymentMethodConstants(t *testing.T) {
	assert.Equal(t, "cash", PayMethodCash)
	assert.Equal(t, "card", PayMethodCard)
	assert.Equal(t, "bank_transfer", PayMethodBank)
	assert.Equal(t, "mobile_money", PayMethodMobileMoney)
}

func TestInventoryLogConstants(t *testing.T) {
	assert.Equal(t, "out", InvLogOut)
}
