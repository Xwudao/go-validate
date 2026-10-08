package validate

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

// createUserRequest is the shape a consumer DTO takes: every constraint is
// declared once in Spec, and Validate/Schema are derived from it.
type createUserRequest struct {
	Name  string
	Email string
	Age   int
	Role  string
	Bio   *string
}

func (r createUserRequest) Spec() Spec {
	return Spec{
		String("name", r.Name, NonEmpty(), MaxRunes(50)),
		String("email", r.Email, WithMessage("邮箱格式不正确", EmailFormat())),
		Int("age", r.Age, MinValue(18), MaxValue(120)),
		String("role", r.Role, Enum("admin", "member")),
		OptionalString("bio", r.Bio, MaxRunes(200)),
	}
}

func TestSpecValidationMatchesMetadata(t *testing.T) {
	t.Parallel()

	valid := createUserRequest{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   30,
		Role:  "admin",
	}
	if err := valid.Spec().Validate(); err != nil {
		t.Fatalf("valid request error = %v, want nil", err)
	}

	bio := "hello"
	validWithBio := valid
	validWithBio.Bio = &bio
	if err := validWithBio.Spec().Validate(); err != nil {
		t.Fatalf("valid request with bio error = %v, want nil", err)
	}

	schema, err := createUserRequest{}.Spec().Schema()
	if err != nil {
		t.Fatalf("Schema() error = %v, want nil", err)
	}
	if got := schema.Names(); !reflect.DeepEqual(got, []string{"name", "email", "age", "role", "bio"}) {
		t.Fatalf("Names() = %v", got)
	}

	name := schema.Field("name")
	if name == nil || name.Type != TypeString || !name.Required || name.Nullable {
		t.Fatalf("name meta = %#v", name)
	}
	if name.MinLength == nil || *name.MinLength != 1 || name.MaxLength == nil || *name.MaxLength != 50 {
		t.Fatalf("name length meta = %#v", name)
	}

	email := schema.Field("email")
	if email == nil || email.Format != "email" || !email.Required {
		t.Fatalf("email meta = %#v", email)
	}

	age := schema.Field("age")
	if age == nil || age.Type != TypeInteger || age.Minimum != 18 || age.Maximum != 120 || !age.Required {
		t.Fatalf("age meta = %#v", age)
	}

	role := schema.Field("role")
	if role == nil || !role.Required || !reflect.DeepEqual(role.Enum, []any{"admin", "member"}) {
		t.Fatalf("role meta = %#v", role)
	}

	bioMeta := schema.Field("bio")
	if bioMeta == nil || bioMeta.Required || !bioMeta.Nullable {
		t.Fatalf("bio meta = %#v", bioMeta)
	}
	if bioMeta.MaxLength == nil || *bioMeta.MaxLength != 200 {
		t.Fatalf("bio length meta = %#v", bioMeta)
	}
}

func TestSpecValidationAggregatesAndKeepsErrorTexts(t *testing.T) {
	t.Parallel()

	invalid := createUserRequest{
		Name:  "",
		Email: "not-an-email",
		Age:   10,
		Role:  "guest",
	}
	err := invalid.Spec().Validate()

	var validationErrors Errors
	if !errors.As(err, &validationErrors) {
		t.Fatalf("error type = %T, want Errors", err)
	}
	want := []string{
		"name: 不能为空",
		"email: 邮箱格式不正确",
		"age: 不能小于 18",
		"role: 值不合法",
	}
	got := make([]string, 0, len(validationErrors))
	for _, fieldErr := range validationErrors {
		got = append(got, fieldErr.Error())
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("errors = %v, want %v", got, want)
	}
}

func TestSpecConstraintErrorTexts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{"non empty", Spec{String("s", "", NonEmpty())}.Validate(), "s: 不能为空"},
		{"min runes", Spec{String("s", "a", MinRunes(2))}.Validate(), "s: 长度不能小于 2"},
		{"max runes", Spec{String("s", "abcd", MaxRunes(3))}.Validate(), "s: 长度不能大于 3"},
		{"email", Spec{String("s", "bad", EmailFormat())}.Validate(), "s: 格式不正确"},
		{"enum", Spec{String("s", "x", Enum("a", "b"))}.Validate(), "s: 值不合法"},
		{"min value", Spec{Int("n", 1, MinValue(2))}.Validate(), "n: 不能小于 2"},
		{"max value", Spec{Int("n", 5, MaxValue(4))}.Validate(), "n: 不能大于 4"},
		{"float min", Spec{Float("n", 0.5, MinValue(1.5))}.Validate(), "n: 不能小于 1.5"},
		{"required value field", Spec{String("s", "", NonEmpty()).Required()}.Validate(), "s: 不能为空"},
		{"required optional field", Spec{OptionalString("s", nil).Required()}.Validate(), "s: 不能为空"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil || tt.err.Error() != tt.want {
				t.Fatalf("error = %v, want %q", tt.err, tt.want)
			}
		})
	}
}

