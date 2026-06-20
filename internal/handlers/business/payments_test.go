package business

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"salesmee/internal/db"
	"salesmee/internal/models"
	"salesmee/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type paymentTestFixture struct {
	handler        *PaymentHandler
	orderHandler   *OrderHandler
	bookingHandler *BookingHandler
	db             *gorm.DB
	biz            models.Business
	client         models.Client
	order          models.Order
	booking        models.Booking
	tmpl           *template.Template
}

func newPaymentTest(t *testing.T) *paymentTestFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	d := testutil.SetupTestDB()

	oldDB := db.DB
	db.DB = d
	t.Cleanup(func() { db.DB = oldDB })

	biz := testutil.CreateBusiness(d, nil)
	client := testutil.CreateClient(d, biz.ID, nil)

	order := testutil.CreateOrder(d, biz.ID, client.ID, map[string]interface{}{
		"Status":      models.OrderConfirmed,
		"TotalAmount": 100.0,
		"PaidAmount":  0.0,
	})

	booking := testutil.CreateBooking(d, biz.ID, client.ID, map[string]interface{}{
		"Status":      models.BookingClientConfirmed,
		"TotalAmount": 100.0,
		"PaidAmount":  0.0,
	})

	tmpl := template.Must(template.New("").Parse(
		`{{define "payments.html"}}ok{{end}}{{define "dashboard/payments_content"}}ok{{end}}{{define "payments_stats_grid"}}ok{{end}}`,
	))

	return &paymentTestFixture{
		handler:        &PaymentHandler{dbProvider{db: d, hub: nil}},
		orderHandler:   &OrderHandler{dbProvider{db: d, hub: nil}},
		bookingHandler: &BookingHandler{dbProvider{db: d, hub: nil}},
		db:             d,
		biz:            biz,
		client:         client,
		order:          order,
		booking:        booking,
		tmpl:           tmpl,
	}
}

func (f *paymentTestFixture) newContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.HTMLRender = &render.HTMLProduction{Template: f.tmpl}

	c.Set("business_id", f.biz.ID)
	c.Set("auth_type", "owner")

	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.Request = req
	return c, w
}

