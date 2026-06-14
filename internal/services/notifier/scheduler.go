package notifier

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"salesmee/internal/models"
	"salesmee/internal/services"

	"gorm.io/gorm"
)

func StartNotificationScheduler(db *gorm.DB) {
	enabled := os.Getenv("NOTIF_SCHEDULER") == "true"
	if !enabled {
		log.Println("[NOTIFIER] Scheduler disabled (NOTIF_SCHEDULER != true)")
		return
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		log.Println("[NOTIFIER] Background scheduler started (every 60s)")
		for range ticker.C {
			CheckBookingReminders(db)
			CheckPaymentDueReminders(db)
			CheckAbandonedCarts(db)
			CheckReEngagement(db)
		}
	}()
}

func CheckBookingReminders(db *gorm.DB) {
	now := time.Now()

	checkWindow := func(hours float64) {
		windowStart := now.Add(time.Duration(hours)*time.Hour - 5*time.Minute)
		windowEnd := now.Add(time.Duration(hours)*time.Hour + 5*time.Minute)

		var bookings []models.Booking
		db.Where("status NOT IN ('completed','cancelled') AND scheduled_date BETWEEN ? AND ?",
			windowStart, windowEnd).
			Preload("Client").
			Preload("Business").
			Find(&bookings)

		for _, b := range bookings {
			notifType := "booking_reminder"
			rid := b.ID
			if HasBeenSent(db, b.BusinessID, b.ClientID, notifType, &rid) {
				continue
			}

			prefs, err := GetOrCreatePrefs(db, b.BusinessID)
			if err != nil {
				continue
			}
			if hours == 1 && !prefs.BookingReminder1h {
				continue
			}
			if hours == 24 && !prefs.BookingReminder24h {
				continue
			}

			serviceName := "Appointment"
			if len(b.BookingItems) > 0 {
				var item models.BookingItem
				db.First(&item, b.BookingItems[0].ID)
				var svc models.Service
				if db.First(&svc, item.ServiceID).Error == nil {
					serviceName = svc.Name
				}
			}

			dateStr := b.ScheduledDate.Format("Mon, Jan 2, 2006")
			timeStr := b.ScheduledDate.Format("3:04 PM")
			durStr := fmt.Sprintf("%d min", b.Duration)

			label := "1h"
			if hours == 24 {
				label = "24h"
			}

			err = services.SendBookingReminderEmail(b.Client.Email, b.Client.Name, b.Business.Name, serviceName, dateStr, timeStr, durStr)
			status := "sent"
			if err != nil {
				log.Printf("[NOTIFIER] Failed to send %s reminder for booking %d: %v", label, b.ID, err)
				status = "failed"
			}
			MarkNotificationSent(db, b.BusinessID, b.ClientID, notifType, "booking", &rid, b.Client.Email, status)

			CreateInAppNotif(db, b.BusinessID, &b.ClientID,
				fmt.Sprintf("%s Reminder", label),
				fmt.Sprintf("%s at %s — %s", serviceName, timeStr, dateStr),
				"fa-clock",
				"/business/bookings")
		}
	}

	checkWindow(1)
	checkWindow(24)
}

func CheckPaymentDueReminders(db *gorm.DB) {
	now := time.Now()
	cutoff := now.AddDate(0, 0, -7)

	var orders []models.Order
	db.Where("status NOT IN ('fulfilled','completed','cancelled') AND paid_amount < total_amount AND created_at < ?", cutoff).
		Preload("Client").
		Preload("Business").
		Find(&orders)

	for _, o := range orders {
		notifType := "payment_due"
		rid := o.ID
		if HasBeenSent(db, o.BusinessID, o.ClientID, notifType, &rid) {
			continue
		}
		prefs, err := GetOrCreatePrefs(db, o.BusinessID)
		if err != nil || !prefs.PaymentDueReminder {
			continue
		}

		due := fmt.Sprintf("%.2f", o.TotalAmount-o.PaidAmount)
		err = services.SendPaymentReminderEmail(o.Client.Email, o.Client.Name, o.Business.Name, o.OrderNumber, due)
		status := "sent"
		if err != nil {
			log.Printf("[NOTIFIER] Failed to send payment reminder for order %d: %v", o.ID, err)
			status = "failed"
		}
		MarkNotificationSent(db, o.BusinessID, o.ClientID, notifType, "order", &rid, o.Client.Email, status)
		CreateInAppNotif(db, o.BusinessID, &o.ClientID,
			"Payment Due",
			fmt.Sprintf("Order %s — %s due", o.OrderNumber, due),
			"fa-credit-card",
			"/business/orders")
	}

	var bookings []models.Booking
	db.Where("status NOT IN ('completed','cancelled') AND paid_amount < total_amount AND created_at < ?", cutoff).
		Preload("Client").
		Preload("Business").
		Find(&bookings)

	for _, b := range bookings {
		notifType := "payment_due"
		rid := b.ID
		if HasBeenSent(db, b.BusinessID, b.ClientID, notifType, &rid) {
			continue
		}
		prefs, err := GetOrCreatePrefs(db, b.BusinessID)
		if err != nil || !prefs.PaymentDueReminder {
			continue
		}

		due := fmt.Sprintf("%.2f", b.TotalAmount-b.PaidAmount)
		err = services.SendPaymentReminderEmail(b.Client.Email, b.Client.Name, b.Business.Name, b.BookingNumber, due)
		status := "sent"
		if err != nil {
			log.Printf("[NOTIFIER] Failed to send payment reminder for booking %d: %v", b.ID, err)
			status = "failed"
		}
		MarkNotificationSent(db, b.BusinessID, b.ClientID, notifType, "booking", &rid, b.Client.Email, status)
		CreateInAppNotif(db, b.BusinessID, &b.ClientID,
			"Payment Due",
			fmt.Sprintf("Booking %s — %s due", b.BookingNumber, due),
			"fa-credit-card",
			"/business/bookings")
	}
}

