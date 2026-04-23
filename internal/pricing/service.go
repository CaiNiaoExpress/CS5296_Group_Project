package pricing

import (
	"math"

	"github.com/example/car-mall-intelligent-agent/internal/model"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) ProcessPricing(task model.PricingTask) model.PricingResult {
	var tierDiscount float64
	switch task.CustomerTier {
	case "gold":
		tierDiscount = 0.08
	case "silver":
		tierDiscount = 0.05
	default:
		tierDiscount = 0.02
	}

	modelDiscount := 0.01
	if task.Model == "EV-Pro" {
		modelDiscount = 0.03
	}

	totalDiscount := tierDiscount + modelDiscount
	finalPrice := task.BasePrice * (1 - totalDiscount)
	finalPrice = math.Round(finalPrice*100) / 100

	return model.PricingResult{
		TaskID:         task.TaskID,
		FinalPrice:     finalPrice,
		Discount:       totalDiscount,
		EstimatedInSec: 2,
	}
}
