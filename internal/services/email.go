package services

import (
	"fmt"
	"log"
	"os"

	"github.com/resend/resend-go/v3"
)

func getFromEmail() string {
	from := os.Getenv("RESEND_FROM_EMAIL")
	if from == "" {
		from = "onboarding@resend.dev"
	}
	return from
}

func SendOTPEmail(toEmail, otpCode string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(apiKey)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; margin: 0; padding: 0;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 40px 20px;">
		<tr><td align="center">
			<table width="480" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 12px rgba(0,0,0,0.08);">
				<tr><td style="background: linear-gradient(135deg, #0d9488, #0891b2); padding: 32px; text-align: center;">
					<h1 style="color: #ffffff; font-size: 22px; margin: 0;">OneFlow</h1>
				</td></tr>
				<tr><td style="padding: 32px;">
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 16px;">Your verification code</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px;">
						Use the code below to complete your login. This code expires in 10 minutes.
					</p>
					<div style="background: #f1f5f9; border-radius: 8px; padding: 20px; text-align: center; letter-spacing: 8px; font-size: 32px; font-weight: 700; color: #0f172a;">
						%s
					</div>
					<p style="color: #94a3b8; font-size: 12px; line-height: 1.5; margin: 24px 0 0;">
						If you didn't request this code, you can safely ignore this email.
					</p>
				</td></tr>
				<tr><td style="background: #f8fafc; padding: 16px 32px; text-align: center; border-top: 1px solid #e2e8f0;">
					<p style="color: #94a3b8; font-size: 12px; margin: 0;">&copy; 2026 OneFlow. All rights reserved.</p>
				</td></tr>
			</table>
		</td></tr>
	</table>
</body>
</html>`, otpCode)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: "Your OneFlow verification code",
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
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(apiKey)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; margin: 0; padding: 0;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 40px 20px;">
		<tr><td align="center">
			<table width="480" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 12px rgba(0,0,0,0.08);">
				<tr><td style="background: linear-gradient(135deg, #0d9488, #0891b2); padding: 32px; text-align: center;">
					<h1 style="color: #ffffff; font-size: 22px; margin: 0;">OneFlow</h1>
				</td></tr>
				<tr><td style="padding: 32px;">
					<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">&#10004;&#65039;</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">Payment successful!</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Hi %s, your <strong>%s</strong> plan is now active.
					</p>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0;">
						You now have access to all the features included in your plan. If you have any questions, feel free to reach out to our support team.
					</p>
				</td></tr>
				<tr><td style="background: #f8fafc; padding: 16px 32px; text-align: center; border-top: 1px solid #e2e8f0;">
					<p style="color: #94a3b8; font-size: 12px; margin: 0;">&copy; 2026 OneFlow. All rights reserved.</p>
				</td></tr>
			</table>
		</td></tr>
	</table>
</body>
</html>`, businessName, planName)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: "Payment successful - OneFlow " + planName + " plan",
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
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(apiKey)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; margin: 0; padding: 0;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 40px 20px;">
		<tr><td align="center">
			<table width="480" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 12px rgba(0,0,0,0.08);">
				<tr><td style="background: linear-gradient(135deg, #dc2626, #ea580c); padding: 32px; text-align: center;">
					<h1 style="color: #ffffff; font-size: 22px; margin: 0;">OneFlow</h1>
				</td></tr>
				<tr><td style="padding: 32px;">
					<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">&#128276;</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">Subscription expired</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Hi %s, your OneFlow subscription has ended.
					</p>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0;">
						Your account has been downgraded to the Free plan. To regain access to premium features, please subscribe to a new plan.
					</p>
				</td></tr>
				<tr><td style="background: #f8fafc; padding: 16px 32px; text-align: center; border-top: 1px solid #e2e8f0;">
					<p style="color: #94a3b8; font-size: 12px; margin: 0;">&copy; 2026 OneFlow. All rights reserved.</p>
				</td></tr>
			</table>
		</td></tr>
	</table>
</body>
</html>`, businessName)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: "Your OneFlow subscription has ended",
		Html:    html,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send subscription expired email: %w", err)
	}

	log.Printf("Subscription expired email sent to %s: %s", toEmail, sent.Id)
	return nil
}

func SendSubscriptionFailed(toEmail, businessName string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(apiKey)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; margin: 0; padding: 0;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 40px 20px;">
		<tr><td align="center">
			<table width="480" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 12px rgba(0,0,0,0.08);">
				<tr><td style="background: linear-gradient(135deg, #dc2626, #ea580c); padding: 32px; text-align: center;">
					<h1 style="color: #ffffff; font-size: 22px; margin: 0;">OneFlow</h1>
				</td></tr>
				<tr><td style="padding: 32px;">
					<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">&#9888;&#65039;</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">Payment failed</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Hi %s, we were unable to process your latest payment.
					</p>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0;">
						Please update your payment method to avoid any interruption to your service. You can manage your billing details from your account dashboard.
					</p>
				</td></tr>
				<tr><td style="background: #f8fafc; padding: 16px 32px; text-align: center; border-top: 1px solid #e2e8f0;">
					<p style="color: #94a3b8; font-size: 12px; margin: 0;">&copy; 2026 OneFlow. All rights reserved.</p>
				</td></tr>
			</table>
		</td></tr>
	</table>
</body>
</html>`, businessName)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{toEmail},
		Subject: "Payment failed - OneFlow subscription",
		Html:    html,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send payment failed email: %w", err)
	}

	log.Printf("Payment failed email sent to %s: %s", toEmail, sent.Id)
	return nil
}
