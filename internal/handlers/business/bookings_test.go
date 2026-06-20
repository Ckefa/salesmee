package business

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"salesmee/internal/db"
	"salesmee/internal/models"
	"salesmee/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type bookingTestFixture struct {
	handler *BookingHandler
	db      *gorm.DB
	biz     models.Business
	tmpl    *template.Template
}

func newBookingTest(t *testing.T) *bookingTestFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	testDB := testutil.SetupTestDB()
	biz := testutil.CreateBusiness(testDB, nil)
	oldDB := db.DB
	db.DB = testDB
	t.Cleanup(func() { db.DB = oldDB })
	tmpl := template.Must(template.New("").Parse(
		`{{define "login.html"}}ok{{end}}{{define "error.html"}}ok{{end}}{{define "receipt_booking.html"}}ok{{end}}`,
	))
	return &bookingTestFixture{
		handler: &BookingHandler{dbProvider{db: testDB, hub: nil}},
		db:      testDB,
		biz:     biz,
		tmpl:    tmpl,
	}
}

func (f *bookingTestFixture) newRequest(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func TestCreateBooking(t *testing.T) {
	f := newBookingTest(t)
	service := testutil.CreateService(f.db, f.biz.ID, nil)

	c, w := f.newRequest("POST", "/business/bookings", map[string]interface{}{
		"service_id":    service.ID,
		"booking_date":  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"customer_name": "Test Customer",
	})
	f.handler.CreateBooking(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))

	var booking models.Booking
	f.db.First(&booking)
	assert.Equal(t, models.BookingPending, booking.Status)
	assert.Equal(t, service.MaxPrice, booking.TotalAmount)

	var item models.BookingItem
	f.db.Where("booking_id = ?", booking.ID).First(&item)
	assert.Equal(t, service.ID, item.ServiceID)
}

func TestCreateBooking_MarkCompleted(t *testing.T) {
	f := newBookingTest(t)
	service := testutil.CreateService(f.db, f.biz.ID, nil)

	c, w := f.newRequest("POST", "/business/bookings", map[string]interface{}{
		"service_id":     service.ID,
		"booking_date":   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"customer_name":  "Test Customer",
		"customer_email": "customer@example.com",
		"mark_completed": true,
	})
	f.handler.CreateBooking(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var booking models.Booking
	f.db.First(&booking)
	assert.Equal(t, models.BookingCompleted, booking.Status)
	assert.Equal(t, booking.TotalAmount, booking.PaidAmount)

	var payment models.Payment
	f.db.Where("booking_id = ?", booking.ID).First(&payment)
	assert.Equal(t, booking.TotalAmount, payment.Amount)
	assert.Equal(t, "cash", payment.Method)
	assert.Equal(t, "Walk-in counter payment", payment.Reference)
}

func TestCreateBooking_MissingService(t *testing.T) {
	f := newBookingTest(t)

	c, w := f.newRequest("POST", "/business/bookings", map[string]interface{}{
		"booking_date":  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"customer_name": "Test Customer",
	})
	f.handler.CreateBooking(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConfirmBooking(t *testing.T) {
	f := newBookingTest(t)
	client := testutil.CreateClient(f.db, f.biz.ID, nil)
	testutil.CreateService(f.db, f.biz.ID, nil)
	booking := testutil.CreateBooking(f.db, f.biz.ID, client.ID, nil)

	c, w := f.newRequest("PUT", "/business/bookings/"+strconv.FormatUint(uint64(booking.ID), 10)+"/status", map[string]interface{}{
		"status": models.BookingClientConfirmed,
	})
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(booking.ID), 10)}}
	f.handler.UpdateBookingStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))

	var updated models.Booking
	f.db.First(&updated, booking.ID)
	assert.Equal(t, models.BookingClientConfirmed, updated.Status)
}

func TestCompleteBooking(t *testing.T) {
	f := newBookingTest(t)
	client := testutil.CreateClient(f.db, f.biz.ID, nil)
	testutil.CreateService(f.db, f.biz.ID, nil)
	booking := testutil.CreateBooking(f.db, f.biz.ID, client.ID, nil)

	f.db.Model(&booking).Update("Status", models.BookingClientConfirmed)

	c, w := f.newRequest("PUT", "/business/bookings/"+strconv.FormatUint(uint64(booking.ID), 10)+"/status", map[string]interface{}{
		"status": models.BookingCompleted,
	})
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(booking.ID), 10)}}
	f.handler.UpdateBookingStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var updated models.Booking
	f.db.First(&updated, booking.ID)
	assert.Equal(t, models.BookingCompleted, updated.Status)
}

func TestCancelBooking(t *testing.T) {
	f := newBookingTest(t)
	client := testutil.CreateClient(f.db, f.biz.ID, nil)
	testutil.CreateService(f.db, f.biz.ID, nil)
	booking := testutil.CreateBooking(f.db, f.biz.ID, client.ID, nil)

	c, w := f.newRequest("PUT", "/business/bookings/"+strconv.FormatUint(uint64(booking.ID), 10)+"/status", map[string]interface{}{
		"status": models.BookingCancelled,
	})
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(booking.ID), 10)}}
	f.handler.UpdateBookingStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var updated models.Booking
	f.db.First(&updated, booking.ID)
	assert.Equal(t, models.BookingCancelled, updated.Status)
}

func TestMarkBookingAsPaid(t *testing.T) {
	f := newBookingTest(t)
	client := testutil.CreateClient(f.db, f.biz.ID, nil)
	testutil.CreateService(f.db, f.biz.ID, nil)
	booking := testutil.CreateBooking(f.db, f.biz.ID, client.ID, nil)

	f.db.Model(&booking).Update("Status", models.BookingClientConfirmed)

	c, w := f.newRequest("PUT", "/business/bookings/"+strconv.FormatUint(uint64(booking.ID), 10)+"/paid", nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(booking.ID), 10)}}
	f.handler.MarkBookingAsPaid(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))

	var updated models.Booking
	f.db.First(&updated, booking.ID)
	assert.Equal(t, updated.TotalAmount, updated.PaidAmount)

	var payment models.Payment
	f.db.Where("booking_id = ?", booking.ID).First(&payment)
	assert.Equal(t, updated.TotalAmount, payment.Amount)
	assert.Equal(t, "cash", payment.Method)
}

func TestGetBookingReceipt(t *testing.T) {
	f := newBookingTest(t)
	client := testutil.CreateClient(f.db, f.biz.ID, nil)
	service := testutil.CreateService(f.db, f.biz.ID, nil)
	booking := testutil.CreateBooking(f.db, f.biz.ID, client.ID, map[string]interface{}{
		"Status": models.BookingCompleted,
	})

	f.db.Create(&models.BookingItem{
		BookingID:  booking.ID,
		ServiceID:  service.ID,
		UnitPrice:  service.MaxPrice,
		TotalPrice: service.MaxPrice,
	})

	f.db.Create(&models.Payment{
		BookingID: &booking.ID,
		ClientID:  client.ID,
		Amount:    booking.TotalAmount,
		Method:    "cash",
		Status:    models.PaymentCompleted,
	})

	c, w := f.newRequest("GET", "/business/bookings/"+strconv.FormatUint(uint64(booking.ID), 10)+"/receipt", nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(booking.ID), 10)}}
	f.handler.GetBookingReceipt(c)

	assert.Equal(t, http.StatusOK, w.Code)
}
