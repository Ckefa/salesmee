package payment

func GetProvider(name string) PaymentProvider {
	switch name {
	case "paddle":
		return NewPaddleAdapter()
	default:
		return NewStripeAdapter()
	}
}