func CheckAbandonedCarts(db *gorm.DB) {
	var businesses []models.Business
	db.Find(&businesses)

	for _, biz := range businesses {
		prefs, err := GetOrCreatePrefs(db, biz.ID)
		if err != nil || !prefs.AbandonedCart {
			continue
		}
		hours := prefs.AbandonedCartHours
		if hours <= 0 {
			hours = 24
		}
		cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)

		var orders []models.Order
		db.Where("business_id = ? AND status IN ('pending','draft') AND created_at < ?", biz.ID, cutoff).
			Preload("Client").
			Find(&orders)

		for _, o := range orders {
			notifType := "abandoned_cart"
			rid := o.ID
			if HasBeenSent(db, biz.ID, o.ClientID, notifType, &rid) {
				continue
			}

			link := fmt.Sprintf("https://%s/b/%s", os.Getenv("APP_DOMAIN"), biz.Slug)
			if os.Getenv("APP_DOMAIN") == "" {
				link = fmt.Sprintf("/b/%s", biz.Slug)
			}

			err = services.SendAbandonedCartEmail(o.Client.Email, o.Client.Name, biz.Name, o.OrderNumber, link)
			status := "sent"
			if err != nil {
				log.Printf("[NOTIFIER] Failed to send abandoned cart email for order %d: %v", o.ID, err)
				status = "failed"
			}
			MarkNotificationSent(db, biz.ID, o.ClientID, notifType, "order", &rid, o.Client.Email, status)
			CreateInAppNotif(db, biz.ID, &o.ClientID,
				"Abandoned Cart",
				fmt.Sprintf("Order %s is still pending — remind client", o.OrderNumber),
				"fa-shopping-cart",
				"/business/orders")
		}
	}
}

func CheckReEngagement(db *gorm.DB) {
	var businesses []models.Business
	db.Find(&businesses)

	for _, biz := range businesses {
		prefs, err := GetOrCreatePrefs(db, biz.ID)
		if err != nil || !prefs.ReEngagement {
			continue
		}
		inactiveDays := prefs.InactiveDays
		if inactiveDays <= 0 {
			inactiveDays = 30
		}
		cutoff := time.Now().AddDate(0, 0, -inactiveDays)

		var clients []models.Client
		db.Where("business_id = ? AND (last_seen_at IS NULL OR last_seen_at < ?)", biz.ID, cutoff).
			Find(&clients)

		for _, c := range clients {
			notifType := "re_engagement"
			if HasBeenSent(db, biz.ID, c.ID, notifType, nil) {
				continue
			}
			lastSent := db.Where("business_id = ? AND client_id = ? AND type = ?", biz.ID, c.ID, notifType).
				Order("sent_at DESC").Limit(1).
				Find(&models.NotificationLog{})
			if lastSent.RowsAffected > 0 {
				continue
			}

			days := strconv.Itoa(inactiveDays)
			err = services.SendInactiveClientEmail(c.Email, c.Name, biz.Name, biz.Slug)
			status := "sent"
			if err != nil {
				log.Printf("[NOTIFIER] Failed to send re-engagement email for client %d: %v", c.ID, err)
				status = "failed"
			}
			MarkNotificationSent(db, biz.ID, c.ID, notifType, "", nil, c.Email, status)

			_ = days
			CreateInAppNotif(db, biz.ID, &c.ID,
				"Inactive Client",
				fmt.Sprintf("%s hasn't visited in %d days", c.Name, inactiveDays),
				"fa-user-clock",
				fmt.Sprintf("/business"))
		}
	}
}
