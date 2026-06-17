package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"salesmee/internal/db"
	"salesmee/internal/services"
	"salesmee/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/stretchr/testify/assert"
)

type noopRender struct{}

func (noopRender) Render(w http.ResponseWriter) error {
	return nil
}

func (noopRender) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
}

type noopHTMLRender struct{}

func (noopHTMLRender) Instance(name string, data any) render.Render {
	return noopRender{}
}

func setupAuthEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	origDB := db.DB
	testDB := testutil.SetupTestDB()
	db.DB = testDB
	t.Cleanup(func() { db.DB = origDB })

	eng := gin.New()
	eng.HTMLRender = noopHTMLRender{}
	return eng
}

func newFormRequest(method, path string, data map[string]string) *http.Request {
	form := urlValues{}
	for k, v := range data {
		form.Set(k, v)
	}
	body := form.Encode()
	req, _ := http.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

type urlValues map[string][]string

func (uv urlValues) Set(key, value string) {
	uv[key] = []string{value}
}

func (uv urlValues) Encode() string {
	var parts []string
	for k, vals := range uv {
		for _, v := range vals {
			parts = append(parts, k+"="+v)
		}
	}
	return strings.Join(parts, "&")
}

func TestShowLogin(t *testing.T) {
	eng := setupAuthEngine(t)
	eng.GET("/business/login", ShowLogin)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/business/login", nil)
	eng.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestShowRegisterStep1(t *testing.T) {
	eng := setupAuthEngine(t)
	eng.GET("/business/register", ShowRegisterStep1)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/business/register", nil)
	eng.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRegisterStep1_MissingFields(t *testing.T) {
	eng := setupAuthEngine(t)
	eng.POST("/business/register", RegisterStep1)

	w := httptest.NewRecorder()
	req := newFormRequest("POST", "/business/register", map[string]string{
		"email": "",
		"name":  "",
	})
	eng.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRegisterStep1_DuplicateEmail(t *testing.T) {
	eng := setupAuthEngine(t)
	eng.POST("/business/register", RegisterStep1)

	testutil.CreateBusiness(db.DB, map[string]interface{}{
		"Email": "dupe@example.com",
		"Slug":  "existing-biz-1",
	})

	w := httptest.NewRecorder()
	req := newFormRequest("POST", "/business/register", map[string]string{
		"email":    "dupe@example.com",
		"password": "password123",
		"name":     "Test Business",
		"username": "testbiz",
	})
	eng.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRegisterStep1_ValidSubmission(t *testing.T) {
	eng := setupAuthEngine(t)
	eng.POST("/business/register", RegisterStep1)

	w := httptest.NewRecorder()
	req := newFormRequest("POST", "/business/register", map[string]string{
		"email":    "newbiz@example.com",
		"password": "password123",
		"name":     "New Business",
		"username": "newbiz",
	})
	eng.ServeHTTP(w, req)

	// RegisterStep1 saves to RegStore (in-memory) and redirects to step 2
	assert.Equal(t, http.StatusFound, w.Code)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	eng := setupAuthEngine(t)
	eng.POST("/business/login", Login)

	w := httptest.NewRecorder()
	req := newFormRequest("POST", "/business/login", map[string]string{
		"email":    "nonexistent@example.com",
		"password": "wrongpassword",
	})
	eng.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_WrongPassword(t *testing.T) {
	eng := setupAuthEngine(t)
	eng.POST("/business/login", Login)

	hash := services.Hash("correctpassword")
	testutil.CreateBusiness(db.DB, map[string]interface{}{
		"Email":    "valid@example.com",
		"Slug":     "valid-biz-2",
		"Password": &hash,
	})

	w := httptest.NewRecorder()
	req := newFormRequest("POST", "/business/login", map[string]string{
		"email":    "valid@example.com",
		"password": "wrongpassword",
	})
	eng.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_Success(t *testing.T) {
	eng := setupAuthEngine(t)
	eng.POST("/business/login", Login)

	hash := services.Hash("correctpassword")
	testutil.CreateBusiness(db.DB, map[string]interface{}{
		"Email":    "success@example.com",
		"Slug":     "success-biz",
		"Password": &hash,
	})

	w := httptest.NewRecorder()
	req := newFormRequest("POST", "/business/login", map[string]string{
		"email":    "success@example.com",
		"password": "correctpassword",
	})
	eng.ServeHTTP(w, req)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/business", w.Header().Get("Location"))
}

func TestLogout(t *testing.T) {
	eng := setupAuthEngine(t)
	eng.POST("/business/logout", Logout)

	w := httptest.NewRecorder()
	req := newFormRequest("POST", "/business/logout", nil)
	eng.ServeHTTP(w, req)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/business/login", w.Header().Get("Location"))
}
