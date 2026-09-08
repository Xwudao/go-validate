package validate

import (
	"errors"
	"fmt"
)

// MinItems rejects a slice with fewer than minimum entries.
func MinItems[T any](minimum int) Rule[[]T] {
	return func(value []T) error {
		if len(value) < minimum {
			return fmt.Errorf("数量不能小于 %d", minimum)
		}
		return nil
	}
}

// MaxItems rejects a slice with more than maximum entries.
func MaxItems[T any](maximum int) Rule[[]T] {
	return func(value []T) error {
		if len(value) > maximum {
			return fmt.Errorf("数量不能大于 %d", maximum)
		}
		return nil
	}
}

// LenItems rejects a slice whose length differs from exact.
func LenItems[T any](exact int) Rule[[]T] {
	return func(value []T) error {
		if len(value) != exact {
			return fmt.Errorf("数量必须为 %d", exact)
		}
		return nil
	}
}

// NonEmptyMap rejects empty maps, matching the existing go-playground required
// rule applied to map fields (len(map) > 0).
func NonEmptyMap[K comparable, V any]() Rule[map[K]V] {
	return func(value map[K]V) error {
		if len(value) == 0 {
			return errors.New("不能为空")
		}
		return nil
	}
}

// Unique rejects slices that contain duplicate comparable values.
func Unique[T comparable]() Rule[[]T] {
	return func(value []T) error {
		seen := make(map[T]struct{}, len(value))
		for _, item := range value {
			if _, ok := seen[item]; ok {
				return fmt.Errorf("不能包含重复项 %v", item)
			}
			seen[item] = struct{}{}
		}
		return nil
	}
}

// Each applies rules to every element and reports the 1-based index of the
// first invalid element, for example "第 2 项不能为空".
func Each[T any](rules ...Rule[T]) Rule[[]T] {
	return func(value []T) error {
		for index, item := range value {
			for _, rule := range rules {
				if rule == nil {
					continue
				}
				if err := rule(item); err != nil {
					return fmt.Errorf("第 %d 项%s", index+1, err.Error())
				}
			}
		}
		return nil
	}
}
