package validate

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// JSON Schema type names used by FieldSchema.Type.
const (
	TypeString  = "string"
	TypeInteger = "integer"
	TypeNumber  = "number"
)

// FieldSchema is the static, JSON-Schema-neutral description of one field.
//
// It is produced by Spec.Schema without running any rule, so exporting field
// metadata never depends on a request value. Field names are the JSON property
// names; a consumer can compare Schema.Names against the payload shape.
type FieldSchema struct {
	Name     string
	Type     string
	Required bool
	Nullable bool

	// MinLength and MaxLength count Unicode code points.
	MinLength *int
	MaxLength *int

	// Minimum and Maximum hold the exact Go numeric bound (int or float64).
	Minimum any
	Maximum any

	// Format is a JSON Schema format hint such as "email".
	Format string

	// Enum lists the allowed values.
	Enum []any

	// Unsupported lists the constraints that could not be represented. A
	// non-empty list makes Schema.Err non-nil.
	Unsupported []string

	// Invalid lists the constraints whose parameters are invalid and any Spec
	// integrity problem such as an empty or duplicate field name. A non-empty
	// list makes Schema.Err non-nil.
	Invalid []string
}

// SpecField is one field's bound value, its runtime rules and its static
// metadata. It is created by String/Int/Float (and their Optional variants) and
// collected into a Spec. The unexported validate method makes it usable with
// Validate.
type SpecField struct {
	meta            FieldSchema
	rules           []func() error
	presence        func() error
	presenceApplied bool
}

// String declares a value-typed string field. After JSON binding the absence
// and the zero value of a value-typed field are indistinguishable, so a
// constraint that rejects "" (for example NonEmpty) also makes the property
// required in the schema.
func String(name, value string, constraints ...Constraint[string]) SpecField {
	return newValueField(name, TypeString, value, constraints)
}

// OptionalString declares a nullable string field. A nil value means the
// property was absent or null; every constraint is then skipped, so the
// property is optional unless Required is called.
func OptionalString(name string, value *string, constraints ...Constraint[string]) SpecField {
	return newOptionalField(name, TypeString, value, constraints)
}

// Int declares a value-typed integer field. See String for how required is
// derived from the zero value.
func Int(name string, value int, constraints ...Constraint[int]) SpecField {
	return newValueField(name, TypeInteger, value, constraints)
}

// OptionalInt declares a nullable integer field. See OptionalString.
func OptionalInt(name string, value *int, constraints ...Constraint[int]) SpecField {
	return newOptionalField(name, TypeInteger, value, constraints)
}

// Float declares a value-typed floating point field. See String for how
// required is derived from the zero value.
func Float(name string, value float64, constraints ...Constraint[float64]) SpecField {
	return newValueField(name, TypeNumber, value, constraints)
}

// OptionalFloat declares a nullable floating point field. See OptionalString.
func OptionalFloat(name string, value *float64, constraints ...Constraint[float64]) SpecField {
	return newOptionalField(name, TypeNumber, value, constraints)
}

// Required marks the field as required to be present and prepends a presence
// rule: a value-typed field rejects its zero value, an optional field rejects
// nil. It returns a copy, so the result must be used:
//
//	validate.String("name", name).Required()
//
// A required optional field is no longer nullable, because nil is rejected.
func (f SpecField) Required() SpecField {
	f.meta.Required = true
	f.meta.Nullable = false
	if f.presence != nil && !f.presenceApplied {
		f.rules = append([]func() error{f.presence}, f.rules...)
		f.presenceApplied = true
	}
	return f
}

func (f SpecField) validate() *FieldError {
	for _, rule := range f.rules {
		if rule == nil {
			continue
		}
		if err := rule(); err != nil {
			return &FieldError{Field: f.meta.Name, Message: err.Error()}
		}
	}
	return nil
}

