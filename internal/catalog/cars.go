package catalog

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Car struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Type             string  `json:"type"`
	RangeKM          int     `json:"range_km"`
	Acceleration0100 float64 `json:"acceleration_0_100"`
	BasePrice        float64 `json:"base_price"`
	Image            string  `json:"image"`
}

type carTemplate struct {
	Slug      string
	Name      string
	Type      string
	RangeKM   int
	Accel     float64
	BasePrice float64
	Image     string
}

var (
	carListOnce sync.Once
	carList     []Car
)

func ListCars() []Car {
	carListOnce.Do(func() {
		carList = buildCars()
	})
	return carList
}

func RecommendCars(message string, limit int) []Car {
	if limit <= 0 {
		limit = 3
	}

	msg := strings.ToLower(message)
	budget := extractBudget(message)
	typeKeywords := []string{
		"suv", "sedan", "mpv", "sports", "coupe", "wagon", "hatchback", "electric", "ev", "hybrid", "phev", "7seater", "family", "city",
	}

	type scoreCar struct {
		car   Car
		score int
	}

	scored := make([]scoreCar, 0, len(ListCars()))
	for _, car := range ListCars() {
		score := 0
		nameType := strings.ToLower(car.Name + " " + car.Type)

		for _, kw := range typeKeywords {
			if strings.Contains(msg, kw) && strings.Contains(nameType, strings.ToLower(kw)) {
				score += 5
			}
		}

		if strings.Contains(msg, "family") {
			if strings.Contains(car.Type, "SUV") || strings.Contains(car.Type, "Family") || strings.Contains(car.Type, "7-Seater") {
				score += 4
			}
		}

		if strings.Contains(msg, "loan") || strings.Contains(msg, "installment") {
			if car.BasePrice <= 300000 {
				score += 2
			}
		}

		if budget > 0 {
			diff := car.BasePrice - budget
			switch {
			case diff <= 0:
				score += 8
			case diff <= 30000:
				score += 4
			case diff <= 60000:
				score += 1
			default:
				score -= 4
			}
		}

		if strings.Contains(msg, "range") || strings.Contains(msg, "longdistance") {
			score += car.RangeKM / 200
		}

		if strings.Contains(msg, "performance") || strings.Contains(msg, "acceleration") {
			if car.Acceleration0100 < 5 {
				score += 3
			}
		}

		if score > 0 {
			scored = append(scored, scoreCar{car: car, score: score})
		}
	}

	if len(scored) == 0 {
		fallback := ListCars()
		if len(fallback) < limit {
			return fallback
		}
		return fallback[:limit]
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].car.BasePrice < scored[j].car.BasePrice
		}
		return scored[i].score > scored[j].score
	})

	result := make([]Car, 0, limit)
	for _, item := range scored {
		result = append(result, item.car)
		if len(result) == limit {
			break
		}
	}
	return result
}