func TestSpecUnicodeLengthUsesRunes(t *testing.T) {
	t.Parallel()

	// "你好" is 2 runes but 6 bytes; min/max length must count runes.
	if err := (Spec{String("s", "你好", MinRunes(2))}).Validate(); err != nil {
		t.Fatalf("MinRunes(2) error = %v, want nil", err)
	}
	if err := (Spec{String("s", "你好", MinRunes(3))}).Validate(); err == nil || err.Error() != "s: 长度不能小于 3" {
		t.Fatalf("MinRunes(3) error = %v", err)
	}
	if err := (Spec{String("s", "你好世界", MaxRunes(3))}).Validate(); err == nil || err.Error() != "s: 长度不能大于 3" {
		t.Fatalf("MaxRunes(3) error = %v", err)
	}

	schema, err := (Spec{String("s", "你好", MinRunes(2), MaxRunes(4))}).Schema()
	if err != nil {
		t.Fatalf("Schema() error = %v", err)
	}
	if *schema.Field("s").MinLength != 2 || *schema.Field("s").MaxLength != 4 {
		t.Fatalf("length metadata = %#v", schema.Field("s"))
	}
}

func TestWithMessagePreservesMetadata(t *testing.T) {
	t.Parallel()

	constraint := WithMessage("邮箱格式不正确", EmailFormat())
	if got := constraint.Meta(); got.Format != "email" || !got.Required {
		t.Fatalf("WithMessage metadata = %#v", got)
	}
	if err := (Spec{String("s", "bad", constraint)}).Validate(); err == nil || err.Error() != "s: 邮箱格式不正确" {
		t.Fatalf("WithMessage error = %v", err)
	}
}

func TestConditionalIsUnsupportedButStillValidates(t *testing.T) {
	t.Parallel()

	schema, err := (Spec{String("company", "", Conditional(true, NonEmpty()))}).Schema()
	if err == nil {
		t.Fatalf("Schema() error = nil, want unsupported error")
	}
	var unsupported UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error type = %T, want UnsupportedError", err)
	}
	if unsupported[0].Field != "company" {
		t.Fatalf("unsupported field = %q", unsupported[0].Field)
	}
	// The metadata must not pretend the conditional rule is unconditional.
	if field := schema.Field("company"); field.Required || field.MinLength != nil {
		t.Fatalf("conditional metadata leaked: %#v", field)
	}

	if err := (Spec{String("company", "", Conditional(true, NonEmpty()))}).Validate(); err == nil || err.Error() != "company: 不能为空" {
		t.Fatalf("conditional(true) error = %v", err)
	}
	if err := (Spec{String("company", "", Conditional(false, NonEmpty()))}).Validate(); err != nil {
		t.Fatalf("conditional(false) error = %v, want nil", err)
	}
}

func TestCustomRuleIsUnsupportedButStillValidates(t *testing.T) {
	t.Parallel()

	even := Custom(func(value int) error {
		if value%2 != 0 {
			return errors.New("必须是偶数")
		}
		return nil
	})

	schema, err := (Spec{Int("n", 3, even)}).Schema()
	if err == nil {
		t.Fatalf("Schema() error = nil, want unsupported error")
	}
	var unsupported UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error type = %T, want UnsupportedError", err)
	}
	if unsupported[0].Field != "n" {
		t.Fatalf("unsupported field = %q", unsupported[0].Field)
	}
	if field := schema.Field("n"); len(field.Unsupported) != 1 {
		t.Fatalf("unsupported metadata = %#v", field)
	}

	if err := (Spec{Int("n", 4, even)}).Validate(); err != nil {
		t.Fatalf("custom rule error = %v, want nil", err)
	}
	if err := (Spec{Int("n", 3, even)}).Validate(); err == nil || err.Error() != "n: 必须是偶数" {
		t.Fatalf("custom rule error = %v", err)
	}
}

