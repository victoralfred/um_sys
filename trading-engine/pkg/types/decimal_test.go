package types

import (
	"encoding/json"
	"testing"
)

func TestNewDecimal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid integer", "123", "123", false},
		{"valid decimal", "123.456", "123.456", false},
		{"valid negative", "-123.456", "-123.456", false},
		{"zero", "0", "0", false},
		{"invalid string", "abc", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDecimal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDecimal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.String() != tt.want {
				t.Errorf("NewDecimal() = %v, want %v", got.String(), tt.want)
			}
		})
	}
}

func TestDecimalArithmetic(t *testing.T) {
	a, _ := NewDecimal("10.5")
	b, _ := NewDecimal("2.5")

	tests := []struct {
		name     string
		operation func() Decimal
		want     string
	}{
		{"addition", func() Decimal { return a.Add(b) }, "13"},
		{"subtraction", func() Decimal { return a.Sub(b) }, "8"},
		{"multiplication", func() Decimal { return a.Mul(b) }, "26.25"},
		{"division", func() Decimal { return a.Div(b) }, "4.2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.operation()
			if got.String() != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, got.String(), tt.want)
			}
		})
	}
}

func TestDecimalComparison(t *testing.T) {
	tests := []struct {
		name   string
		a, b   string
		wantCmp int
	}{
		{"equal", "10.5", "10.5", 0},
		{"greater", "10.5", "9.5", 1},
		{"less", "9.5", "10.5", -1},
		{"zero comparison", "0", "0.0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := NewDecimal(tt.a)
			b, _ := NewDecimal(tt.b)
			got := a.Cmp(b)
			if got != tt.wantCmp {
				t.Errorf("Cmp() = %v, want %v", got, tt.wantCmp)
			}
		})
	}
}

func TestDecimalJSON(t *testing.T) {
	original, _ := NewDecimal("123.456")
	
	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Errorf("Marshal error: %v", err)
	}
	
	// Unmarshal
	var restored Decimal
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Errorf("Unmarshal error: %v", err)
	}
	
	if original.Cmp(restored) != 0 {
		t.Errorf("JSON roundtrip failed: %v != %v", original.String(), restored.String())
	}
}

func TestDecimalPrecision(t *testing.T) {
	// Test that we maintain precision in financial calculations
	price, _ := NewDecimal("1234.56789")
	quantity, _ := NewDecimal("100.123456")
	
	total := price.Mul(quantity)
	
	// Check that we get a reasonable approximation
	expectedFloat := 1234.56789 * 100.123456
	actualFloat := total.Float64()
	
	if diff := actualFloat - expectedFloat; diff > 0.01 || diff < -0.01 {
		t.Errorf("Precision test failed: got %f, expected approximately %f", actualFloat, expectedFloat)
	}
}

func BenchmarkDecimalArithmetic(b *testing.B) {
	a, _ := NewDecimal("1234.5678")
	c, _ := NewDecimal("9876.5432")
	
	b.Run("Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = a.Add(c)
		}
	})
	
	b.Run("Mul", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = a.Mul(c)
		}
	})
}