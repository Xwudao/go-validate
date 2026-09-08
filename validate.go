// Package validate provides small, code-first validation primitives for
// application input values.
//
// Design goals:
//
//   - No struct tags and no reflection. Rules are ordinary Go functions, so the
//     compiler checks every call and refactoring is safe.
//   - No dependencies on HTTP frameworks, databases or application services.
//     Rules are pure: they only inspect the value passed to them.
//   - Extensible. Rule[T] is an exported function type, so any package that
//     depends on this module can add its own rules without forking the library.
//
// Basic usage:
//
//	err := validate.Validate(
//	    validate.Field("email", email,
//	        validate.Message("邮箱不能为空", validate.Required()),
//	        validate.Message("邮箱格式不正确", validate.Email()),
//	    ),
//	    validate.Field("age", age,
//	        validate.Min(18),
//	        validate.Max(120),
//	    ),
//	)
//
// Custom rules are plain functions and can live in the consuming project:
//
//	// internal/validatex/even.go
//	func Even() validate.Rule[int] {
//	    return func(value int) error {
//	        if value%2 != 0 {
//	            return errors.New("必须是偶数")
//	        }
//	        return nil
//	    }
//	}
//
//	validate.Field("count", count, validatex.Even())
package validate

import (
	"errors"
	"strings"
)

// Rule validates one value and returns a human readable error when the value is
// invalid. A nil error means the value passed validation.
//
// Rule is deliberately an exported function type: any package can define its
// own rules and mix them with the ones provided by this library.
type Rule[T any] func(T) error

// FieldError identifies one invalid input field.
type FieldError struct {
	Field   string
	Message string
}

func (e FieldError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

// Errors aggregates at most the first failing rule for every invalid field.
type Errors []FieldError

func (e Errors) Error() string {
	messages := make([]string, 0, len(e))
	for _, fieldErr := range e {
		messages = append(messages, fieldErr.Error())
	}
	return strings.Join(messages, "; ")
}

// First returns the first field error message without the field name. HTTP
// adapters can use it to preserve APIs that historically return a single
// validation message.
func (e Errors) First() error {
	if len(e) == 0 {
		return nil
	}
	return errors.New(e[0].Message)
}

// For returns the first error recorded for the named field, or nil when the
// field is valid.
func (e Errors) For(field string) *FieldError {
	for i := range e {
		if e[i].Field == field {
			return &e[i]
		}
	}
	return nil
}

// Has reports whether the named field failed validation.
func (e Errors) Has(field string) bool {
	return e.For(field) != nil
}

// Fields returns the names of every invalid field, in declaration order.
func (e Errors) Fields() []string {
	fields := make([]string, 0, len(e))
	for _, fieldErr := range e {
		fields = append(fields, fieldErr.Field)
	}
	return fields
}

// Messages returns every error message without field names, in declaration
// order.
func (e Errors) Messages() []string {
	messages := make([]string, 0, len(e))
	for _, fieldErr := range e {
		messages = append(messages, fieldErr.Message)
	}
	return messages
}

// fieldResult is implemented by Field. The validate method is unexported on
// purpose so that only this package can satisfy the interface.
type fieldResult interface {
	validate() *FieldError
}

type field[T any] struct {
	name  string
	value T
	rules []Rule[T]
}

// Field declares a named value and the rules that must all pass. Rules run in
// declaration order and stop at the first failure for that field.
//
// Field returns an unexported type: callers pass the result straight to
// Validate and never need to name the type themselves.
func Field[T any](name string, value T, rules ...Rule[T]) field[T] {
	return field[T]{name: name, value: value, rules: rules}
}

func (f field[T]) validate() *FieldError {
	for _, rule := range f.rules {
		if rule == nil {
			continue
		}
		if err := rule(f.value); err != nil {
			return &FieldError{Field: f.name, Message: err.Error()}
		}
	}
	return nil
}

// Validate runs every field and aggregates one error per invalid field. It
// returns nil when all fields are valid.
func Validate(fields ...fieldResult) error {
	var result Errors
	for _, f := range fields {
		if err := f.validate(); err != nil {
			result = append(result, *err)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// When applies rule only when condition is true. A false condition makes the
// field valid, which is useful for conditionally required fields.
func When[T any](condition bool, rule Rule[T]) Rule[T] {
	return func(value T) error {
		if !condition || rule == nil {
			return nil
		}
		return rule(value)
	}
}

// Message replaces a rule's error text while preserving its validation logic.
// It is commonly used to translate the built-in Chinese messages or to give a
// field specific wording.
func Message[T any](message string, rule Rule[T]) Rule[T] {
	return func(value T) error {
		if rule == nil || rule(value) == nil {
			return nil
		}
		return errors.New(message)
	}
}

// Optional skips rule for nil pointers and applies it to a supplied value. It
// turns any rule into an "only validate when present" rule.
func Optional[T any](rule Rule[T]) Rule[*T] {
	return func(value *T) error {
		if value == nil || rule == nil {
			return nil
		}
		return rule(*value)
	}
}

// Func adapts a plain function to a Rule. It is a convenience for one-off
// custom rules that do not need their own named constructor:
//
//	validate.Field("code", code, validate.Func(func(value string) error {
//	    if len(value) != 6 {
//	        return errors.New("验证码必须是 6 位")
//	    }
//	    return nil
//	}))
func Func[T any](fn func(T) error) Rule[T] {
	return Rule[T](fn)
}

// All applies rules in order and returns the first error. It is useful when a
// set of rules needs to be reused as a single rule.
func All[T any](rules ...Rule[T]) Rule[T] {
	return func(value T) error {
		for _, rule := range rules {
			if rule == nil {
				continue
			}
			if err := rule(value); err != nil {
				return err
			}
		}
		return nil
	}
}

// Any passes when at least one rule passes. When every rule fails it returns
// the first error, which keeps the message stable and predictable.
func Any[T any](rules ...Rule[T]) Rule[T] {
	return func(value T) error {
		var first error
		for _, rule := range rules {
			if rule == nil {
				continue
			}
			err := rule(value)
			if err == nil {
				return nil
			}
			if first == nil {
				first = err
			}
		}
		return first
	}
}

// Not inverts rule: it fails with message when rule passes.
func Not[T any](rule Rule[T], message string) Rule[T] {
	return func(value T) error {
		if rule == nil {
			return nil
		}
		if rule(value) == nil {
			return errors.New(message)
		}
		return nil
	}
}
