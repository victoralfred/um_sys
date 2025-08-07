package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// Decimal represents a fixed-point decimal number for financial calculations
// Uses big.Rat internally for precision
type Decimal struct {
	value *big.Rat
}

// NewDecimal creates a new decimal from a string
func NewDecimal(s string) (Decimal, error) {
	rat := new(big.Rat)
	if _, ok := rat.SetString(s); !ok {
		return Decimal{}, fmt.Errorf("invalid decimal: %s", s)
	}
	return Decimal{value: rat}, nil
}

// NewDecimalFromFloat creates a decimal from float64
func NewDecimalFromFloat(f float64) Decimal {
	return Decimal{value: big.NewRat(1, 1).SetFloat64(f)}
}

// NewDecimalFromInt creates a decimal from int64
func NewDecimalFromInt(i int64) Decimal {
	return Decimal{value: big.NewRat(i, 1)}
}

// Zero returns a decimal with value 0
func Zero() Decimal {
	return Decimal{value: big.NewRat(0, 1)}
}

// String returns string representation
func (d Decimal) String() string {
	if d.value == nil {
		return "0"
	}
	// Use RatString for exact representation, then clean it up
	str := d.value.RatString()
	if !strings.Contains(str, "/") {
		return str
	}
	// If it's a fraction, convert to decimal with reasonable precision
	f, _ := d.value.Float64()
	return removeTrailingZeros(fmt.Sprintf("%.8f", f))
}

// removeTrailingZeros removes trailing zeros from decimal string
func removeTrailingZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

// Float64 returns float64 representation
func (d Decimal) Float64() float64 {
	if d.value == nil {
		return 0
	}
	f, _ := d.value.Float64()
	return f
}

// Add returns d + other
func (d Decimal) Add(other Decimal) Decimal {
	result := new(big.Rat)
	result.Add(d.value, other.value)
	return Decimal{value: result}
}

// Sub returns d - other
func (d Decimal) Sub(other Decimal) Decimal {
	result := new(big.Rat)
	result.Sub(d.value, other.value)
	return Decimal{value: result}
}

// Mul returns d * other
func (d Decimal) Mul(other Decimal) Decimal {
	result := new(big.Rat)
	result.Mul(d.value, other.value)
	return Decimal{value: result}
}

// Div returns d / other
func (d Decimal) Div(other Decimal) Decimal {
	result := new(big.Rat)
	result.Quo(d.value, other.value)
	return Decimal{value: result}
}

// Cmp compares d and other
func (d Decimal) Cmp(other Decimal) int {
	return d.value.Cmp(other.value)
}

// IsZero returns true if decimal is zero
func (d Decimal) IsZero() bool {
	if d.value == nil {
		return true
	}
	return d.value.Sign() == 0
}

// IsPositive returns true if decimal is positive
func (d Decimal) IsPositive() bool {
	if d.value == nil {
		return false
	}
	return d.value.Sign() > 0
}

// IsNegative returns true if decimal is negative
func (d Decimal) IsNegative() bool {
	if d.value == nil {
		return false
	}
	return d.value.Sign() < 0
}

// Abs returns absolute value
func (d Decimal) Abs() Decimal {
	result := new(big.Rat)
	result.Abs(d.value)
	return Decimal{value: result}
}

// MarshalJSON implements json.Marshaler
func (d Decimal) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON implements json.Unmarshaler
func (d *Decimal) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	decimal, err := NewDecimal(s)
	if err != nil {
		return err
	}

	*d = decimal
	return nil
}

// Value implements driver.Valuer for database storage
func (d Decimal) Value() (driver.Value, error) {
	return d.String(), nil
}

// Scan implements sql.Scanner for database retrieval
func (d *Decimal) Scan(value interface{}) error {
	if value == nil {
		*d = Zero()
		return nil
	}

	var s string
	switch v := value.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	case float64:
		s = strconv.FormatFloat(v, 'f', -1, 64)
	case int64:
		s = strconv.FormatInt(v, 10)
	default:
		return fmt.Errorf("cannot scan %T into Decimal", value)
	}

	decimal, err := NewDecimal(strings.TrimSpace(s))
	if err != nil {
		return err
	}

	*d = decimal
	return nil
}
