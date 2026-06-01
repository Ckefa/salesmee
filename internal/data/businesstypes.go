package data

type BusinessType struct {
	Value string
	Label string
	Icon  string
	Color string
}

var BusinessTypes = []BusinessType{
	{"retail", "Retail & E-commerce", "fa-bag-shopping", "teal"},
	{"food", "Food & Dining", "fa-utensils", "orange"},
	{"health", "Health & Medical", "fa-heart-pulse", "red"},
	{"beauty_wellness", "Beauty & Wellness", "fa-sparkles", "pink"},
	{"professional", "Professional Services", "fa-briefcase", "sky"},
	{"home_services", "Home & Repair Services", "fa-wrench", "yellow"},
	{"automotive", "Automotive", "fa-car", "orange"},
	{"education", "Education & Childcare", "fa-graduation-cap", "indigo"},
	{"real_estate", "Real Estate", "fa-building", "amber"},
	{"agriculture", "Agriculture & Farming", "fa-tractor", "green"},
	{"other", "Other", "fa-ellipsis", "slate"},
}
