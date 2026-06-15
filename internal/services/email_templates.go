package services

import "fmt"

func brandHeader(gradient string) string {
	return fmt.Sprintf(`<tr style="background: %s;"><td style="background: #ffffff; padding: 12px 32px; text-align: center;" bgcolor="#ffffff"><img src="%s" alt="SalesMee" style="height: 28px; width: auto; display: block; margin: 0 auto;"></td></tr>`, gradient, AppURL("/static/images/salesmeebrand.png"))
}

const emailFooter = `<tr><td style="background: #f8fafc; padding: 16px 32px; text-align: center; border-top: 1px solid #e2e8f0;">
					<p style="color: #94a3b8; font-size: 12px; margin: 0;">&copy; 2026 salesmee. All rights reserved.</p>
				</td></tr>`

func emailHead(title string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; margin: 0; padding: 0;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 40px 20px;">
		<tr><td align="center">
			<table width="480" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 12px rgba(0,0,0,0.08);">
				%s
				<tr><td style="padding: 32px;">`, brandHeader("linear-gradient(135deg, #0d9488, #0891b2)"))
}

func emailTail() string {
	return fmt.Sprintf(`</td></tr>
				%s
			</table>
		</td></tr>
	</table>
</body>
</html>`, emailFooter)
}

func OTPEmailHTML(otpCode string) string {
	return emailHead("OTP") +
		`<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 16px;">Your verification code</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px;">
						Use the code below to complete your login. This code expires in 10 minutes.
					</p>
					<div style="background: #f1f5f9; border-radius: 8px; padding: 20px; text-align: center; letter-spacing: 8px; font-size: 32px; font-weight: 700; color: #0f172a;">
						` + otpCode + `
					</div>
					<p style="color: #94a3b8; font-size: 12px; line-height: 1.5; margin: 24px 0 0;">
						If you didn't request this code, you can safely ignore this email.
					</p>` +
		emailTail()
}

func SubscriptionSuccessHTML(businessName, planName string) string {
	return emailHead("Subscription Success") +
		fmt.Sprintf(`<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">&#10004;&#65039;</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">Payment successful!</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Hi %s, your <strong>%s</strong> plan is now active.
					</p>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0;">
						You now have access to all the features included in your plan. If you have any questions, feel free to reach out to our support team.
					</p>`, businessName, planName) +
		emailTail()
}

func SubscriptionExpiredHTML(businessName string) string {
	head := emailHead("Subscription Expired")
	head = replaceBrandHeader(head, "linear-gradient(135deg, #dc2626, #ea580c)")
	return head +
		fmt.Sprintf(`<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">&#128276;</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">Subscription expired</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Hi %s, your SalesMee subscription has ended.
					</p>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0;">
						Your account has been downgraded to the Free plan. To regain access to premium features, please subscribe to a new plan.
					</p>`, businessName) +
		emailTail()
}

func PasswordResetHTML(resetLink string) string {
	return emailHead("Password Reset") +
		fmt.Sprintf(`<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">&#128273;</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">Reset your password</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Click the button below to reset your password. This link expires in 1 hour.
					</p>
					<div style="text-align: center;">
						<a href="%s" style="display: inline-block; padding: 14px 32px; background: linear-gradient(135deg, #0d9488, #0891b2); color: #ffffff; text-decoration: none; border-radius: 8px; font-size: 15px; font-weight: 600;">Reset Password</a>
					</div>
					<p style="color: #94a3b8; font-size: 12px; line-height: 1.5; margin: 24px 0 0;">
						If you didn't request a password reset, you can safely ignore this email.
					</p>`, resetLink) +
		emailTail()
}

func VerificationEmailHTML(verifyLink string) string {
	return emailHead("Verify Email") +
		fmt.Sprintf(`<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">&#10071;</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">Verify your email</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Click the button below to verify your email address and activate your account.
					</p>
					<div style="text-align: center;">
						<a href="%s" style="display: inline-block; padding: 14px 32px; background: linear-gradient(135deg, #0d9488, #0891b2); color: #ffffff; text-decoration: none; border-radius: 8px; font-size: 15px; font-weight: 600;">Verify Email</a>
					</div>
					<p style="color: #94a3b8; font-size: 12px; line-height: 1.5; margin: 24px 0 0;">
						If you didn't create an account, you can safely ignore this email.
					</p>`, verifyLink) +
		emailTail()
}

func SubscriptionFailedHTML(businessName string) string {
	head := emailHead("Payment Failed")
	head = replaceBrandHeader(head, "linear-gradient(135deg, #dc2626, #ea580c)")
	return head +
		fmt.Sprintf(`<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">&#9888;&#65039;</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">Payment failed</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Hi %s, we were unable to process your latest payment.
					</p>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0;">
						Please update your payment method to avoid any interruption to your service. You can manage your billing details from your account dashboard.
					</p>`, businessName) +
		emailTail()
}

func BookingReminderHTML(clientName, businessName, serviceName, date, time, duration string) string {
	return emailHead("Booking Reminder") +
		fmt.Sprintf(`<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">&#128197;</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">Upcoming appointment reminder</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Hi %s, this is a reminder for your upcoming appointment at <strong>%s</strong>.
					</p>
					<table width="100%%" cellpadding="0" cellspacing="0" style="background: #f1f5f9; border-radius: 8px; padding: 16px;">
						<tr><td style="padding: 4px 0;"><strong style="color: #1e293b; font-size: 14px;">Service:</strong></td><td style="color: #64748b; font-size: 14px;">%s</td></tr>
						<tr><td style="padding: 4px 0;"><strong style="color: #1e293b; font-size: 14px;">Date:</strong></td><td style="color: #64748b; font-size: 14px;">%s</td></tr>
						<tr><td style="padding: 4px 0;"><strong style="color: #1e293b; font-size: 14px;">Time:</strong></td><td style="color: #64748b; font-size: 14px;">%s</td></tr>
						<tr><td style="padding: 4px 0;"><strong style="color: #1e293b; font-size: 14px;">Duration:</strong></td><td style="color: #64748b; font-size: 14px;">%s</td></tr>
					</table>`, clientName, businessName, serviceName, date, time, duration) +
		emailTail()
}

func OrderStatusHTML(clientName, businessName, orderNumber, status, chatLink string) string {
	var statusEmoji, statusLine, ctaLabel string
	switch status {
	case "pending":
		statusEmoji = "&#128276;"
		statusLine = "is awaiting your confirmation"
		ctaLabel = "Review in Chat"
	case "confirmed", "client_confirmed":
		statusEmoji = "&#9989;"
		statusLine = "has been confirmed"
		ctaLabel = "View in Chat"
	case "paid":
		statusEmoji = "&#128179;"
		statusLine = "has been paid"
		ctaLabel = "View in Chat"
	case "completed", "fulfilled":
		statusEmoji = "&#127881;"
		statusLine = "is complete"
		ctaLabel = "View Receipt"
	case "cancelled":
		statusEmoji = "&#128683;"
		statusLine = "has been cancelled"
		ctaLabel = "View in Chat"
	default:
		statusEmoji = "&#128722;"
		statusLine = "has been updated"
		ctaLabel = "View in Chat"
	}

	return emailHead("Order Status") +
		fmt.Sprintf(`<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">%s</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">Order %s</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Hi %s, your order <strong>%s</strong> at %s %s.
					</p>
					<div style="text-align: center;">
						<a href="%s" style="display: inline-block; padding: 14px 32px; background: linear-gradient(135deg, #0d9488, #0891b2); color: #ffffff; text-decoration: none; border-radius: 8px; font-size: 15px; font-weight: 600;">%s</a>
					</div>`, statusEmoji, status, clientName, orderNumber, businessName, statusLine, chatLink, ctaLabel) +
		emailTail()
}

func BookingStatusHTML(clientName, businessName, bookingNumber, status, chatLink string) string {
	var statusEmoji, statusLine, ctaLabel string
	switch status {
	case "pending":
		statusEmoji = "&#128276;"
		statusLine = "is awaiting confirmation"
		ctaLabel = "Review in Chat"
	case "client_confirmed", "confirmed":
		statusEmoji = "&#9989;"
		statusLine = "has been confirmed"
		ctaLabel = "View in Chat"
	case "paid":
		statusEmoji = "&#128179;"
		statusLine = "has been paid"
		ctaLabel = "View in Chat"
	case "completed":
		statusEmoji = "&#127881;"
		statusLine = "is complete"
		ctaLabel = "View Receipt"
	case "cancelled":
		statusEmoji = "&#128683;"
		statusLine = "has been cancelled"
		ctaLabel = "View in Chat"
	default:
		statusEmoji = "&#128197;"
		statusLine = "has been updated"
		ctaLabel = "View in Chat"
	}

	return emailHead("Booking Status") +
		fmt.Sprintf(`<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">%s</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">Booking %s</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Hi %s, your booking <strong>%s</strong> at %s %s.
					</p>
					<div style="text-align: center;">
						<a href="%s" style="display: inline-block; padding: 14px 32px; background: linear-gradient(135deg, #0d9488, #0891b2); color: #ffffff; text-decoration: none; border-radius: 8px; font-size: 15px; font-weight: 600;">%s</a>
					</div>`, statusEmoji, status, clientName, bookingNumber, businessName, statusLine, chatLink, ctaLabel) +
		emailTail()
}

func PaymentReminderHTML(clientName, businessName, refNumber, amount string) string {
	head := emailHead("Payment Reminder")
	head = replaceBrandHeader(head, "linear-gradient(135deg, #dc2626, #ea580c)")
	return head +
		fmt.Sprintf(`<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">&#128179;</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">Payment reminder</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Hi %s, this is a reminder regarding <strong>%s</strong> at %s.
					</p>
					<table width="100%%" cellpadding="0" cellspacing="0" style="background: #f1f5f9; border-radius: 8px; padding: 16px;">
						<tr><td style="padding: 4px 0;"><strong style="color: #1e293b; font-size: 14px;">Reference:</strong></td><td style="color: #64748b; font-size: 14px;">%s</td></tr>
						<tr><td style="padding: 4px 0;"><strong style="color: #1e293b; font-size: 14px;">Amount due:</strong></td><td style="color: #64748b; font-size: 14px;">%s</td></tr>
					</table>`, clientName, refNumber, businessName, refNumber, amount) +
		emailTail()
}

func AbandonedCartHTML(clientName, businessName, orderNumber, link string) string {
	head := emailHead("Abandoned Cart")
	head = replaceBrandHeader(head, "linear-gradient(135deg, #f59e0b, #d97706)")
	return head +
		fmt.Sprintf(`<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">&#128722;</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">You left something behind!</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Hi %s, you have an unfinished order (<strong>%s</strong>) at %s. Complete it now before it's too late!
					</p>
					<div style="text-align: center;">
						<a href="%s" style="display: inline-block; padding: 14px 32px; background: linear-gradient(135deg, #f59e0b, #d97706); color: #ffffff; text-decoration: none; border-radius: 8px; font-size: 15px; font-weight: 600;">Complete Order</a>
					</div>`, clientName, orderNumber, businessName, link) +
		emailTail()
}

func InactiveClientHTML(clientName, businessName, link string) string {
	return emailHead("Re-engagement") +
		fmt.Sprintf(`<div style="text-align: center; margin-bottom: 20px; font-size: 48px;">&#128153;</div>
					<h2 style="color: #1e293b; font-size: 18px; margin: 0 0 8px; text-align: center;">We miss you!</h2>
					<p style="color: #64748b; font-size: 14px; line-height: 1.6; margin: 0 0 24px; text-align: center;">
						Hi %s, it's been a while since your last visit to <strong>%s</strong>. We'd love to see you again!
					</p>
					<div style="text-align: center;">
						<a href="%s" style="display: inline-block; padding: 14px 32px; background: linear-gradient(135deg, #0d9488, #0891b2); color: #ffffff; text-decoration: none; border-radius: 8px; font-size: 15px; font-weight: 600;">Visit Us Again</a>
					</div>`, clientName, businessName, link) +
		emailTail()
}

func replaceBrandHeader(html, gradient string) string {
	// Find the brand header in the pre-built emailHead and replace it
	old := brandHeader("linear-gradient(135deg, #0d9488, #0891b2)")
	new := brandHeader(gradient)
	for i := 0; i <= len(html)-len(old); i++ {
		if html[i:i+len(old)] == old {
			return html[:i] + new + html[i+len(old):]
		}
	}
	return html
}
