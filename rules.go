package validate

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Required rejects an empty string. It intentionally does not trim whitespace
// so migrated endpoints preserve Gin validator's existing required semantics.
// Use NotBlank when whitespace-only input must also be rejected.
func Required() Rule[string] {
	return func(value string) error {
		if value == "" {
			return errors.New("不能为空")
		}
		return nil
	}
}

// NotBlank rejects strings that are empty or contain only whitespace.
func NotBlank() Rule[string] {
	return func(value string) error {
		if strings.TrimSpace(value) == "" {
			return errors.New("不能为空")
		}
		return nil
	}
}

// NotZero rejects the zero value. Its explicit type argument keeps the API
// type-safe because Go cannot infer a type parameter from a zero-argument rule.
// It also works for pointers: NotZero[*string]() rejects a nil pointer.
func NotZero[T comparable]() Rule[T] {
	return func(value T) error {
		var zero T
		if value == zero {
			return errors.New("不能为空")
		}
		return nil
	}
}

// Min rejects values smaller than minimum.
func Min[T cmp.Ordered](minimum T) Rule[T] {
	return func(value T) error {
		if value < minimum {
			return fmt.Errorf("不能小于 %v", minimum)
		}
		return nil
	}
}

// Max rejects values greater than maximum.
func Max[T cmp.Ordered](maximum T) Rule[T] {
	return func(value T) error {
		if value > maximum {
			return fmt.Errorf("不能大于 %v", maximum)
		}
		return nil
	}
}

// Between rejects values outside the inclusive [minimum, maximum] range.
func Between[T cmp.Ordered](minimum, maximum T) Rule[T] {
	return func(value T) error {
		if value < minimum || value > maximum {
			return fmt.Errorf("必须在 %v 和 %v 之间", minimum, maximum)
		}
		return nil
	}
}

// Positive rejects values less than or equal to zero.
func Positive[T cmp.Ordered]() Rule[T] {
	return func(value T) error {
		var zero T
		if value <= zero {
			return errors.New("必须大于 0")
		}
		return nil
	}
}

// NonNegative rejects values less than zero.
func NonNegative[T cmp.Ordered]() Rule[T] {
	return func(value T) error {
		var zero T
		if value < zero {
			return errors.New("不能小于 0")
		}
		return nil
	}
}

// Negative rejects values greater than or equal to zero.
func Negative[T cmp.Ordered]() Rule[T] {
	return func(value T) error {
		var zero T
		if value >= zero {
			return errors.New("必须小于 0")
		}
		return nil
	}
}

// NonPositive rejects values greater than zero.
func NonPositive[T cmp.Ordered]() Rule[T] {
	return func(value T) error {
		var zero T
		if value > zero {
			return errors.New("不能大于 0")
		}
		return nil
	}
}

// Eq rejects values different from want.
func Eq[T comparable](want T) Rule[T] {
	return func(value T) error {
		if value != want {
			return fmt.Errorf("必须等于 %v", want)
		}
		return nil
	}
}

// Ne rejects values equal to unwanted.
func Ne[T comparable](unwanted T) Rule[T] {
	return func(value T) error {
		if value == unwanted {
			return fmt.Errorf("不能等于 %v", unwanted)
		}
		return nil
	}
}

// OneOf rejects values that are not in the allowed set.
func OneOf[T comparable](values ...T) Rule[T] {
	return func(value T) error {
		if slices.Contains(values, value) {
			return nil
		}
		return errors.New("值不合法")
	}
}

// NotOneOf rejects values that are in the forbidden set.
func NotOneOf[T comparable](values ...T) Rule[T] {
	return func(value T) error {
		if slices.Contains(values, value) {
			return errors.New("值不合法")
		}
		return nil
	}
}