func TestSchemaExportDoesNotRunRules(t *testing.T) {
	t.Parallel()

	runs := 0
	rule := Custom(func(value int) error {
		runs++
		return nil
	})

	// Zero-value receiver: no request values, but metadata export must be
	// static and must not execute any rule.
	schema, err := (Spec{Int("n", 0, MinValue(18), rule)}).Schema()
	if err == nil {
		t.Fatalf("Schema() error = nil, want unsupported error from custom rule")
	}
	if runs != 0 {
		t.Fatalf("rules executed during Schema export: runs = %d", runs)
	}
	if schema.Field("n").Minimum != 18 {
		t.Fatalf("supported metadata lost: %#v", schema.Field("n"))
	}
}

func TestRequiredIsInferredFromRejectedZeroValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		field SpecField
		want  bool
	}{
		{"non empty", String("s", "", NonEmpty()), true},
		{"min runes positive", String("s", "", MinRunes(1)), true},
		{"min runes zero", String("s", "", MinRunes(0)), false},
		{"max runes", String("s", "", MaxRunes(5)), false},
		{"email", String("s", "", EmailFormat()), true},
		{"enum excludes zero", String("s", "", Enum("a", "b")), true},
		{"enum includes zero", String("s", "", Enum("", "a")), false},
		{"min value zero", Int("n", 0, MinValue(0)), false},
		{"min value positive", Int("n", 0, MinValue(1)), true},
		{"max value zero", Int("n", 0, MaxValue(0)), false},
		{"max value negative", Int("n", 0, MaxValue(-1)), true},
		{"plain", String("s", ""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := (Spec{tt.field}).Schema()
			if err != nil {
				t.Fatalf("Schema() error = %v", err)
			}
			if got := schema.Field(tt.field.meta.Name).Required; got != tt.want {
				t.Fatalf("Required = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAbsentVsEmptySemantics(t *testing.T) {
	t.Parallel()

	// A value field cannot distinguish absent from empty, so NonEmpty also
	// makes the property required.
	valueSchema, err := (Spec{String("name", "", NonEmpty())}).Schema()
	if err != nil {
		t.Fatalf("Schema() error = %v", err)
	}
	if !valueSchema.Field("name").Required {
		t.Fatalf("value field should be required")
	}

	// An optional pointer field keeps nil (absent/null) valid while still
	// constraining a present value.
	optional := Spec{OptionalString("nickname", nil, MinRunes(1))}
	optionalSchema, err := optional.Schema()
	if err != nil {
		t.Fatalf("Schema() error = %v", err)
	}
	nickname := optionalSchema.Field("nickname")
	if nickname.Required || !nickname.Nullable || *nickname.MinLength != 1 {
		t.Fatalf("optional metadata = %#v", nickname)
	}
	if err := optional.Validate(); err != nil {
		t.Fatalf("nil optional error = %v, want nil", err)
	}

	empty := ""
	if err := (Spec{OptionalString("nickname", &empty, MinRunes(1))}).Validate(); err == nil || err.Error() != "nickname: 长度不能小于 1" {
		t.Fatalf("present empty optional error = %v", err)
	}

	// Required on an optional field rejects nil and is no longer nullable.
	requiredSchema, err := (Spec{OptionalString("nickname", nil).Required()}).Schema()
	if err != nil {
		t.Fatalf("Schema() error = %v", err)
	}
	required := requiredSchema.Field("nickname")
	if !required.Required || required.Nullable {
		t.Fatalf("required optional metadata = %#v", required)
	}
}

func TestInvalidConstraintParametersFailClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		field     SpecField
		wantError string
	}{
		{
			"negative min runes",
			String("s", "abc", MinRunes(-1)),
			"s: 约束参数不合法: MinRunes requires min >= 0, got -1",
		},
		{
			"negative max runes",
			String("s", "abc", MaxRunes(-1)),
			"s: 约束参数不合法: MaxRunes requires max >= 0, got -1",
		},
		{
			"empty enum",
			String("s", "a", Enum[string]()),
			"s: 约束参数不合法: Enum requires at least one allowed value",
		},
		{
			"nan minimum",
			Float("n", 1, MinValue(math.NaN())),
			"n: 约束参数不合法: MinValue requires a finite number",
		},
		{
			"infinite maximum",
			Float("n", 1, MaxValue(math.Inf(1))),
			"n: 约束参数不合法: MaxValue requires a finite number",
		},
		{
			"negative infinite minimum",
			Float("n", 1, MinValue(math.Inf(-1))),
			"n: 约束参数不合法: MinValue requires a finite number",
		},
		{
			"inconsistent length bounds",
			String("s", "abc", MinRunes(5), MaxRunes(3)),
			"s: 长度不能小于 5",
		},
		{
			"inconsistent numeric bounds",
			Int("n", 4, MinValue(5), MaxValue(3)),
			"n: 不能小于 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spec := Spec{tt.field}
			if err := spec.Validate(); err == nil || err.Error() != tt.wantError {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantError)
			}

			schema, err := spec.Schema()
			if err == nil {
				t.Fatalf("Schema() error = nil, want invalid error")
			}
			var invalid InvalidError
			if !errors.As(err, &invalid) {
				t.Fatalf("error type = %T, want InvalidError", err)
			}
			field := schema.Field(tt.field.meta.Name)
			if field == nil || len(field.Invalid) == 0 {
				t.Fatalf("field invalid metadata = %#v", field)
			}
			// An invalid constraint must never masquerade as supported metadata.
			if field.MinLength != nil && *field.MinLength < 0 {
				t.Fatalf("negative minLength leaked into schema: %#v", field)
			}
			if field.MaxLength != nil && *field.MaxLength < 0 {
				t.Fatalf("negative maxLength leaked into schema: %#v", field)
			}
		})
	}
}

