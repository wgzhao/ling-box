package bmi

import (
	"math"
	"testing"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name     string
		height   float64
		weight   float64
		wantBMI  float64
		wantCat  Category
	}{
		{"underweight", 175, 50, 16.3, Underweight},
		{"normal", 170, 65, 22.5, Normal},
		{"overweight", 170, 80, 27.7, Overweight},
		{"obese I", 160, 85, 33.2, ObeseClassI},
		{"obese II", 160, 100, 39.1, ObeseClassII},
		{"obese III", 160, 120, 46.9, ObeseClassIII},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := Calculate(tt.height, tt.weight)
			if err != nil {
				t.Fatalf("Calculate returned error: %v", err)
			}
			if math.Round(r.BMI*10)/10 != tt.wantBMI {
				t.Errorf("BMI = %.1f, want %.1f", r.BMI, tt.wantBMI)
			}
			if r.Category != tt.wantCat {
				t.Errorf("Category = %q, want %q", r.Category, tt.wantCat)
			}
		})
	}
}

func TestCalculateEdgeCases(t *testing.T) {
	// Exactly at boundaries
	r, err := Calculate(100, 18.5)
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	if r.Category != Normal {
		t.Errorf("BMI 18.5 should be Normal, got %s", r.Category)
	}
}

func TestCalculateErrors(t *testing.T) {
	_, err := Calculate(0, 70)
	if err == nil {
		t.Error("expected error for zero height")
	}

	_, err = Calculate(170, 0)
	if err == nil {
		t.Error("expected error for zero weight")
	}

	_, err = Calculate(-170, 70)
	if err == nil {
		t.Error("expected error for negative height")
	}
}

func TestString(t *testing.T) {
	r, _ := Calculate(170, 65)
	s := r.String()
	if len(s) == 0 {
		t.Error("String() returned empty")
	}
}
