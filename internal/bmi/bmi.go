package bmi

import (
	"fmt"
)

// Category describes the BMI classification.
type Category string

const (
	Underweight   Category = "Underweight"
	Normal        Category = "Normal weight"
	Overweight    Category = "Overweight"
	ObeseClassI   Category = "Obese (Class I)"
	ObeseClassII  Category = "Obese (Class II)"
	ObeseClassIII Category = "Obese (Class III)"
)

// Result holds the BMI calculation output.
type Result struct {
	BMI      float64  `json:"bmi"`
	Category Category `json:"category"`
}

// Calculate computes BMI from height (cm) and weight (kg).
func Calculate(heightCM, weightKG float64) (*Result, error) {
	if heightCM <= 0 {
		return nil, fmt.Errorf("height must be positive")
	}
	if weightKG <= 0 {
		return nil, fmt.Errorf("weight must be positive")
	}

	heightM := heightCM / 100.0
	bmi := weightKG / (heightM * heightM)

	cat := classify(bmi)

	return &Result{BMI: bmi, Category: cat}, nil
}

func classify(bmi float64) Category {
	switch {
	case bmi < 18.5:
		return Underweight
	case bmi < 25.0:
		return Normal
	case bmi < 30.0:
		return Overweight
	case bmi < 35.0:
		return ObeseClassI
	case bmi < 40.0:
		return ObeseClassII
	default:
		return ObeseClassIII
	}
}

func (r *Result) String() string {
	return fmt.Sprintf("BMI: %.1f (%s)", r.BMI, r.Category)
}
