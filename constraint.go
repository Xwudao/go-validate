package validate

import (
	"errors"
	"fmt"
	"math"
	"net/mail"
	"slices"
	"unicode/utf8"
)

// Meta is JSON-Schema-neutral metadata describing one Constraint.
//
// It deliberately avoids any HTTP or OpenAPI dependency. The field names mirror
// the vocabulary shared by JSON Schema and OpenAPI (minLength, maxLength,
// minimum, maximum, format, enum) so any consumer can render them without this
// package importing that consumer.
//
// Every constructor builds the rule and the metadata from the same arguments,
// so the two cannot drift apart.
type Meta struct {
	// Required is true when the restriction rejects the zero value of its
	// type, i.e. it requires the field to carry a value. A value-typed field
	// turns this into JSON Schema "required", because after JSON binding an
	// absent property is indistinguishable from its zero value. An optional
	// pointer field skips nil entirely, so there Required is only produced by
	// SpecField.Required.
	Required bool

	// MinLength and MaxLength apply to JSON strings. They count Unicode code
	// points, matching utf8.RuneCountInString and JSON Schema's minLength.
	MinLength *int
	MaxLength *int

	// Minimum and Maximum apply to JSON numbers. They hold the exact Go value
	// passed to the constructor (int or float64), never a lossy float64
	// conversion, so large integer bounds survive.
	Minimum any
	Maximum any

	// Format is a JSON Schema format hint such as "email".
	Format string

	// Enum lists the allowed values.
	Enum []any

	// Unsupported is non-empty when the constraint has no machine-readable
	// representation. Schema export reports it as a hard error instead of
	// silently dropping or misdocumenting the restriction.
	Unsupported string

	// Invalid is non-empty when the constructor was given parameters that
	// cannot describe a valid restriction, for example a negative length
	// bound or a non-finite numeric bound. Schema export reports it as a hard
	// error instead of emitting an invalid JSON Schema.
	Invalid string
}

// Constraint pairs a runtime Rule with the metadata that describes the very
// same restriction. Constructors build both from one set of arguments, so
// validation and documentation cannot drift.
type Constraint[T any] struct {
	rule Rule[T]
	meta Meta
}

// Rule returns the runtime rule. It is nil only for the zero Constraint.
func (c Constraint[T]) Rule() Rule[T] { return c.rule }

// Meta returns a defensive copy of the machine-readable metadata. The copy
// owns its *int bounds and its Enum slice, so mutating the result cannot change
// the constraint, and Schema output cannot drift from the runtime Rule.
func (c Constraint[T]) Meta() Meta {
	meta := c.meta
	meta.MinLength = cloneInt(meta.MinLength)
	meta.MaxLength = cloneInt(meta.MaxLength)
	meta.Enum = cloneAny(meta.Enum)
	return meta
}

// NonEmpty rejects the empty string. It is the schema-aware counterpart of
// Required: value-typed fields advertise both required: true and minLength: 1.
func NonEmpty() Constraint[string] {
	return Constraint[string]{
		rule: func(value string) error {
			if value == "" {
				return errors.New("不能为空")
			}
			return nil
		},
		meta: Meta{Required: true, MinLength: intPtr(1)},
	}
}

// MinRunes rejects strings whose Unicode code point count is smaller than min.
// Metadata: minLength. When min > 0 the empty string is rejected as well.
//
// A negative min cannot describe a valid minLength, so the constraint fails
// closed: its rule rejects every value and Schema export reports it as invalid.
func MinRunes(min int) Constraint[string] {
	if min < 0 {
		return invalid[string](fmt.Sprintf("MinRunes requires min >= 0, got %d", min))
	}
	return Constraint[string]{
		rule: func(value string) error {
			if utf8.RuneCountInString(value) < min {
				return fmt.Errorf("长度不能小于 %d", min)
			}
			return nil
		},
		meta: Meta{Required: min > 0, MinLength: intPtr(min)},
	}
}

// MaxRunes rejects strings whose Unicode code point count is greater than max.
// Metadata: maxLength. A non-negative max accepts the empty string, so the
// field stays optional.
//
// A negative max cannot describe a valid maxLength, so the constraint fails
// closed: its rule rejects every value and Schema export reports it as invalid.
func MaxRunes(max int) Constraint[string] {
	if max < 0 {
		return invalid[string](fmt.Sprintf("MaxRunes requires max >= 0, got %d", max))
	}
	return Constraint[string]{
		rule: func(value string) error {
			if utf8.RuneCountInString(value) > max {
				return fmt.Errorf("长度不能大于 %d", max)
			}
			return nil
		},
		meta: Meta{MaxLength: intPtr(max)},
	}
}