// merge folds one constraint's metadata into the field. Runtime enforces every
// constraint, so overlapping bounds and enums are tightened to the effective
// restriction instead of the last declaration winning.
func mergeConstraint(dst *FieldSchema, constraint Meta, inferRequired bool) {
	if inferRequired && constraint.Required {
		dst.Required = true
	}
	if constraint.MinLength != nil {
		if dst.MinLength == nil || *constraint.MinLength > *dst.MinLength {
			dst.MinLength = constraint.MinLength
		}
	}
	if constraint.MaxLength != nil {
		if dst.MaxLength == nil || *constraint.MaxLength < *dst.MaxLength {
			dst.MaxLength = constraint.MaxLength
		}
	}
	if constraint.Minimum != nil {
		if dst.Minimum == nil {
			dst.Minimum = constraint.Minimum
		} else {
			dst.Minimum = tightenMinimum(dst.Minimum, constraint.Minimum)
		}
	}
	if constraint.Maximum != nil {
		if dst.Maximum == nil {
			dst.Maximum = constraint.Maximum
		} else {
			dst.Maximum = tightenMaximum(dst.Maximum, constraint.Maximum)
		}
	}
	if constraint.Format != "" {
		dst.Format = constraint.Format
	}
	if constraint.Enum != nil {
		if dst.Enum == nil {
			dst.Enum = constraint.Enum
		} else {
			dst.Enum = intersection(dst.Enum, constraint.Enum)
		}
	}
	if constraint.Unsupported != "" && !slices.Contains(dst.Unsupported, constraint.Unsupported) {
		dst.Unsupported = append(dst.Unsupported, constraint.Unsupported)
	}
	if constraint.Invalid != "" && !slices.Contains(dst.Invalid, constraint.Invalid) {
		dst.Invalid = append(dst.Invalid, constraint.Invalid)
	}
	// Overlapping bounds are tightened above. When the effective minimum is
	// greater than the effective maximum no value can ever satisfy the field, so
	// the declaration is inconsistent and must be reported instead of emitting a
	// schema that can never validate.
	if dst.MinLength != nil && dst.MaxLength != nil && *dst.MinLength > *dst.MaxLength {
		addInvalid(dst, fmt.Sprintf("inconsistent length bounds: minLength %d > maxLength %d", *dst.MinLength, *dst.MaxLength))
	}
	if dst.Minimum != nil && dst.Maximum != nil && compareNumbers(dst.Minimum, dst.Maximum) > 0 {
		addInvalid(dst, fmt.Sprintf("inconsistent numeric bounds: minimum %v > maximum %v", dst.Minimum, dst.Maximum))
	}
}

// addInvalid records a field-level declaration problem once.
func addInvalid(dst *FieldSchema, reason string) {
	if !slices.Contains(dst.Invalid, reason) {
		dst.Invalid = append(dst.Invalid, reason)
	}
}

func newValueField[T comparable](name, typeName string, value T, constraints []Constraint[T]) SpecField {
	field := SpecField{meta: FieldSchema{Name: name, Type: typeName}}
	for _, constraint := range constraints {
		mergeConstraint(&field.meta, constraint.meta, true)
		if constraint.rule != nil {
			rule := constraint.rule
			bound := value
			field.rules = append(field.rules, func() error { return rule(bound) })
		}
	}
	field.presence = func() error {
		var zero T
		if value == zero {
			return errors.New("不能为空")
		}
		return nil
	}
	return field
}

func newOptionalField[T any](name, typeName string, value *T, constraints []Constraint[T]) SpecField {
	field := SpecField{meta: FieldSchema{Name: name, Type: typeName, Nullable: true}}
	for _, constraint := range constraints {
		// A constraint never makes an optional property required: nil is
		// skipped at runtime, so absence stays valid.
		mergeConstraint(&field.meta, constraint.meta, false)
		if constraint.rule != nil && value != nil {
			rule := constraint.rule
			bound := *value
			field.rules = append(field.rules, func() error { return rule(bound) })
		}
	}
	field.presence = func() error {
		if value == nil {
			return errors.New("不能为空")
		}
		return nil
	}
	return field
}

// Spec is the ordered list of fields and constraints declared by one DTO
// method. The same value yields runtime validation (Validate) and static
// metadata (Schema).
type Spec []SpecField

// Validate runs every field's rules against the bound values and aggregates one
// error per invalid field. It returns nil when the values are valid.
func (s Spec) Validate() error {
	fields := make([]fieldResult, 0, len(s))
	for i := range s {
		fields = append(fields, s[i])
	}
	return Validate(fields...)
}

// Schema exports the static field metadata without running any rule, so it is
// safe to call on a zero value. Every returned FieldSchema is a defensive copy
// that owns its bounds, enum and reason slices, so a consumer cannot mutate the
// Spec or make later exports drift from the runtime rules.
//
// It returns a non-nil error when any field cannot be represented faithfully:
// an UnsupportedError for constraints with no machine-readable form, or an
// InvalidError for invalid constraint parameters, inconsistent/overlapping
// bounds and empty or duplicate field names. The returned schema still contains
// the metadata of every field so a consumer can decide how to proceed.
func (s Spec) Schema() (Schema, error) {
	schema := Schema{Fields: make([]FieldSchema, 0, len(s))}
	seen := make(map[string]bool, len(s))
	for _, field := range s {
		meta := cloneFieldSchema(field.meta)
		switch {
		case meta.Name == "":
			meta.Invalid = append(meta.Invalid, "field name must not be empty")
		case seen[meta.Name]:
			meta.Invalid = append(meta.Invalid, fmt.Sprintf("duplicate field name %q", meta.Name))
		}
		seen[meta.Name] = true
		schema.Fields = append(schema.Fields, meta)
	}
	return schema, schema.Err()
}

