package services

import (
	"fmt"
	"log"
	"strings"

	"salesmee/internal/config"

	"github.com/resend/resend-go/v3"
)

func getFromEmail() string {
	from := config.C.ResendFromEmail
	if from == "" {
		from = "onboarding@resend.dev"
	}
	return from
}

func isResendEnabled() bool {
	return config.C.ResendEnabled
}

func AppURL(path string) string {
	base := config.C.AppURL
	if base == "" {
		domain := config.C.AppDomain
		if domain == "" {
			return path
		}
		base = fmt.Sprintf("https://%s", domain)
	}
	return fmt.Sprintf("%s%s", strings.TrimRight(base, "/"), path)
}

func SendOTPEmail(toEmail, otpCode string) error {
	if !isResendEnabled() {
		log.Printf("[EMAIL SKIPPED] RESEND not active, email not sent:\n  To: %s\n  Subject: Your SalesMee verification code\n  Body: Your verification code is: %s\n  Expires in: 10 minutes", toEmail, otpCode)
		return nil
	}
	if config.C.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(config.C.ResendAPIKey)

	html := OTPEmailHTML(otpCode)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: "Your SalesMee verification code",
		Html:    html,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	log.Printf("OTP email sent to %s: %s", toEmail, sent.Id)
	return nil
}

func SendSubscriptionSuccess(toEmail, businessName, planName string) error {
	if !isResendEnabled() {
		log.Printf("[EMAIL SKIPPED] RESEND not active, email not sent:\n  To: %s\n  Subject: Payment successful - SalesMee %s plan\n  Body: Hi %s, your %s plan is now active.", toEmail, planName, businessName, planName)
		return nil
	}
	if config.C.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(config.C.ResendAPIKey)

	html := SubscriptionSuccessHTML(businessName, planName)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: "Payment successful - SalesMee " + planName + " plan",
		Html:    html,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send subscription success email: %w", err)
	}

	log.Printf("Subscription success email sent to %s: %s", toEmail, sent.Id)
	return nil
}

func SendSubscriptionExpired(toEmail, businessName string) error {
	if !isResendEnabled() {
		log.Printf("[EMAIL SKIPPED] RESEND not active, email not sent:\n  To: %s\n  Subject: Your salesmee subscription has ended\n  Body: Hi %s, your SalesMee subscription has ended. Your account has been downgraded to the Free plan.", toEmail, businessName)
		return nil
	}
	if config.C.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(config.C.ResendAPIKey)

	html := SubscriptionExpiredHTML(businessName)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: "Your salesmee subscription has ended",
		Html:    html,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send subscription expired email: %w", err)
	}

	log.Printf("Subscription expired email sent to %s: %s", toEmail, sent.Id)
	return nil
}

func SendPasswordResetEmail(toEmail, resetLink string) error {
	if !isResendEnabled() {
		log.Printf("[EMAIL SKIPPED] RESEND not active, email not sent:\n  To: %s\n  Subject: Reset your SalesMee password\n  Body: Click the link below to reset your password (expires in 1 hour):\n  %s", toEmail, resetLink)
		return nil
	}
	if config.C.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(config.C.ResendAPIKey)

	html := PasswordResetHTML(resetLink)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: "Reset your SalesMee password",
		Html:    html,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	log.Printf("Password reset email sent to %s: %s", toEmail, sent.Id)
	return nil
}

func SendVerificationEmail(toEmail, verifyLink string) error {
	if !isResendEnabled() {
		log.Printf("[EMAIL SKIPPED] RESEND not active, email not sent:\n  To: %s\n  Subject: Verify your SalesMee email address\n  Body: Click the link below to verify your email:\n  %s", toEmail, verifyLink)
		return nil
	}
	if config.C.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(config.C.ResendAPIKey)

	html := VerificationEmailHTML(verifyLink)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: "Verify your SalesMee email address",
		Html:    html,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	log.Printf("Verification email sent to %s: %s", toEmail, sent.Id)
	return nil
}

func SendSubscriptionFailed(toEmail, businessName string) error {
	if !isResendEnabled() {
		log.Printf("[EMAIL SKIPPED] RESEND not active, email not sent:\n  To: %s\n  Subject: Payment failed - SalesMee subscription\n  Body: Hi %s, we were unable to process your latest payment. Please update your payment method.", toEmail, businessName)
		return nil
	}
	if config.C.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(config.C.ResendAPIKey)

	html := SubscriptionFailedHTML(businessName)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: "Payment failed - SalesMee subscription",
		Html:    html,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send payment failed email: %w", err)
	}

	log.Printf("Payment failed email sent to %s: %s", toEmail, sent.Id)
	return nil
}

