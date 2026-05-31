package payment

func GetProvider(name string) PaymentProvider {
	switch name {
	case "paypal":
		return NewPayPalAdapter()
	default:
		return NewStripeAdapter()
	}
}