// cloneFieldSchema deep-copies every reference-typed member so callers cannot
// reach back into the Spec through a returned FieldSchema.
func cloneFieldSchema(field FieldSchema) FieldSchema {
	clone := field
	clone.MinLength = cloneInt(field.MinLength)
	clone.MaxLength = cloneInt(field.MaxLength)
	clone.Enum = cloneAny(field.Enum)
	clone.Unsupported = slices.Clone(field.Unsupported)
	clone.Invalid = slices.Clone(field.Invalid)
	return clone
}

// Schema is the static description of a whole Spec.
type Schema struct {
	Fields []FieldSchema
}

// Names returns the field names in declaration order. A consumer compares
// these against the JSON property names of the request.
func (s Schema) Names() []string {
	names := make([]string, 0, len(s.Fields))
	for _, field := range s.Fields {
		names = append(names, field.Name)
	}
	return names
}

// Field returns the metadata for the named field, or nil when it is absent.
func (s Schema) Field(name string) *FieldSchema {
	for i := range s.Fields {
		if s.Fields[i].Name == name {
			return &s.Fields[i]
		}
	}
	return nil
}

// Err returns a non-nil error listing every field that cannot be represented
// faithfully: an UnsupportedError for constraints with no machine-readable
// form, or an InvalidError for invalid parameters, inconsistent bounds and
// empty or duplicate field names. When both kinds are present the returned
// error wraps both and matches errors.As for each.
func (s Schema) Err() error {
	var (
		unsupported UnsupportedError
		invalid     InvalidError
	)
	for _, field := range s.Fields {
		for _, reason := range field.Unsupported {
			unsupported = append(unsupported, Unsupported{Field: field.Name, Reason: reason})
		}
		for _, reason := range field.Invalid {
			invalid = append(invalid, Invalid{Field: field.Name, Reason: reason})
		}
	}
	errs := make([]error, 0, 2)
	if len(unsupported) > 0 {
		errs = append(errs, unsupported)
	}
	if len(invalid) > 0 {
		errs = append(errs, invalid)
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

// Unsupported identifies a field whose constraint cannot be represented in the
// machine-readable schema.
type Unsupported struct {
	Field  string
	Reason string
}

func (u Unsupported) String() string { return u.Field + ": " + u.Reason }

// UnsupportedError aggregates every unsupported constraint found while
// exporting a schema.
type UnsupportedError []Unsupported

func (e UnsupportedError) Error() string {
	parts := make([]string, 0, len(e))
	for _, unsupported := range e {
		parts = append(parts, unsupported.String())
	}
	return "unsupported constraints: " + strings.Join(parts, "; ")
}

// Invalid identifies a field whose declaration is invalid: a constraint with
// bad parameters, inconsistent bounds, or an empty or duplicate field name.
type Invalid struct {
	Field  string
	Reason string
}

func (i Invalid) String() string { return i.Field + ": " + i.Reason }

// InvalidError aggregates every invalid declaration found while exporting a
// schema.
type InvalidError []Invalid

func (e InvalidError) Error() string {
	parts := make([]string, 0, len(e))
	for _, invalid := range e {
		parts = append(parts, invalid.String())
	}
	return "invalid constraints: " + strings.Join(parts, "; ")
}

func tightenMinimum(current, candidate any) any {
	if compareNumbers(candidate, current) > 0 {
		return candidate
	}
	return current
}

func tightenMaximum(current, candidate any) any {
	if compareNumbers(candidate, current) < 0 {
		return candidate
	}
	return current
}

// compareNumbers compares two numeric bounds produced by MinValue/MaxValue. It
// only handles the exact types accepted by the numeric constraint, so the
// comparison is exact and never routes through float64.
func compareNumbers(a, b any) int {
	switch av := a.(type) {
	case int:
		if bv, ok := b.(int); ok {
			return cmp.Compare(av, bv)
		}
	case float64:
		if bv, ok := b.(float64); ok {
			return cmp.Compare(av, bv)
		}
	}
	return 0
}

func intersection(a, b []any) []any {
	result := make([]any, 0, len(a))
	for _, item := range a {
		if slices.Contains(b, item) {
			result = append(result, item)
		}
	}
	return result
}