func TestInvalidConstraintOnOptionalFieldIsReported(t *testing.T) {
	t.Parallel()

	// nil keeps the documented optional semantics: absent values skip every
	// constraint, so this validates even though the declaration is invalid.
	absent := Spec{OptionalString("nickname", nil, MinRunes(-1))}
	if err := absent.Validate(); err != nil {
		t.Fatalf("nil optional error = %v, want nil", err)
	}

	// The invalid declaration is still a hard schema error so it cannot ship
	// silently.
	if _, err := absent.Schema(); err == nil {
		t.Fatalf("Schema() error = nil, want invalid error")
	}

	// A present value hits the fail-closed rule.
	value := "bob"
	present := Spec{OptionalString("nickname", &value, MinRunes(-1))}
	if err := present.Validate(); err == nil || err.Error() != "nickname: 约束参数不合法: MinRunes requires min >= 0, got -1" {
		t.Fatalf("present optional error = %v", err)
	}
}

func TestSchemaRejectsEmptyAndDuplicateFieldNames(t *testing.T) {
	t.Parallel()

	spec := Spec{
		String("", "a"),
		String("x", "b"),
		String("x", "c"),
	}

	// Name integrity is a declaration concern: valid values still validate.
	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}

	schema, err := spec.Schema()
	if err == nil {
		t.Fatalf("Schema() error = nil, want invalid error")
	}
	var invalid InvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("error type = %T, want InvalidError", err)
	}

	var sawEmpty, sawDuplicate bool
	for _, entry := range invalid {
		switch {
		case entry.Field == "" && strings.Contains(entry.Reason, "must not be empty"):
			sawEmpty = true
		case entry.Field == "x" && strings.Contains(entry.Reason, "duplicate"):
			sawDuplicate = true
		}
	}
	if !sawEmpty || !sawDuplicate {
		t.Fatalf("invalid entries = %#v, want empty and duplicate reasons", invalid)
	}
	// Neither invalid name may carry valid-looking constraint metadata.
	var sawEmptyField, sawDuplicateField bool
	for _, field := range schema.Fields {
		switch {
		case field.Name == "" && len(field.Invalid) > 0:
			sawEmptyField = true
		case field.Name == "x" && len(field.Invalid) > 0:
			sawDuplicateField = true
		}
	}
	if !sawEmptyField || !sawDuplicateField {
		t.Fatalf("schema fields = %#v, want invalid metadata on empty and duplicate names", schema.Fields)
	}

	// Duplicate names keep validating each declared field, which documents the
	// runtime behaviour instead of silently dropping one.
	if err := (Spec{String("x", "", NonEmpty()), String("x", "", NonEmpty())}).Validate(); err == nil {
		t.Fatalf("Validate() error = nil, want field errors")
	}
}