func TestGetPayments(t *testing.T) {
	f := newPaymentTest(t)

	f.db.Create(&models.Payment{
		OrderID:  &f.order.ID,
		ClientID: f.client.ID,
		Amount:   50.0,
		Method:   models.PayMethodMobileMoney,
		Status:   models.PaymentPending,
	})

	c, w := f.newContext("GET", "/business/payments", nil)
	f.handler.GetPayments(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetPaymentsStats(t *testing.T) {
	f := newPaymentTest(t)

	f.db.Create(&models.Payment{
		OrderID:  &f.order.ID,
		ClientID: f.client.ID,
		Amount:   50.0,
		Method:   models.PayMethodCash,
		Status:   models.PaymentCompleted,
	})

	c, w := f.newContext("GET", "/business/payments/stats", nil)
	f.handler.GetPaymentsStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetPaymentsStatsGrid(t *testing.T) {
	f := newPaymentTest(t)

	c, w := f.newContext("GET", "/business/payments/stats-grid", nil)
	f.handler.GetPaymentsStatsGrid(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCreatePaymentMethod(t *testing.T) {
	f := newPaymentTest(t)

	body := map[string]interface{}{
		"method_type": "bank_transfer",
		"label":       "Bank Transfer",
		"sort_order":  1,
	}
	c, w := f.newContext("POST", "/business/payment-methods", body)
	f.handler.CreatePaymentMethod(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	methodResp := resp["method"].(map[string]interface{})
	assert.Equal(t, "bank_transfer", methodResp["method_type"])
	assert.Equal(t, "Bank Transfer", methodResp["label"])

	var method models.PaymentMethod
	require.NoError(t, f.db.Where("business_id = ?", f.biz.ID).First(&method).Error)
	assert.Equal(t, "bank_transfer", method.MethodType)
	assert.Equal(t, "Bank Transfer", method.Label)
	assert.True(t, method.IsActive)
	assert.Equal(t, 1, method.SortOrder)
}

func TestGetPaymentMethods(t *testing.T) {
	f := newPaymentTest(t)

	f.db.Create(&models.PaymentMethod{
		BusinessID: f.biz.ID,
		MethodType: "cash",
		Label:      "Cash",
		IsActive:   true,
	})
	f.db.Create(&models.PaymentMethod{
		BusinessID: f.biz.ID,
		MethodType: "card",
		Label:      "Card",
		IsActive:   true,
	})

	c, w := f.newContext("GET", "/business/payment-methods", nil)
	f.handler.GetPaymentMethods(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	methods := resp["methods"].([]interface{})
	assert.Len(t, methods, 2)
}

func TestConfirmOrderPayment(t *testing.T) {
	f := newPaymentTest(t)

	payment := models.Payment{
		OrderID:   &f.order.ID,
		ClientID:  f.client.ID,
		Amount:    50.0,
		Method:    models.PayMethodMobileMoney,
		Status:    models.PaymentPending,
		Reference: "test-payment-ref",
	}
	require.NoError(t, f.db.Create(&payment).Error)

	c, w := f.newContext("POST", fmt.Sprintf("/business/orders/%d/payments/%d/confirm", f.order.ID, payment.ID), nil)
	c.Params = []gin.Param{
		{Key: "id", Value: fmt.Sprintf("%d", f.order.ID)},
		{Key: "payment_id", Value: fmt.Sprintf("%d", payment.ID)},
	}
	f.handler.ConfirmOrderPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
	assert.Equal(t, "Payment confirmed successfully", resp["message"])

	var updatedPayment models.Payment
	require.NoError(t, f.db.First(&updatedPayment, payment.ID).Error)
	assert.Equal(t, models.PaymentCompleted, updatedPayment.Status)

	var updatedOrder models.Order
	require.NoError(t, f.db.First(&updatedOrder, f.order.ID).Error)
	assert.Equal(t, 50.0, updatedOrder.PaidAmount)
}

func TestRejectOrderPayment(t *testing.T) {
	f := newPaymentTest(t)

	payment := models.Payment{
		OrderID:   &f.order.ID,
		ClientID:  f.client.ID,
		Amount:    50.0,
		Method:    models.PayMethodMobileMoney,
		Status:    models.PaymentPending,
		Reference: "test-payment-ref",
	}
	require.NoError(t, f.db.Create(&payment).Error)

	c, w := f.newContext("POST", fmt.Sprintf("/business/orders/%d/payments/%d/reject", f.order.ID, payment.ID), nil)
	c.Params = []gin.Param{
		{Key: "id", Value: fmt.Sprintf("%d", f.order.ID)},
		{Key: "payment_id", Value: fmt.Sprintf("%d", payment.ID)},
	}
	f.handler.RejectOrderPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
	assert.Equal(t, "Payment claim rejected", resp["message"])

	var updatedPayment models.Payment
	require.NoError(t, f.db.First(&updatedPayment, payment.ID).Error)
	assert.Equal(t, models.PaymentFailed, updatedPayment.Status)

	var order models.Order
	require.NoError(t, f.db.First(&order, f.order.ID).Error)
	assert.Equal(t, 0.0, order.PaidAmount)
}

func TestConfirmBookingPayment(t *testing.T) {
	f := newPaymentTest(t)

	payment := models.Payment{
		BookingID: &f.booking.ID,
		ClientID:  f.client.ID,
		Amount:    50.0,
		Method:    models.PayMethodMobileMoney,
		Status:    models.PaymentPending,
		Reference: "test-booking-payment-ref",
	}
	require.NoError(t, f.db.Create(&payment).Error)

	c, w := f.newContext("POST", fmt.Sprintf("/business/bookings/%d/payments/%d/confirm", f.booking.ID, payment.ID), nil)
	c.Params = []gin.Param{
		{Key: "id", Value: fmt.Sprintf("%d", f.booking.ID)},
		{Key: "payment_id", Value: fmt.Sprintf("%d", payment.ID)},
	}
	f.handler.ConfirmBookingPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	var updatedPayment models.Payment
	require.NoError(t, f.db.First(&updatedPayment, payment.ID).Error)
	assert.Equal(t, models.PaymentCompleted, updatedPayment.Status)

	var updatedBooking models.Booking
	require.NoError(t, f.db.First(&updatedBooking, f.booking.ID).Error)
	assert.Equal(t, 50.0, updatedBooking.PaidAmount)
}

func TestRejectBookingPayment(t *testing.T) {
	f := newPaymentTest(t)

	payment := models.Payment{
		BookingID: &f.booking.ID,
		ClientID:  f.client.ID,
		Amount:    50.0,
		Method:    models.PayMethodMobileMoney,
		Status:    models.PaymentPending,
		Reference: "test-booking-payment-ref",
	}
	require.NoError(t, f.db.Create(&payment).Error)

	c, w := f.newContext("POST", fmt.Sprintf("/business/bookings/%d/payments/%d/reject", f.booking.ID, payment.ID), nil)
	c.Params = []gin.Param{
		{Key: "id", Value: fmt.Sprintf("%d", f.booking.ID)},
		{Key: "payment_id", Value: fmt.Sprintf("%d", payment.ID)},
	}
	f.handler.RejectBookingPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	var updatedPayment models.Payment
	require.NoError(t, f.db.First(&updatedPayment, payment.ID).Error)
	assert.Equal(t, models.PaymentFailed, updatedPayment.Status)

	var booking models.Booking
	require.NoError(t, f.db.First(&booking, f.booking.ID).Error)
	assert.Equal(t, 0.0, booking.PaidAmount)
}

func TestPaymentMarkOrderAsPaid(t *testing.T) {
	f := newPaymentTest(t)

	c, w := f.newContext("POST", fmt.Sprintf("/business/orders/%d/paid", f.order.ID), nil)
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatUint(uint64(f.order.ID), 10)}}
	f.orderHandler.MarkOrderAsPaid(c)

	require.Equal(t, http.StatusOK, w.Code)

	var updated models.Order
	require.NoError(t, f.db.First(&updated, f.order.ID).Error)
	assert.Equal(t, updated.TotalAmount, updated.PaidAmount)

	var payment models.Payment
	require.NoError(t, f.db.Where("order_id = ?", f.order.ID).First(&payment).Error)
	assert.Equal(t, "cash", payment.Method)
	assert.Equal(t, models.PaymentCompleted, payment.Status)
	assert.Equal(t, updated.TotalAmount, payment.Amount)
	assert.Equal(t, "quick-paid", payment.Reference)
}
