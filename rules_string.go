package validate

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MinLen rejects strings whose rune count is smaller than minimum.
func MinLen(minimum int) Rule[string] {
	return func(value string) error {
		if utf8.RuneCountInString(value) < minimum {
			return fmt.Errorf("长度不能小于 %d", minimum)
		}
		return nil
	}
}

// MaxLen rejects strings whose rune count is greater than maximum.
func MaxLen(maximum int) Rule[string] {
	return func(value string) error {
		if utf8.RuneCountInString(value) > maximum {
			return fmt.Errorf("长度不能大于 %d", maximum)
		}
		return nil
	}
}

// Len rejects strings whose rune count differs from exact.
func Len(exact int) Rule[string] {
	return func(value string) error {
		if utf8.RuneCountInString(value) != exact {
			return fmt.Errorf("长度必须为 %d", exact)
		}
		return nil
	}
}

// StartsWith rejects values that do not start with prefix.
func StartsWith(prefix string) Rule[string] {
	return func(value string) error {
		if !strings.HasPrefix(value, prefix) {
			return fmt.Errorf("必须以 %s 开头", prefix)
		}
		return nil
	}
}

// EndsWith rejects values that do not end with suffix.
func EndsWith(suffix string) Rule[string] {
	return func(value string) error {
		if !strings.HasSuffix(value, suffix) {
			return fmt.Errorf("必须以 %s 结尾", suffix)
		}
		return nil
	}
}

// Contains rejects values that do not contain sub.
func Contains(sub string) Rule[string] {
	return func(value string) error {
		if !strings.Contains(value, sub) {
			return fmt.Errorf("必须包含 %s", sub)
		}
		return nil
	}
}

// NotContains rejects values that contain sub.
func NotContains(sub string) Rule[string] {
	return func(value string) error {
		if strings.Contains(value, sub) {
			return fmt.Errorf("不能包含 %s", sub)
		}
		return nil
	}
}

// Match rejects values that do not match the regular expression.
func Match(re *regexp.Regexp) Rule[string] {
	return func(value string) error {
		if re == nil || !re.MatchString(value) {
			return errors.New("格式不正确")
		}
		return nil
	}
}

// MatchString is Match for a pattern that is compiled once. It panics when the
// pattern is invalid, because an invalid pattern is a programming error rather
// than invalid user input.
func MatchString(pattern string) Rule[string] {
	return Match(regexp.MustCompile(pattern))
}

// Alpha accepts ASCII letters only.
func Alpha() Rule[string] {
	return func(value string) error {
		for _, char := range value {
			if !isASCIILetter(char) {
				return errors.New("只能包含字母")
			}
		}
		return nil
	}
}

// AlphaNum accepts ASCII letters and digits, matching the existing
// go-playground alphanum rule used by account names.
func AlphaNum() Rule[string] {
	return func(value string) error {
		for _, char := range value {
			if !isASCIILetter(char) && (char < '0' || char > '9') {
				return errors.New("只能包含字母、数字")
			}
		}
		return nil
	}
}

// ASCII accepts strings that contain only ASCII characters.
func ASCII() Rule[string] {
	return func(value string) error {
		for _, char := range value {
			if char > unicode.MaxASCII {
				return errors.New("只能包含 ASCII 字符")
			}
		}
		return nil
	}
}

// NoWhitespace rejects strings containing any whitespace character.
func NoWhitespace() Rule[string] {
	return func(value string) error {
		for _, char := range value {
			if unicode.IsSpace(char) {
				return errors.New("不能包含空白字符")
			}
		}
		return nil
	}
}

// Lowercase accepts strings that contain no uppercase letters.
func Lowercase() Rule[string] {
	return func(value string) error {
		if value != strings.ToLower(value) {
			return errors.New("只能包含小写字母")
		}
		return nil
	}
}

// Uppercase accepts strings that contain no lowercase letters.
func Uppercase() Rule[string] {
	return func(value string) error {
		if value != strings.ToUpper(value) {
			return errors.New("只能包含大写字母")
		}
		return nil
	}
}

func isASCIILetter(char rune) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
}
