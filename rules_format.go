package validate

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/netip"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Email accepts a single valid RFC 5322 address, matching the existing
// go-playground email rule used by account mail fields.
func Email() Rule[string] {
	return func(value string) error {
		address, err := mail.ParseAddress(value)
		if err != nil || address.Address != value {
			return errors.New("格式不正确")
		}
		return nil
	}
}

// URL accepts absolute URLs with a host, which matches the existing download
// URL use case and prevents relative paths from passing as external URLs.
func URL() Rule[string] {
	return func(value string) error {
		parsed, err := url.ParseRequestURI(value)
		if err != nil || !parsed.IsAbs() || parsed.Host == "" {
			return errors.New("格式不正确")
		}
		return nil
	}
}

// IP accepts a valid IPv4 or IPv6 address, matching the existing
// go-playground ip rule used by server address fields.
func IP() Rule[string] {
	return func(value string) error {
		if _, err := netip.ParseAddr(value); err != nil {
			return errors.New("IP格式不正确")
		}
		return nil
	}
}

// IPv4 accepts a valid IPv4 address.
func IPv4() Rule[string] {
	return func(value string) error {
		address, err := netip.ParseAddr(value)
		if err != nil || !address.Is4() {
			return errors.New("IPv4格式不正确")
		}
		return nil
	}
}

// IPv6 accepts a valid IPv6 address.
func IPv6() Rule[string] {
	return func(value string) error {
		address, err := netip.ParseAddr(value)
		if err != nil || !address.Is6() {
			return errors.New("IPv6格式不正确")
		}
		return nil
	}
}

// CIDR accepts a valid IPv4 or IPv6 prefix such as "10.0.0.0/8".
func CIDR() Rule[string] {
	return func(value string) error {
		if _, err := netip.ParsePrefix(value); err != nil {
			return errors.New("CIDR格式不正确")
		}
		return nil
	}
}

// Port accepts a decimal port number between 1 and 65535.
func Port() Rule[string] {
	return func(value string) error {
		port, err := strconv.Atoi(value)
		if err != nil || port < 1 || port > 65535 {
			return errors.New("端口不合法")
		}
		return nil
	}
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// UUID accepts canonical 8-4-4-4-12 UUID strings.
func UUID() Rule[string] {
	return func(value string) error {
		if !uuidPattern.MatchString(value) {
			return errors.New("UUID格式不正确")
		}
		return nil
	}
}

// Hex accepts strings that contain only hexadecimal digits.
func Hex() Rule[string] {
	return func(value string) error {
		for _, char := range value {
			if !isHexDigit(char) {
				return errors.New("只能包含十六进制字符")
			}
		}
		return nil
	}
}

func isHexDigit(char rune) bool {
	return char >= '0' && char <= '9' || char >= 'a' && char <= 'f' || char >= 'A' && char <= 'F'
}

// Base64 accepts standard, raw and URL-safe base64 encodings.
func Base64() Rule[string] {
	return func(value string) error {
		for _, encoding := range []*base64.Encoding{
			base64.StdEncoding,
			base64.RawStdEncoding,
			base64.URLEncoding,
			base64.RawURLEncoding,
		} {
			if _, err := encoding.DecodeString(value); err == nil {
				return nil
			}
		}
		return errors.New("Base64格式不正确")
	}
}

// JSON accepts strings that contain a valid JSON value.
func JSON() Rule[string] {
	return func(value string) error {
		if !json.Valid([]byte(value)) {
			return errors.New("JSON格式不正确")
		}
		return nil
	}
}

var phonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

// Phone accepts mainland China mobile phone numbers.
func Phone() Rule[string] {
	return func(value string) error {
		if !phonePattern.MatchString(value) {
			return errors.New("手机号格式不正确")
		}
		return nil
	}
}

var hostnamePattern = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

// Hostname accepts RFC 1123 host names.
func Hostname() Rule[string] {
	return func(value string) error {
		if len(value) == 0 || len(value) > 253 || !hostnamePattern.MatchString(value) {
			return errors.New("主机名格式不正确")
		}
		return nil
	}
}

var macPattern = regexp.MustCompile(`^([0-9a-fA-F]{2}[:-]){5}[0-9a-fA-F]{2}$`)

// MAC accepts MAC-48 addresses using ":" or "-" separators.
func MAC() Rule[string] {
	return func(value string) error {
		if !macPattern.MatchString(value) {
			return errors.New("MAC地址格式不正确")
		}
		return nil
	}
}

// Date accepts a calendar date in YYYY-MM-DD form, matching the existing
// datetime=2006-01-02 binding tag used by date-range query params.
func Date() Rule[string] {
	return func(value string) error {
		if _, err := time.Parse(time.DateOnly, value); err != nil {
			return errors.New("日期格式不正确")
		}
		return nil
	}
}

// DateTime accepts a time in the given Go layout, for example
// DateTime(time.DateTime) accepts "2006-01-02 15:04:05".
func DateTime(layout string) Rule[string] {
	return func(value string) error {
		if _, err := time.Parse(layout, value); err != nil {
			return fmt.Errorf("时间格式不正确，应为 %s", layout)
		}
		return nil
	}
}

// Duration accepts values parseable by time.ParseDuration, such as "1h30m".
func Duration() Rule[string] {
	return func(value string) error {
		if _, err := time.ParseDuration(value); err != nil {
			return errors.New("时长格式不正确")
		}
		return nil
	}
}

// Numeric accepts a string containing only ASCII digits, matching the
// existing go-playground numeric rule used by verification codes.
func Numeric() Rule[string] {
	return func(value string) error {
		for _, char := range value {
			if char < '0' || char > '9' {
				return errors.New("只能包含数字")
			}
		}
		return nil
	}
}

// Trimmed accepts strings that contain no leading or trailing whitespace.
func Trimmed() Rule[string] {
	return func(value string) error {
		if value != strings.TrimSpace(value) {
			return errors.New("首尾不能包含空白字符")
		}
		return nil
	}
}
