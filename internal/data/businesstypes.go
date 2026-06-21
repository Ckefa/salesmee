package data

type BusinessType struct {
	Value string
	Label string
	Icon  string
	Color string
}

var BusinessTypes = []BusinessType{
	{"retail", "Retail & E-commerce", "shopping-bag", "teal"},
	{"food", "Food & Dining", "fire", "orange"},
	{"health", "Health & Medical", "heart", "red"},
	{"beauty_wellness", "Beauty & Wellness", "sparkles", "pink"},
	{"professional", "Professional Services", "briefcase", "sky"},
	{"home_services", "Home & Repair Services", "wrench", "yellow"},
	{"automotive", "Automotive", "truck", "orange"},
	{"education", "Education & Childcare", "academic-cap", "indigo"},
	{"real_estate", "Real Estate", "building-office", "amber"},
	{"agriculture", "Agriculture & Farming", "truck", "green"},
	{"other", "Other", "ellipsis-horizontal", "slate"},
}