func buildCars() []Car {
	brands := []string{"Apex", "Nova", "Aurora", "Titan", "Voyage", "Zenith", "Pioneer", "Vertex", "Orbit", "Summit"}
	trims := []string{"Air", "Plus", "Pro", "Max", "Ultra", "Elite", "Sport", "Touring", "Signature", "Flagship"}
	templates := []carTemplate{
		{Slug: "ev-pro", Name: "EV Pro", Type: "New Energy Sedan", RangeKM: 650, Accel: 3.9, BasePrice: 269000, Image: "https://images.unsplash.com/photo-1619767886558-efdc259cde1a?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "family-suv", Name: "Family SUV", Type: "Family SUV", RangeKM: 820, Accel: 6.4, BasePrice: 318000, Image: "https://images.unsplash.com/photo-1549399542-7e3f8b79c341?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "city-mini", Name: "City Mini", Type: "City Car", RangeKM: 420, Accel: 8.7, BasePrice: 139000, Image: "https://images.unsplash.com/photo-1511919884226-fd3cad34687c?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "lux-sedan", Name: "Lux Sedan", Type: "Luxury Sedan", RangeKM: 710, Accel: 4.4, BasePrice: 388000, Image: "https://images.unsplash.com/photo-1503376780353-7e6692767b70?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "urban-suv", Name: "Urban SUV", Type: "Electric SUV", RangeKM: 560, Accel: 5.8, BasePrice: 239000, Image: "https://images.unsplash.com/photo-1492144534655-ae79c964c9d7?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "roadster", Name: "Roadster", Type: "Sports Car", RangeKM: 540, Accel: 3.5, BasePrice: 459000, Image: "https://images.unsplash.com/photo-1502161254066-6c74afbf07aa?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "hybrid-commute", Name: "Hybrid Commute", Type: "Hybrid Sedan", RangeKM: 1100, Accel: 7.2, BasePrice: 189000, Image: "https://images.unsplash.com/photo-1494905998402-395d579af36f?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "executive", Name: "Executive", Type: "Executive Sedan", RangeKM: 760, Accel: 5.2, BasePrice: 429000, Image: "https://images.unsplash.com/photo-1553440569-bcc63803a83d?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "adventure", Name: "Adventure", Type: "7-Seater SUV", RangeKM: 860, Accel: 6.8, BasePrice: 336000, Image: "https://images.unsplash.com/photo-1519641471654-76ce0107ad1b?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "smart-hatch", Name: "Smart Hatch", Type: "Hatchback EV", RangeKM: 460, Accel: 7.9, BasePrice: 158000, Image: "https://images.unsplash.com/photo-1606664515524-ed2f786a0bd6?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "performance-gt", Name: "Performance GT", Type: "High-Performance Coupe", RangeKM: 630, Accel: 3.7, BasePrice: 399000, Image: "https://images.unsplash.com/photo-1494976388531-d1058494cdd8?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "eco-suv", Name: "Eco SUV", Type: "Compact SUV", RangeKM: 510, Accel: 7.5, BasePrice: 179000, Image: "https://images.unsplash.com/photo-1533473359331-0135ef1b58bf?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "business-mpv", Name: "Business MPV", Type: "Business MPV", RangeKM: 900, Accel: 8.3, BasePrice: 358000, Image: "https://images.unsplash.com/photo-1552519507-da3b142c6e3d?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "youth-coupe", Name: "Youth Coupe", Type: "Coupe", RangeKM: 520, Accel: 5.9, BasePrice: 229000, Image: "https://images.unsplash.com/photo-1489824904134-891ab64532f1?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "mountain-phev", Name: "Mountain PHEV", Type: "PHEV SUV", RangeKM: 1180, Accel: 6.1, BasePrice: 289000, Image: "https://images.unsplash.com/photo-1511919884226-fd3cad34687c?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "elite-wagon", Name: "Elite Wagon", Type: "Wagon", RangeKM: 680, Accel: 6.0, BasePrice: 312000, Image: "https://images.unsplash.com/photo-1493238792000-8113da705763?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "city-suv", Name: "City SUV", Type: "City SUV", RangeKM: 580, Accel: 6.9, BasePrice: 219000, Image: "https://images.unsplash.com/photo-1504215680853-026ed2a45def?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "flagship-ev", Name: "Flagship EV", Type: "Flagship Electric Sedan", RangeKM: 780, Accel: 3.8, BasePrice: 468000, Image: "https://images.unsplash.com/photo-1498887960847-2a5e46312788?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "family-hybrid", Name: "Family Hybrid", Type: "7-Seater Hybrid SUV", RangeKM: 1020, Accel: 7.1, BasePrice: 276000, Image: "https://images.unsplash.com/photo-1532581140115-3e355d1ed1de?auto=format&fit=crop&w=1200&q=80"},
		{Slug: "entry-sedan", Name: "Entry Sedan", Type: "Entry-Level Electric Sedan", RangeKM: 430, Accel: 8.5, BasePrice: 129000, Image: "https://images.unsplash.com/photo-1542282088-fe8426682b8f?auto=format&fit=crop&w=1200&q=80"},
	}

	cars := make([]Car, 0, len(templates)*len(brands)*len(trims))
	for templateIdx, tpl := range templates {
		for brandIdx, brand := range brands {
			for trimIdx, trim := range trims {
				year := 2025 + (brandIdx+trimIdx+templateIdx)%3
				rangeDelta := brandIdx*8 + trimIdx*11 - 36
				accelDelta := float64((trimIdx%5)-2) * 0.15
				priceDelta := float64(brandIdx*5000 + trimIdx*6500 + templateIdx*1800)
				id := fmt.Sprintf("%s-%s-%s-%d", tpl.Slug, slugify(brand), slugify(trim), year)
				name := fmt.Sprintf("%s %s %s %d", brand, tpl.Name, trim, year)

				cars = append(cars, Car{
					ID:               id,
					Name:             name,
					Type:             tpl.Type,
					RangeKM:          tpl.RangeKM + rangeDelta,
					Acceleration0100: roundOneDecimal(tpl.Accel - accelDelta),
					BasePrice:        tpl.BasePrice + priceDelta,
					Image:            tpl.Image,
				})
			}
		}
	}
	return cars
}

func slugify(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, " ", "-"))
}

func roundOneDecimal(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}

func extractBudget(message string) float64 {
	msg := strings.ToLower(strings.ReplaceAll(message, " ", ""))
	words := []string{"budget", "price", "cost", "within", "notover", "under"}
	for _, marker := range words {
		if idx := strings.Index(msg, marker); idx >= 0 {
			if amount := scanAmount(msg[idx+len(marker):]); amount > 0 {
				return amount
			}
		}
	}
	return scanAmount(msg)
}

func scanAmount(text string) float64 {
	for i := 0; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			continue
		}
		j := i
		for j < len(text) && text[j] >= '0' && text[j] <= '9' {
			j++
		}
		raw := text[i:j]
		n, err := strconv.Atoi(raw)
		if err != nil {
			continue
		}

		if strings.HasPrefix(text[j:], "w") || strings.HasPrefix(text[j:], "k") || strings.HasPrefix(text[j:], "0000") {
			return float64(n) * 10000
		}
		if n >= 10 && n <= 500 {
			return float64(n) * 10000
		}
		if n >= 100000 {
			return float64(n)
		}
	}
	return 0
}