func TestMetadataAndSchemaAreDefensivelyCopied(t *testing.T) {
	t.Parallel()

	// Constraint.Meta must not expose the constraint's own *int or []any.
	constraint := MinRunes(2)
	meta := constraint.Meta()
	*meta.MinLength = 99
	if got := *constraint.Meta().MinLength; got != 2 {
		t.Fatalf("constraint MinLength after Meta mutation = %d, want 2", got)
	}

	enum := Enum("admin", "member")
	enumMeta := enum.Meta()
	enumMeta.Enum[0] = "root"
	if got := enum.Meta().Enum[0]; got != "admin" {
		t.Fatalf("constraint Enum after Meta mutation = %v, want admin", got)
	}

	// Schema output must not alias the Spec's metadata: mutating one export
	// cannot change later exports or the runtime rules.
	spec := Spec{String("role", "admin", MinRunes(2), Enum("admin", "member"))}
	first, err := spec.Schema()
	if err != nil {
		t.Fatalf("Schema() error = %v", err)
	}
	role := first.Field("role")
	*role.MinLength = 0
	role.Enum[0] = "root"
	role.Unsupported = append(role.Unsupported, "leaked")
	role.Invalid = append(role.Invalid, "leaked")

	second, err := spec.Schema()
	if err != nil {
		t.Fatalf("second Schema() error = %v", err)
	}
	fresh := second.Field("role")
	if fresh.MinLength == nil || *fresh.MinLength != 2 {
		t.Fatalf("MinLength drifted = %#v", fresh.MinLength)
	}
	if !reflect.DeepEqual(fresh.Enum, []any{"admin", "member"}) {
		t.Fatalf("Enum drifted = %#v", fresh.Enum)
	}
	if len(fresh.Unsupported) != 0 || len(fresh.Invalid) != 0 {
		t.Fatalf("reasons drifted = unsupported %v invalid %v", fresh.Unsupported, fresh.Invalid)
	}

	// The runtime rule is derived from the constructor, not the metadata, so it
	// stays correct after callers mutate an exported schema.
	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
	if err := (Spec{String("role", "a", MinRunes(2))}).Validate(); err == nil {
		t.Fatalf("Validate() accepted a value that is too short")
	}
}

func TestSchemaReportsUnsupportedAndInvalidTogether(t *testing.T) {
	t.Parallel()

	spec := Spec{
		String("company", "", Conditional(true, NonEmpty())),
		String("s", "abc", MinRunes(-1)),
	}
	_, err := spec.Schema()
	if err == nil {
		t.Fatalf("Schema() error = nil, want combined error")
	}

	var unsupported UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want UnsupportedError match", err)
	}
	if len(unsupported) != 1 || unsupported[0].Field != "company" {
		t.Fatalf("unsupported = %#v", unsupported)
	}

	var invalid InvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("error = %v, want InvalidError match", err)
	}
	if len(invalid) != 1 || invalid[0].Field != "s" {
		t.Fatalf("invalid = %#v", invalid)
	}
}

func TestSpecTightensOverlappingConstraints(t *testing.T) {
	t.Parallel()

	schema, err := (Spec{
		String("s", "", MinRunes(1), MinRunes(3), MaxRunes(10), MaxRunes(5)),
		Int("n", 0, MinValue(1), MinValue(5), MaxValue(100), MaxValue(50)),
	}).Schema()
	if err != nil {
		t.Fatalf("Schema() error = %v", err)
	}
	if got := *schema.Field("s").MinLength; got != 3 {
		t.Fatalf("MinLength = %d, want 3", got)
	}
	if got := *schema.Field("s").MaxLength; got != 5 {
		t.Fatalf("MaxLength = %d, want 5", got)
	}
	if got := schema.Field("n").Minimum; got != 5 {
		t.Fatalf("Minimum = %v, want 5", got)
	}
	if got := schema.Field("n").Maximum; got != 50 {
		t.Fatalf("Maximum = %v, want 50", got)
	}
}

func TestLegacyAPIStillWorks(t *testing.T) {
	t.Parallel()

	// The legacy Rule pipeline is untouched and can coexist with the new Spec.
	err := Validate(
		Field("name", "", Message("名称不能为空", Required())),
		Field("age", 1, Min(18)),
	)
	if err == nil || err.Error() != "name: 名称不能为空; age: 不能小于 18" {
		t.Fatalf("legacy Validate() error = %v", err)
	}
	if got := Validate(Field("ok", "value", Required())); got != nil {
		t.Fatalf("legacy valid error = %v, want nil", got)
	}

	// The legacy Required message and behaviour are unchanged.
	if err := Validate(Field("name", "", Required())); err == nil || err.Error() != "name: 不能为空" {
		t.Fatalf("legacy Required error = %v", err)
	}
}
