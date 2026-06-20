package business

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"salesmee/internal/db"
	"salesmee/internal/models"
	"salesmee/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type orderTestFixture struct {
	handler *OrderHandler
	db      *gorm.DB
	biz     models.Business
	client  models.Client
	product models.Product
}

func newOrderTest(t *testing.T) *orderTestFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	d := testutil.SetupTestDB()

	oldDB := db.DB
	db.DB = d
	t.Cleanup(func() { db.DB = oldDB })

	biz := testutil.CreateBusiness(d, nil)
	client := testutil.CreateClient(d, biz.ID, nil)
	product := testutil.CreateProduct(d, biz.ID, nil)
	return &orderTestFixture{
		handler: &OrderHandler{dbProvider{db: d, hub: nil}},
		db:      d,
		biz:     biz,
		client:  client,
		product: product,
	}
}

func (f *orderTestFixture) newContext(t *testing.T, method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	return f.newContextWithTmpl(t, method, path, body, "")
}

func (f *orderTestFixture) newContextWithTmpl(t *testing.T, method, path string, body interface{}, tmplStr string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	if tmplStr != "" {
		tmpl := template.Must(template.New("").Parse(tmplStr))
		r.SetHTMLTemplate(tmpl)
	}

	c.Set("business_id", f.biz.ID)
	c.Set("auth_type", "owner")

	var req *http.Request
	if body != nil {
		var buf bytes.Buffer
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
		req = httptest.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req
	return c, w
}

func TestCreateOrder_Success(t *testing.T) {
	f := newOrderTest(t)

	body := map[string]interface{}{
		"client_id":  f.client.ID,
		"product_id": f.product.ID,
		"quantity":   2,
	}
	c, w := f.newContext(t, "POST", "/business/orders", body)
	f.handler.CreateOrder(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	var order models.Order
	require.NoError(t, f.db.First(&order).Error)
	assert.Equal(t, f.biz.ID, order.BusinessID)
	assert.Equal(t, f.client.ID, order.ClientID)
	assert.Equal(t, models.OrderPending, order.Status)
	assert.Equal(t, "business", order.Sender)
	expectedTotal := float64(2) * f.product.Price
	assert.Equal(t, expectedTotal, order.TotalAmount)

	var item models.OrderItem
	require.NoError(t, f.db.Where("order_id = ?", order.ID).First(&item).Error)
	assert.Equal(t, f.product.ID, item.ProductID)
	assert.Equal(t, 2, item.Quantity)
	assert.Equal(t, f.product.Price, item.UnitPrice)

	var product models.Product
	require.NoError(t, f.db.First(&product, f.product.ID).Error)
	assert.Equal(t, 98, product.Stock)

	var invLog models.InventoryLog
	require.NoError(t, f.db.Where("product_id = ?", f.product.ID).First(&invLog).Error)
	assert.Equal(t, "out", invLog.Type)
	assert.Equal(t, 2, invLog.Quantity)
}

func TestCreateOrder_MarkCompleted(t *testing.T) {
	f := newOrderTest(t)

	body := map[string]interface{}{
		"client_id":      f.client.ID,
		"product_id":     f.product.ID,
		"quantity":       1,
		"mark_completed": true,
	}
	c, w := f.newContext(t, "POST", "/business/orders", body)
	f.handler.CreateOrder(c)

	require.Equal(t, http.StatusOK, w.Code)

	var order models.Order
	require.NoError(t, f.db.First(&order).Error)
	assert.Equal(t, models.OrderFulfilled, order.Status)
	assert.Equal(t, order.TotalAmount, order.PaidAmount)

	var payment models.Payment
	require.NoError(t, f.db.Where("order_id = ?", order.ID).First(&payment).Error)
	assert.Equal(t, "cash", payment.Method)
	assert.Equal(t, models.OrderCompleted, payment.Status)
	assert.Equal(t, "Walk-in counter payment", payment.Reference)
	assert.Equal(t, order.TotalAmount, payment.Amount)
}

func TestCreateOrder_WalkinCustomer(t *testing.T) {
	f := newOrderTest(t)

	body := map[string]interface{}{
		"product_id":     f.product.ID,
		"quantity":       1,
		"customer_name":  "Walk-in John",
		"customer_email": "walkin@example.com",
		"customer_phone": "123-456-7890",
	}
	c, w := f.newContext(t, "POST", "/business/orders", body)
	f.handler.CreateOrder(c)

	require.Equal(t, http.StatusOK, w.Code)

	var newClient models.Client
	require.NoError(t, f.db.Where("name = ? AND email = ?", "Walk-in John", "walkin@example.com").First(&newClient).Error)
	assert.NotEqual(t, f.client.ID, newClient.ID)
	assert.Equal(t, f.biz.ID, *newClient.BusinessID)

	var order models.Order
	require.NoError(t, f.db.Where("client_id = ?", newClient.ID).First(&order).Error)
	assert.Equal(t, newClient.ID, order.ClientID)
}

func TestCreateOrder_NoAuth(t *testing.T) {
	f := newOrderTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"client_id":  f.client.ID,
		"product_id": f.product.ID,
		"quantity":   1,
	}
	var buf bytes.Buffer
	require.NoError(t, json.NewEncoder(&buf).Encode(body))
	req := httptest.NewRequest("POST", "/business/orders", &buf)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	f.handler.CreateOrder(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateOrder_InvalidJSON(t *testing.T) {
	f := newOrderTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("business_id", f.biz.ID)

	req := httptest.NewRequest("POST", "/business/orders", bytes.NewBufferString("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	f.handler.CreateOrder(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateOrder_InsufficientStock(t *testing.T) {
	f := newOrderTest(t)

	require.NoError(t, f.db.Model(&f.product).Update("stock", 1).Error)

	body := map[string]interface{}{
		"client_id":  f.client.ID,
		"product_id": f.product.ID,
		"quantity":   5,
	}
	c, w := f.newContext(t, "POST", "/business/orders", body)
	f.handler.CreateOrder(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "Insufficient stock")
}

func TestCreateOrder_MissingProductID(t *testing.T) {
	f := newOrderTest(t)

	body := map[string]interface{}{
		"client_id": f.client.ID,
		"product_id": 0,
		"quantity":   1,
	}
	c, w := f.newContext(t, "POST", "/business/orders", body)
	f.handler.CreateOrder(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSendOrderToClient(t *testing.T) {
	f := newOrderTest(t)

	order := testutil.CreateOrder(f.db, f.biz.ID, f.client.ID, nil)
	require.Equal(t, models.OrderDraft, order.Status)

	c, w := f.newContext(t, "POST", fmt.Sprintf("/business/orders/%d/send", order.ID), nil)
	c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", order.ID)}}
	f.handler.SendOrderToClient(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	var updated models.Order
	require.NoError(t, f.db.First(&updated, order.ID).Error)
	assert.Equal(t, models.OrderPending, updated.Status)
	assert.False(t, updated.Draft)
}

func TestSendOrderToClient_NotDraft(t *testing.T) {
	f := newOrderTest(t)

	order := testutil.CreateOrder(f.db, f.biz.ID, f.client.ID, map[string]interface{}{
		"Status": models.OrderPending,
	})

	c, w := f.newContext(t, "POST", fmt.Sprintf("/business/orders/%d/send", order.ID), nil)
	c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", order.ID)}}
	f.handler.SendOrderToClient(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSendOrderToClient_NotFound(t *testing.T) {
	f := newOrderTest(t)

	c, w := f.newContext(t, "POST", "/business/orders/99999/send", nil)
	c.Params = []gin.Param{{Key: "id", Value: "99999"}}
	f.handler.SendOrderToClient(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestConfirmOrderBusiness(t *testing.T) {
	f := newOrderTest(t)

	order := testutil.CreateOrder(f.db, f.biz.ID, f.client.ID, map[string]interface{}{
		"Status": models.OrderPending,
	})

	c, w := f.newContext(t, "POST", fmt.Sprintf("/business/orders/%d/confirm", order.ID), nil)
	c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", order.ID)}}
	f.handler.ConfirmOrderBusiness(c)

	require.Equal(t, http.StatusOK, w.Code)

	var updated models.Order
	require.NoError(t, f.db.First(&updated, order.ID).Error)
	assert.Equal(t, models.OrderConfirmed, updated.Status)
	assert.True(t, updated.ConfirmedByBusiness)
	assert.NotNil(t, updated.ConfirmedByBusinessAt)
}

func TestConfirmOrderBusiness_InvalidStatus(t *testing.T) {
	f := newOrderTest(t)

	order := testutil.CreateOrder(f.db, f.biz.ID, f.client.ID, map[string]interface{}{
		"Status": models.OrderDraft,
	})

	c, w := f.newContext(t, "POST", fmt.Sprintf("/business/orders/%d/confirm", order.ID), nil)
	c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", order.ID)}}
	f.handler.ConfirmOrderBusiness(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConfirmOrderBusiness_NotFound(t *testing.T) {
	f := newOrderTest(t)

	c, w := f.newContext(t, "POST", "/business/orders/99999/confirm", nil)
	c.Params = []gin.Param{{Key: "id", Value: "99999"}}
	f.handler.ConfirmOrderBusiness(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFulfillOrder(t *testing.T) {
	f := newOrderTest(t)

	order := testutil.CreateOrder(f.db, f.biz.ID, f.client.ID, map[string]interface{}{
		"Status": models.OrderConfirmed,
	})

	c, w := f.newContext(t, "POST", fmt.Sprintf("/business/orders/%d/fulfill", order.ID), nil)
	c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", order.ID)}}
	f.handler.FulfillOrder(c)

	require.Equal(t, http.StatusOK, w.Code)

	var updated models.Order
	require.NoError(t, f.db.First(&updated, order.ID).Error)
	assert.Equal(t, models.OrderFulfilled, updated.Status)
}

func TestFulfillOrder_NotConfirmed(t *testing.T) {
	f := newOrderTest(t)

	order := testutil.CreateOrder(f.db, f.biz.ID, f.client.ID, map[string]interface{}{
		"Status": models.OrderPending,
	})

	c, w := f.newContext(t, "POST", fmt.Sprintf("/business/orders/%d/fulfill", order.ID), nil)
	c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", order.ID)}}
	f.handler.FulfillOrder(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRejectOrder(t *testing.T) {
	f := newOrderTest(t)

	order := testutil.CreateOrder(f.db, f.biz.ID, f.client.ID, map[string]interface{}{
		"Status": models.OrderPending,
	})

	c, w := f.newContext(t, "POST", fmt.Sprintf("/business/orders/%d/reject", order.ID), nil)
	c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", order.ID)}}
	f.handler.RejectOrder(c)

	require.Equal(t, http.StatusOK, w.Code)

	var updated models.Order
	require.NoError(t, f.db.First(&updated, order.ID).Error)
	assert.Equal(t, models.OrderCancelled, updated.Status)
}

func TestRejectOrder_Confirmed(t *testing.T) {
	f := newOrderTest(t)

	order := testutil.CreateOrder(f.db, f.biz.ID, f.client.ID, map[string]interface{}{
		"Status": models.OrderConfirmed,
	})

	c, w := f.newContext(t, "POST", fmt.Sprintf("/business/orders/%d/reject", order.ID), nil)
	c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", order.ID)}}
	f.handler.RejectOrder(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMarkOrderAsPaid(t *testing.T) {
	f := newOrderTest(t)

	order := testutil.CreateOrder(f.db, f.biz.ID, f.client.ID, map[string]interface{}{
		"Status":      models.OrderConfirmed,
		"TotalAmount": 100.00,
	})

	c, w := f.newContext(t, "POST", fmt.Sprintf("/business/orders/%d/paid", order.ID), nil)
	c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", order.ID)}}
	f.handler.MarkOrderAsPaid(c)

	require.Equal(t, http.StatusOK, w.Code)

	var updated models.Order
	require.NoError(t, f.db.First(&updated, order.ID).Error)
	assert.Equal(t, updated.TotalAmount, updated.PaidAmount)

	var payment models.Payment
	require.NoError(t, f.db.Where("order_id = ?", order.ID).First(&payment).Error)
	assert.Equal(t, "cash", payment.Method)
	assert.Equal(t, models.OrderCompleted, payment.Status)
	assert.Equal(t, "quick-paid", payment.Reference)
	assert.Equal(t, updated.TotalAmount, payment.Amount)
}

func TestMarkOrderAsPaid_NotConfirmed(t *testing.T) {
	f := newOrderTest(t)

	order := testutil.CreateOrder(f.db, f.biz.ID, f.client.ID, map[string]interface{}{
		"Status": models.OrderPending,
	})

	c, w := f.newContext(t, "POST", fmt.Sprintf("/business/orders/%d/paid", order.ID), nil)
	c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", order.ID)}}
	f.handler.MarkOrderAsPaid(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMarkOrderAsPaid_NotFound(t *testing.T) {
	f := newOrderTest(t)

	c, w := f.newContext(t, "POST", "/business/orders/99999/paid", nil)
	c.Params = []gin.Param{{Key: "id", Value: "99999"}}
	f.handler.MarkOrderAsPaid(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetOrders(t *testing.T) {
	f := newOrderTest(t)

	testutil.CreateOrder(f.db, f.biz.ID, f.client.ID, map[string]interface{}{
		"Status": models.OrderPending,
	})
	testutil.CreateOrder(f.db, f.biz.ID, f.client.ID, map[string]interface{}{
		"Status": models.OrderDraft,
	})

	const tmpl = `{{define "orders.html"}}{{end}}{{define "dashboard/orders_content"}}{{end}}`
	c, w := f.newContextWithTmpl(t, "GET", "/business/orders", nil, tmpl)
	f.handler.GetOrders(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetOrderReceipt(t *testing.T) {
	f := newOrderTest(t)

	order := testutil.CreateOrder(f.db, f.biz.ID, f.client.ID, map[string]interface{}{
		"Status": models.OrderFulfilled,
	})

	const tmpl = `{{define "receipt_order.html"}}{{end}}`
	c, w := f.newContextWithTmpl(t, "GET", fmt.Sprintf("/business/orders/%d/receipt", order.ID), nil, tmpl)
	c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", order.ID)}}
	f.handler.GetOrderReceipt(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetOrderReceipt_NotCompleted(t *testing.T) {
	f := newOrderTest(t)

	order := testutil.CreateOrder(f.db, f.biz.ID, f.client.ID, map[string]interface{}{
		"Status": models.OrderPending,
	})

	const tmpl = `{{define "error.html"}}error_template{{end}}`
	c, w := f.newContextWithTmpl(t, "GET", fmt.Sprintf("/business/orders/%d/receipt", order.ID), nil, tmpl)
	c.Params = []gin.Param{{Key: "id", Value: fmt.Sprintf("%d", order.ID)}}
	f.handler.GetOrderReceipt(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
