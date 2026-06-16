package payment

func GetProvider(name string) PaymentProvider {
	switch name {
	case "paddle":
		return NewPaddleAdapter()
	case "polar":
		return NewPolarAdapter()
	default:
		return NewStripeAdapter()
	}
}
