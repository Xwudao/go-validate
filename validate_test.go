package validate

import (
	"errors"
	"regexp"
	"testing"
	"time"
)

func TestRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{"required", Validate(Field("name", "", Required())), "name: 不能为空"},
		{"min length", Validate(Field("name", "a", MinLen(2))), "name: 长度不能小于 2"},
		{"max length unicode", Validate(Field("name", "你好世界", MaxLen(3))), "name: 长度不能大于 3"},
		{"min", Validate(Field("age", 17, Min(18))), "age: 不能小于 18"},
		{"max", Validate(Field("age", 121, Max(120))), "age: 不能大于 120"},
		{"not zero", Validate(Field("id", 0, NotZero[int]())), "id: 不能为空"},
		{"email", Validate(Field("email", "invalid", Email())), "email: 格式不正确"},
		{"one of", Validate(Field("type", "other", OneOf("personal", "company"))), "type: 值不合法"},
		{"min items", Validate(Field("permissions", []string{}, MinItems[string](1))), "permissions: 数量不能小于 1"},
		{"optional", Validate(Field("limit", (*int)(nil), Optional(Min(0)))), ""},
		{"url", Validate(Field("url", "https://example.com/file", URL())), ""},
		{"invalid url", Validate(Field("url", "/file", URL())), "url: 格式不正确"},
		{"when", Validate(Field("company_name", "", When(true, Required()))), "company_name: 不能为空"},
		{"when skipped", Validate(Field("company_name", "", When(false, Required()))), ""},
		{"message", Validate(Field("email", "", Message("邮箱不能为空", Required()))), "email: 邮箱不能为空"},
		{"valid", Validate(Field("email", "a@example.com", Required(), Email()), Field("age", 18, Min(18), Max(120))), ""},
		{"date", Validate(Field("date", "2026-08-20", Date())), ""},
		{"invalid date", Validate(Field("date", "20/08/2026", Date())), "date: 日期格式不正确"},
		{"ip", Validate(Field("ip", "192.168.1.1", IP())), ""},
		{"invalid ip", Validate(Field("ip", "not-an-ip", IP())), "ip: IP格式不正确"},
		{"numeric", Validate(Field("code", "123456", Numeric())), ""},
		{"invalid numeric", Validate(Field("code", "12a456", Numeric())), "code: 只能包含数字"},
		{"starts with", Validate(Field("version", "v1.0", StartsWith("v"))), ""},
		{"invalid starts with", Validate(Field("version", "1.0", StartsWith("v"))), "version: 必须以 v 开头"},
		{"max items", Validate(Field("tags", make([]string, 3), MaxItems[string](2))), "tags: 数量不能大于 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want == "" {
				if tt.err != nil {
					t.Fatalf("Validate() error = %v, want nil", tt.err)
				}
				return
			}
			if tt.err == nil || tt.err.Error() != tt.want {
				t.Fatalf("Validate() error = %v, want %q", tt.err, tt.want)
			}
		})
	}
}

func TestValidateAggregatesFieldsAndStopsAtFirstRule(t *testing.T) {
	t.Parallel()

	err := Validate(
		Field("email", "", Message("邮箱不能为空", Required()), Message("邮箱格式不正确", Email())),
		Field("age", 0, Message("年龄不能小于 18", Min(18))),
	)

	var validationErrors Errors
	if !errors.As(err, &validationErrors) {
		t.Fatalf("error type = %T, want Errors", err)
	}
	if len(validationErrors) != 2 {
		t.Fatalf("error count = %d, want 2", len(validationErrors))
	}
	if validationErrors[0].Message != "邮箱不能为空" {
		t.Fatalf("first field message = %q, want required message", validationErrors[0].Message)
	}
	if got := validationErrors.First(); got == nil || got.Error() != "邮箱不能为空" {
		t.Fatalf("First() = %v", got)
	}
}

func TestErrorsHelpers(t *testing.T) {
	t.Parallel()

	err := Validate(
		Field("email", "", Required()),
		Field("age", 1, Min(18)),
	)
	var validationErrors Errors
	if !errors.As(err, &validationErrors) {
		t.Fatalf("error type = %T, want Errors", err)
	}

	if !validationErrors.Has("email") || validationErrors.Has("name") {
		t.Fatalf("Has() returned wrong result")
	}
	if got := validationErrors.For("age"); got == nil || got.Message != "不能小于 18" {
		t.Fatalf("For(age) = %#v", got)
	}
	if got := validationErrors.Fields(); len(got) != 2 || got[0] != "email" || got[1] != "age" {
		t.Fatalf("Fields() = %v", got)
	}
	if got := validationErrors.Messages(); len(got) != 2 || got[0] != "不能为空" || got[1] != "不能小于 18" {
		t.Fatalf("Messages() = %v", got)
	}
	if got := (Errors{}).First(); got != nil {
		t.Fatalf("empty First() = %v, want nil", got)
	}
}

func TestComparisonRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{"between ok", Validate(Field("n", 5, Between(1, 10))), ""},
		{"between low", Validate(Field("n", 0, Between(1, 10))), "n: 必须在 1 和 10 之间"},
		{"between high", Validate(Field("n", 11, Between(1, 10))), "n: 必须在 1 和 10 之间"},
		{"positive", Validate(Field("n", 0, Positive[int]())), "n: 必须大于 0"},
		{"non negative", Validate(Field("n", -1, NonNegative[int]())), "n: 不能小于 0"},
		{"negative", Validate(Field("n", 0, Negative[int]())), "n: 必须小于 0"},
		{"non positive", Validate(Field("n", 1, NonPositive[int]())), "n: 不能大于 0"},
		{"eq", Validate(Field("n", 2, Eq(1))), "n: 必须等于 1"},
		{"ne", Validate(Field("n", 1, Ne(1))), "n: 不能等于 1"},
		{"not one of", Validate(Field("n", "b", NotOneOf("a", "b"))), "n: 值不合法"},
		{"not zero pointer", Validate(Field("p", (*string)(nil), NotZero[*string]())), "p: 不能为空"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ""
			if tt.err != nil {
				got = tt.err.Error()
			}
			if got != tt.want {
				t.Fatalf("error = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStringRules(t *testing.T) {
	t.Parallel()

	pattern := regexp.MustCompile(`^\d{4}$`)

	tests := []struct {
		name string
		err  error
		want string
	}{
		{"len ok", Validate(Field("s", "abcd", Len(4))), ""},
		{"len bad", Validate(Field("s", "abc", Len(4))), "s: 长度必须为 4"},
		{"ends with", Validate(Field("s", "a.go", EndsWith(".go"))), ""},
		{"invalid ends with", Validate(Field("s", "a.txt", EndsWith(".go"))), "s: 必须以 .go 结尾"},
		{"contains", Validate(Field("s", "hello", Contains("ell"))), ""},
		{"invalid contains", Validate(Field("s", "hello", Contains("xyz"))), "s: 必须包含 xyz"},
		{"not contains", Validate(Field("s", "hello", NotContains("xyz"))), ""},
		{"invalid not contains", Validate(Field("s", "hello", NotContains("ell"))), "s: 不能包含 ell"},
		{"match", Validate(Field("s", "2026", Match(pattern))), ""},
		{"invalid match", Validate(Field("s", "20a6", Match(pattern))), "s: 格式不正确"},
		{"match string", Validate(Field("s", "abc123", MatchString(`^[a-z]+\d+$`))), ""},
		{"alpha", Validate(Field("s", "abc", Alpha())), ""},
		{"invalid alpha", Validate(Field("s", "abc1", Alpha())), "s: 只能包含字母"},
		{"alphanum", Validate(Field("s", "abc123", AlphaNum())), ""},
		{"invalid alphanum", Validate(Field("s", "abc_", AlphaNum())), "s: 只能包含字母、数字"},
		{"ascii", Validate(Field("s", "abc", ASCII())), ""},
		{"invalid ascii", Validate(Field("s", "中文", ASCII())), "s: 只能包含 ASCII 字符"},
		{"no whitespace", Validate(Field("s", "abc", NoWhitespace())), ""},
		{"invalid no whitespace", Validate(Field("s", "a b", NoWhitespace())), "s: 不能包含空白字符"},
		{"lowercase", Validate(Field("s", "abc", Lowercase())), ""},
		{"invalid lowercase", Validate(Field("s", "aBc", Lowercase())), "s: 只能包含小写字母"},
		{"uppercase", Validate(Field("s", "ABC", Uppercase())), ""},
		{"invalid uppercase", Validate(Field("s", "AbC", Uppercase())), "s: 只能包含大写字母"},
		{"not blank", Validate(Field("s", "  x  ", NotBlank())), ""},
		{"invalid not blank", Validate(Field("s", "   ", NotBlank())), "s: 不能为空"},
		{"trimmed", Validate(Field("s", "abc", Trimmed())), ""},
		{"invalid trimmed", Validate(Field("s", " abc", Trimmed())), "s: 首尾不能包含空白字符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ""
			if tt.err != nil {
				got = tt.err.Error()
			}
			if got != tt.want {
				t.Fatalf("error = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ipv4", Validate(Field("s", "192.168.1.1", IPv4())), ""},
		{"invalid ipv4", Validate(Field("s", "::1", IPv4())), "s: IPv4格式不正确"},
		{"ipv6", Validate(Field("s", "::1", IPv6())), ""},
		{"invalid ipv6", Validate(Field("s", "1.2.3.4", IPv6())), "s: IPv6格式不正确"},
		{"cidr", Validate(Field("s", "10.0.0.0/8", CIDR())), ""},
		{"invalid cidr", Validate(Field("s", "10.0.0.0", CIDR())), "s: CIDR格式不正确"},
		{"port", Validate(Field("s", "8080", Port())), ""},
		{"invalid port", Validate(Field("s", "70000", Port())), "s: 端口不合法"},
		{"uuid", Validate(Field("s", "550e8400-e29b-41d4-a716-446655440000", UUID())), ""},
		{"invalid uuid", Validate(Field("s", "not-a-uuid", UUID())), "s: UUID格式不正确"},
		{"hex", Validate(Field("s", "deadBEEF01", Hex())), ""},
		{"invalid hex", Validate(Field("s", "xyz", Hex())), "s: 只能包含十六进制字符"},
		{"base64", Validate(Field("s", "aGVsbG8=", Base64())), ""},
		{"invalid base64", Validate(Field("s", "a", Base64())), "s: Base64格式不正确"},
		{"json", Validate(Field("s", `{"a":1}`, JSON())), ""},
		{"invalid json", Validate(Field("s", "{", JSON())), "s: JSON格式不正确"},
		{"phone", Validate(Field("s", "13800138000", Phone())), ""},
		{"invalid phone", Validate(Field("s", "12345", Phone())), "s: 手机号格式不正确"},
		{"hostname", Validate(Field("s", "api.example.com", Hostname())), ""},
		{"invalid hostname", Validate(Field("s", "-bad-.com", Hostname())), "s: 主机名格式不正确"},
		{"mac", Validate(Field("s", "01:23:45:67:89:ab", MAC())), ""},
		{"invalid mac", Validate(Field("s", "01:23", MAC())), "s: MAC地址格式不正确"},
		{"datetime", Validate(Field("s", "2026-08-20 10:30:00", DateTime(time.DateTime))), ""},
		{"invalid datetime", Validate(Field("s", "2026-08-20", DateTime(time.DateTime))), "s: 时间格式不正确，应为 2006-01-02 15:04:05"},
		{"duration", Validate(Field("s", "1h30m", Duration())), ""},
		{"invalid duration", Validate(Field("s", "abc", Duration())), "s: 时长格式不正确"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ""
			if tt.err != nil {
				got = tt.err.Error()
			}
			if got != tt.want {
				t.Fatalf("error = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCollectionRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{"len items", Validate(Field("s", []int{1, 2}, LenItems[int](2))), ""},
		{"invalid len items", Validate(Field("s", []int{1}, LenItems[int](2))), "s: 数量必须为 2"},
		{"unique", Validate(Field("s", []int{1, 2, 3}, Unique[int]())), ""},
		{"invalid unique", Validate(Field("s", []int{1, 2, 2}, Unique[int]())), "s: 不能包含重复项 2"},
		{"non empty map", Validate(Field("m", map[string]any{}, NonEmptyMap[string, any]())), "m: 不能为空"},
		{"non empty map passes", Validate(Field("m", map[string]any{"a": 1}, NonEmptyMap[string, any]())), ""},
		{"each", Validate(Field("s", []string{"a", "b"}, Each(Required()))), ""},
		{"invalid each", Validate(Field("s", []string{"a", ""}, Each(Required()))), "s: 第 2 项不能为空"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ""
			if tt.err != nil {
				got = tt.err.Error()
			}
			if got != tt.want {
				t.Fatalf("error = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCombinators(t *testing.T) {
	t.Parallel()

	if err := Validate(Field("s", "abc", All(Required(), MinLen(2), MaxLen(5)))); err != nil {
		t.Fatalf("All() = %v, want nil", err)
	}
	if err := Validate(Field("s", "", All(Required(), MinLen(2)))); err == nil || err.Error() != "s: 不能为空" {
		t.Fatalf("All() first error = %v", err)
	}
	if err := Validate(Field("s", "123456", Any(Numeric(), Email()))); err != nil {
		t.Fatalf("Any() = %v, want nil", err)
	}
	if err := Validate(Field("s", "abc", Any(Numeric(), Email()))); err == nil || err.Error() != "s: 只能包含数字" {
		t.Fatalf("Any() first error = %v", err)
	}
	if err := Validate(Field("s", "abc", Not(Required(), "不能填写"))); err == nil || err.Error() != "s: 不能填写" {
		t.Fatalf("Not() = %v", err)
	}
	if err := Validate(Field("s", "", Not(Required(), "不能填写"))); err != nil {
		t.Fatalf("Not() = %v, want nil", err)
	}
	if err := Validate(Field("s", "abc", Func(func(value string) error {
		if value != "abc" {
			return errors.New("必须为 abc")
		}
		return nil
	}))); err != nil {
		t.Fatalf("Func() = %v, want nil", err)
	}
}

// TestCustomRulesAreExtensible documents that any dependent project can
// define and use its own rules without changing this library.
func TestCustomRulesAreExtensible(t *testing.T) {
	t.Parallel()

	even := Rule[int](func(value int) error {
		if value%2 != 0 {
			return errors.New("必须是偶数")
		}
		return nil
	})

	if err := Validate(Field("n", 4, even)); err != nil {
		t.Fatalf("custom rule error = %v, want nil", err)
	}
	if err := Validate(Field("n", 3, even)); err == nil || err.Error() != "n: 必须是偶数" {
		t.Fatalf("custom rule error = %v", err)
	}

	// Custom rules compose with built-in combinators.
	optionalEven := Optional(even)
	if err := Validate(Field("n", (*int)(nil), optionalEven)); err != nil {
		t.Fatalf("optional custom rule error = %v, want nil", err)
	}
}
