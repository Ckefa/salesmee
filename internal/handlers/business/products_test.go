package business

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"salesmee/internal/models"
	"salesmee/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type productTestFixture struct {
	handler *ProductHandler
	db      *gorm.DB
	biz     models.Business
	tmpl    *template.Template
}

func newProductTest(t *testing.T) *productTestFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.SetupTestDB()
	biz := testutil.CreateBusiness(db, nil)
	tmpl := template.Must(template.New("").Parse(
		`{{define "login.html"}}ok{{end}}{{define "products.html"}}ok{{end}}{{define "dashboard/products_content"}}ok{{end}}`,
	))
	return &productTestFixture{
		handler: &ProductHandler{db: db, hub: nil},
		db:      db,
		biz:     biz,
		tmpl:    tmpl,
	}
}

func (f *productTestFixture) newRequest(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func TestCreateProduct(t *testing.T) {
	f := newProductTest(t)
	c, w := f.newRequest("POST", "/business/products", map[string]interface{}{
		"name":  "New Product",
		"price": 19.99,
		"stock": 50,
	})
	f.handler.CreateProduct(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))

	var product models.Product
	f.db.First(&product)
	assert.Equal(t, "New Product", product.Name)
	assert.Equal(t, 19.99, product.Price)
	assert.Equal(t, 50, product.Stock)
}

func TestCreateProduct_MissingName(t *testing.T) {
	f := newProductTest(t)
	c, w := f.newRequest("POST", "/business/products", map[string]interface{}{
		"price": 19.99,
		"stock": 50,
	})
	f.handler.CreateProduct(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))

	var product models.Product
	f.db.First(&product)
	assert.Equal(t, "", product.Name)
}

func TestGetProducts(t *testing.T) {
	f := newProductTest(t)
	testutil.CreateProduct(f.db, f.biz.ID, nil)
	testutil.CreateProduct(f.db, f.biz.ID, map[string]interface{}{
		"Name": "Second Product",
	})

	c, w := f.newRequest("GET", "/business/products", nil)
	f.handler.GetProducts(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateProduct(t *testing.T) {
	f := newProductTest(t)
	product := testutil.CreateProduct(f.db, f.biz.ID, nil)

	c, w := f.newRequest("PUT", "/business/products/"+strconv.FormatUint(uint64(product.ID), 10), map[string]interface{}{
		"name":  "Updated Name",
		"price": 39.99,
	})
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(product.ID), 10)}}
	f.handler.UpdateProduct(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))

	var updated models.Product
	f.db.First(&updated, product.ID)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, 39.99, updated.Price)
}

func TestDeleteProduct(t *testing.T) {
	f := newProductTest(t)
	product := testutil.CreateProduct(f.db, f.biz.ID, nil)

	c, w := f.newRequest("DELETE", "/business/products/"+strconv.FormatUint(uint64(product.ID), 10), nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(product.ID), 10)}}
	f.handler.DeleteProduct(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))

	var deleted models.Product
	err := f.db.Unscoped().First(&deleted, product.ID).Error
	require.NoError(t, err)
	assert.NotNil(t, deleted.DeletedAt)
	assert.True(t, deleted.DeletedAt.Valid)
}

func TestGetProduct(t *testing.T) {
	f := newProductTest(t)
	product := testutil.CreateProduct(f.db, f.biz.ID, nil)

	c, w := f.newRequest("GET", "/business/products/"+strconv.FormatUint(uint64(product.ID), 10), nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(product.ID), 10)}}
	f.handler.GetProduct(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))

	respProduct := resp["product"].(map[string]interface{})
	assert.Equal(t, "Test Product", respProduct["name"])
	assert.Equal(t, 29.99, respProduct["price"])
}

func TestCreateProduct_NoAuth(t *testing.T) {
	f := newProductTest(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("POST", "/business/products", nil)
	f.handler.CreateProduct(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "not authenticated")
}