// EmailFormat accepts a single valid RFC 5322 address and documents it as
// format: email. An empty string is rejected, so value-typed fields become
// required.
func EmailFormat() Constraint[string] {
	return Constraint[string]{
		rule: func(value string) error {
			address, err := mail.ParseAddress(value)
			if err != nil || address.Address != value {
				return errors.New("格式不正确")
			}
			return nil
		},
		meta: Meta{Required: true, Format: "email"},
	}
}

// Enum accepts only the listed values and documents them as a JSON Schema
// enum. When the zero value is not part of the set the field is required for
// value-typed fields.
//
// An empty value set cannot describe a valid enum, so the constraint fails
// closed: its rule rejects every value and Schema export reports it as invalid.
func Enum[T comparable](values ...T) Constraint[T] {
	if len(values) == 0 {
		return invalid[T]("Enum requires at least one allowed value")
	}
	allowed := slices.Clone(values)
	var zero T
	return Constraint[T]{
		rule: func(value T) error {
			if slices.Contains(allowed, value) {
				return nil
			}
			return errors.New("值不合法")
		},
		meta: Meta{Required: !slices.Contains(allowed, zero), Enum: anySlice(allowed)},
	}
}

// numeric is the set of Go numeric types accepted by MinValue and MaxValue. It
// matches the types used by the Int and Float field constructors, which keeps
// the metadata a single, comparable JSON number.
type numeric interface {
	int | float64
}

// MinValue rejects numeric values smaller than min and documents it as
// minimum. A non-finite min (NaN or ±Inf) is not a valid JSON number, so the
// constraint fails closed and Schema export reports it as invalid.
func MinValue[T numeric](min T) Constraint[T] {
	if isNonFinite(min) {
		return invalid[T]("MinValue requires a finite number")
	}
	var zero T
	return Constraint[T]{
		rule: func(value T) error {
			if value < min {
				return fmt.Errorf("不能小于 %v", min)
			}
			return nil
		},
		meta: Meta{Required: zero < min, Minimum: min},
	}
}

// MaxValue rejects numeric values greater than max and documents it as
// maximum. A non-finite max (NaN or ±Inf) is not a valid JSON number, so the
// constraint fails closed and Schema export reports it as invalid.
func MaxValue[T numeric](max T) Constraint[T] {
	if isNonFinite(max) {
		return invalid[T]("MaxValue requires a finite number")
	}
	var zero T
	return Constraint[T]{
		rule: func(value T) error {
			if value > max {
				return fmt.Errorf("不能大于 %v", max)
			}
			return nil
		},
		meta: Meta{Required: zero > max, Maximum: max},
	}
}

// WithMessage replaces the constraint's error text while preserving its
// metadata and runtime logic. It is the metadata-safe counterpart of Message.
func WithMessage[T any](message string, constraint Constraint[T]) Constraint[T] {
	inner := constraint
	return Constraint[T]{
		rule: func(value T) error {
			if inner.rule == nil || inner.rule(value) == nil {
				return nil
			}
			return errors.New(message)
		},
		meta: inner.meta,
	}
}

// Conditional applies constraint only when condition is true. Because the
// restriction is dynamic it has no unconditional machine-readable form:
// metadata is marked unsupported and Schema export fails with a clear error
// instead of silently documenting it as always applied.
func Conditional[T any](condition bool, constraint Constraint[T]) Constraint[T] {
	inner := constraint
	return Constraint[T]{
		rule: func(value T) error {
			if !condition || inner.rule == nil {
				return nil
			}
			return inner.rule(value)
		},
		meta: Meta{Unsupported: "conditional constraint has no unconditional machine-readable form"},
	}
}

// Custom adapts a plain rule. It validates at runtime but carries no metadata,
// so Schema export reports it as unsupported rather than silently dropping it.
func Custom[T any](rule Rule[T]) Constraint[T] {
	return Constraint[T]{
		rule: rule,
		meta: Meta{Unsupported: "custom rule has no machine-readable metadata; document it explicitly"},
	}
}

// invalid builds a Constraint whose parameters cannot describe a valid
// restriction. Its rule refuses every value (fail closed) and its metadata
// carries the reason so Schema export returns a hard error instead of emitting
// an invalid JSON Schema.
func invalid[T any](reason string) Constraint[T] {
	return Constraint[T]{
		rule: func(T) error { return errors.New("约束参数不合法: " + reason) },
		meta: Meta{Invalid: reason},
	}
}

// isNonFinite reports whether value is a NaN or an infinite float. Integer
// bounds are always finite, so it is false for them.
func isNonFinite[T numeric](value T) bool {
	number, ok := any(value).(float64)
	return ok && (math.IsNaN(number) || math.IsInf(number, 0))
}

func intPtr(value int) *int { return &value }

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneAny(values []any) []any {
	if values == nil {
		return nil
	}
	return slices.Clone(values)
}

func anySlice[T any](values []T) []any {
	result := make([]any, len(values))
	for i, value := range values {
		result[i] = value
	}
	return result
}
