package handlers

import (
	"net/http"
	"html"
	"salesmee/internal/db"
	"salesmee/internal/models"
	"salesmee/internal/services"
	"time"

	"github.com/gin-gonic/gin"
)

func DevPage(c *gin.Context) {
	// Get user info from context
	businessID, exists := c.Get("business_id")
	var isLoggedIn bool
	if exists {
		isLoggedIn = businessID != nil
	}

	// Get clients for display
	var clients []models.Client
	if exists {
		db.DB.Where("business_id = ?", businessID).Find(&clients)
	}

	c.HTML(http.StatusOK, "test.html", gin.H{
		"Title":    "Dev Panel",
		"LoggedIn": isLoggedIn,
		"Clients":  clients,
	})
}

func Ping(c *gin.Context) {
	c.HTML(http.StatusOK, "ping.html", gin.H{
		"Status": "OK",
		"Time":   time.Now().Format("15:04:05"),
	})
}

func ListItems(c *gin.Context) {
	userID := c.GetUint("business_id")
	var clients []models.Client
	db.DB.Where("business_id = ?", userID).Find(&clients)
	c.HTML(http.StatusOK, "items.html", gin.H{
		"Items": clients,
		"Count": len(clients),
	})
}

func CreateItem(c *gin.Context) {
	clientID := c.GetUint("client_id")
	name := c.PostForm("name")
	client := models.Client{
		ID:     clientID,
		Name:   name,
		Status: models.StatusNew,
	}
	db.DB.Create(&client)
	ListItems(c)
}

func DeleteItem(c *gin.Context) {
	userID := c.GetUint("business_id")
	id := c.Param("id")
	db.DB.Where("id = ? AND business_id = ?", id, userID).Delete(&models.Client{})
	ListItems(c)
}

type emailPreview struct {
	Name string
	HTML string
}

var emailPreviews = []emailPreview{
	{"OTP Email", services.OTPEmailHTML("123456")},
	{"Subscription Success", services.SubscriptionSuccessHTML("Acme Corp", "Gold")},
	{"Subscription Expired", services.SubscriptionExpiredHTML("Acme Corp")},
	{"Password Reset", services.PasswordResetHTML("https://app.salesmee.com/reset?token=abc123")},
	{"Verify Email", services.VerificationEmailHTML("https://app.salesmee.com/verify?token=abc123")},
	{"Subscription Payment Failed", services.SubscriptionFailedHTML("Acme Corp")},
	{"Booking Reminder", services.BookingReminderHTML("Jane Doe", "Acme Corp", "Haircut", "June 20, 2026", "2:00 PM", "45 min")},
	{"Order Status — Pending", services.OrderStatusHTML("Jane Doe", "Acme Corp", "ORD-001", models.OrderPending, "https://app.salesmee.com/chat/1")},
	{"Order Status — Confirmed", services.OrderStatusHTML("Jane Doe", "Acme Corp", "ORD-001", models.OrderConfirmed, "https://app.salesmee.com/chat/1")},
	{"Order Status — Paid", services.OrderStatusHTML("Jane Doe", "Acme Corp", "ORD-001", "paid", "https://app.salesmee.com/chat/1")},
	{"Order Status — Completed", services.OrderStatusHTML("Jane Doe", "Acme Corp", "ORD-001", models.OrderFulfilled, "https://app.salesmee.com/chat/1")},
	{"Order Status — Cancelled", services.OrderStatusHTML("Jane Doe", "Acme Corp", "ORD-001", models.OrderCancelled, "https://app.salesmee.com/chat/1")},
	{"Booking Status — Pending", services.BookingStatusHTML("Jane Doe", "Acme Corp", "BKG-001", models.OrderPending, "https://app.salesmee.com/chat/1")},
	{"Booking Status — Confirmed", services.BookingStatusHTML("Jane Doe", "Acme Corp", "BKG-001", models.OrderConfirmed, "https://app.salesmee.com/chat/1")},
	{"Booking Status — Completed", services.BookingStatusHTML("Jane Doe", "Acme Corp", "BKG-001", models.OrderCompleted, "https://app.salesmee.com/chat/1")},
	{"Booking Status — Cancelled", services.BookingStatusHTML("Jane Doe", "Acme Corp", "BKG-001", models.OrderCancelled, "https://app.salesmee.com/chat/1")},
	{"Payment Reminder", services.PaymentReminderHTML("Jane Doe", "Acme Corp", "ORD-001", "$50.00")},
	{"Abandoned Cart", services.AbandonedCartHTML("Jane Doe", "Acme Corp", "ORD-001", "https://app.salesmee.com/chat/1")},
	{"Inactive Client", services.InactiveClientHTML("Jane Doe", "Acme Corp", "https://app.salesmee.com/b/acme")},
}

func ShowEmailTestPage(c *gin.Context) {
	name := c.Query("name")
	if name != "" {
		for _, e := range emailPreviews {
			if e.Name == name {
				c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(e.HTML))
				return
			}
		}
		c.String(http.StatusNotFound, "Not found")
		return
	}

	page := `<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>Email Templates</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#f1f5f9;padding:24px;max-width:1200px;margin:0 auto}
h1{font-size:20px;margin-bottom:16px;color:#0f172a}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(340px,1fr));gap:16px}
.card{background:#fff;border-radius:8px;box-shadow:0 1px 3px rgba(0,0,0,0.1);overflow:hidden;transition:box-shadow .2s}
.card:hover{box-shadow:0 2px 8px rgba(0,0,0,0.15)}
.card-hdr{padding:14px 16px;background:#f8fafc;border-bottom:1px solid #e2e8f0;font-size:13px;font-weight:600;color:#1e293b;display:flex;justify-content:space-between;align-items:center}
.card-badge{font-size:11px;background:#0d9488;color:#fff;padding:2px 8px;border-radius:4px}
.preview{height:420px;overflow:auto;border-bottom:1px solid #e2e8f0}
.preview iframe{width:100%;height:100%;border:none}
.view-link{display:block;padding:10px 16px;text-align:center;background:#fff;border-top:1px solid #e2e8f0;color:#0d9488;text-decoration:none;font-size:13px;font-weight:500}
.view-link:hover{background:#f0fdfa}
</style></head>
<body><h1>` + "\u2709\ufe0f Email Templates (" + itoa(len(emailPreviews)) + `)</h1><div class="grid">`

	for _, e := range emailPreviews {
		page += `<div class="card"><div class="card-hdr"><span>` + e.Name + `</span><span class="card-badge">preview</span></div><div class="preview"><iframe srcdoc="` + html.EscapeString(e.HTML) + `"></iframe></div><a class="view-link" href="?name=` + e.Name + `" target="_blank">Open full page &rarr;</a></div>`
	}

	page += `</div></body></html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
