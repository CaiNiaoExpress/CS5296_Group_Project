package intent

import (
	"strings"
)

const (
	IntentPricing  = "smart_pricing"
	IntentPurchase = "purchase_consult"
	IntentLoan     = "loan_consult"
	IntentGeneral  = "general_qa"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Recognize(message string) string {
	msg := strings.ToLower(message)
	switch {
	case strings.Contains(msg, "price"), strings.Contains(msg, "quote"), strings.Contains(msg, "discount"),
		strings.Contains(msg, "cost"), strings.Contains(msg, "deal"):
		return IntentPricing
	case strings.Contains(msg, "buy"), strings.Contains(msg, "order"), strings.Contains(msg, "delivery"),
		strings.Contains(msg, "purchase"), strings.Contains(msg, "testdrive"), strings.Contains(msg, "testdrive"):
		return IntentPurchase
	case strings.Contains(msg, "loan"), strings.Contains(msg, "installment"), strings.Contains(msg, "interest"),
		strings.Contains(msg, "payment"), strings.Contains(msg, "finance"), strings.Contains(msg, "downpayment"):
		return IntentLoan
	default:
		return IntentGeneral
	}
}