func SendBookingReminderEmail(toEmail, clientName, businessName, serviceName, date, time, duration string) error {
	if !isResendEnabled() {
		log.Printf("[EMAIL SKIPPED] RESEND not active, email not sent:\n  To: %s\n  Subject: Reminder — %s at %s", toEmail, serviceName, date)
		return nil
	}
	if config.C.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(config.C.ResendAPIKey)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; margin: 0; padding: 0;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 40px 20px;">
		<tr><td align="center">
			<table width="480" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 12px rgba(0,0,0,0.08);">
				<tr><td style="background: linear-gradient(135deg, #0d9488, #0891b2); padding: 32px; text-align: center;">
					<h1 style="color: #ffffff; font-size: 22px; margin: 0;">salesmee</h1>
				</td></tr>
				<tr><td style="padding: 32px;">
					<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">&#128197;</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">Upcoming appointment reminder</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Hi %s, this is a reminder for your upcoming appointment at <strong>%s</strong>.
					</p>
					<table width="100%%" cellpadding="0" cellspacing="0" style="background: #f1f5f9; border-radius: 8px; padding: 16px;">
						<tr><td style="padding: 4px 0;"><strong style="color: #1e293b; font-size: 14px;">Service:</strong></td><td style="color: #64748b; font-size: 14px;">%s</td></tr>
						<tr><td style="padding: 4px 0;"><strong style="color: #1e293b; font-size: 14px;">Date:</strong></td><td style="color: #64748b; font-size: 14px;">%s</td></tr>
						<tr><td style="padding: 4px 0;"><strong style="color: #1e293b; font-size: 14px;">Time:</strong></td><td style="color: #64748b; font-size: 14px;">%s</td></tr>
						<tr><td style="padding: 4px 0;"><strong style="color: #1e293b; font-size: 14px;">Duration:</strong></td><td style="color: #64748b; font-size: 14px;">%s</td></tr>
					</table>
				</td></tr>
				<tr><td style="background: #f8fafc; padding: 16px 32px; text-align: center; border-top: 1px solid #e2e8f0;">
					<p style="color: #94a3b8; font-size: 12px; margin: 0;">&copy; 2026 salesmee. All rights reserved.</p>
				</td></tr>
			</table>
		</td></tr>
	</table>
</body>
</html>`, clientName, businessName, serviceName, date, time, duration)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: fmt.Sprintf("Reminder — %s at %s", serviceName, businessName),
		Html:    html,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send booking reminder email: %w", err)
	}

	log.Printf("Booking reminder email sent to %s: %s", toEmail, sent.Id)
	return nil
}

func SendOrderStatusEmail(toEmail, clientName, businessName, orderNumber, status, chatLink string) error {
	if !isResendEnabled() {
		log.Printf("[EMAIL SKIPPED] RESEND not active, email not sent:\n  To: %s\n  Subject: Order %s — %s\n  Chat: %s", toEmail, orderNumber, status, chatLink)
		return nil
	}
	if config.C.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(config.C.ResendAPIKey)

	html := OrderStatusHTML(clientName, businessName, orderNumber, status, chatLink)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: fmt.Sprintf("Order %s — %s", orderNumber, status),
		Html:    html,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send order status email: %w", err)
	}

	log.Printf("Order status email sent to %s: %s", toEmail, sent.Id)
	return nil
}

func SendBookingStatusEmail(toEmail, clientName, businessName, bookingNumber, status, chatLink string) error {
	if !isResendEnabled() {
		log.Printf("[EMAIL SKIPPED] RESEND not active, email not sent:\n  To: %s\n  Subject: Booking %s — %s\n  Chat: %s", toEmail, bookingNumber, status, chatLink)
		return nil
	}
	if config.C.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(config.C.ResendAPIKey)

	html := BookingStatusHTML(clientName, businessName, bookingNumber, status, chatLink)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: fmt.Sprintf("Booking %s — %s", bookingNumber, status),
		Html:    html,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send booking status email: %w", err)
	}

	log.Printf("Booking status email sent to %s: %s", toEmail, sent.Id)
	return nil
}

func SendPaymentReminderEmail(toEmail, clientName, businessName, refNumber, amount string) error {
	if !isResendEnabled() {
		log.Printf("[EMAIL SKIPPED] RESEND not active, email not sent:\n  To: %s\n  Subject: Payment reminder — %s", toEmail, refNumber)
		return nil
	}
	if config.C.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(config.C.ResendAPIKey)

	html := PaymentReminderHTML(clientName, businessName, refNumber, amount)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: fmt.Sprintf("Payment reminder — %s", refNumber),
		Html:    html,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send payment reminder email: %w", err)
	}

	log.Printf("Payment reminder email sent to %s: %s", toEmail, sent.Id)
	return nil
}

func SendAbandonedCartEmail(toEmail, clientName, businessName, orderNumber, link string) error {
	if !isResendEnabled() {
		log.Printf("[EMAIL SKIPPED] RESEND not active, email not sent:\n  To: %s\n  Subject: Complete your order %s", toEmail, orderNumber)
		return nil
	}
	if config.C.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(config.C.ResendAPIKey)

	html := AbandonedCartHTML(clientName, businessName, orderNumber, link)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: fmt.Sprintf("Complete your order %s at %s", orderNumber, businessName),
		Html:    html,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send abandoned cart email: %w", err)
	}

	log.Printf("Abandoned cart email sent to %s: %s", toEmail, sent.Id)
	return nil
}

func SendInactiveClientEmail(toEmail, clientName, businessName, link string) error {
	if !isResendEnabled() {
		log.Printf("[EMAIL SKIPPED] RESEND not active, email not sent:\n  To: %s\n  Subject: We miss you at %s", toEmail, businessName)
		return nil
	}
	if config.C.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(config.C.ResendAPIKey)

	html := InactiveClientHTML(clientName, businessName, link)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: fmt.Sprintf("We miss you at %s", businessName),
		Html:    html,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send inactive client email: %w", err)
	}

	log.Printf("Inactive client email sent to %s: %s", toEmail, sent.Id)
	return nil
}
