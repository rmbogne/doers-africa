package validation

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MaxEmailLength = 254
	MaxURLLength   = 2048
)

var postalCodePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 -]{1,10}[A-Za-z0-9]$`)

type FieldError struct {
	Field   string
	Message string
}

func (err FieldError) Error() string {
	if err.Field == "" {
		return err.Message
	}
	return err.Field + ": " + err.Message
}

func IsFieldError(err error) bool {
	var fieldError FieldError
	return errors.As(err, &fieldError)
}

func SingleLine(field, rawValue string, minimumRunes, maximumRunes int) (string, error) {
	value := strings.TrimSpace(rawValue)
	if err := validateUTF8AndControls(field, value, false); err != nil {
		return "", err
	}
	if err := validateRuneLength(field, value, minimumRunes, maximumRunes); err != nil {
		return "", err
	}
	return value, nil
}

func OptionalSingleLine(field, rawValue string, maximumRunes int) (string, error) {
	value := strings.TrimSpace(rawValue)
	if value == "" {
		return "", nil
	}
	return SingleLine(field, value, 1, maximumRunes)
}

func Multiline(field, rawValue string, minimumRunes, maximumRunes int) (string, error) {
	value := strings.TrimSpace(rawValue)
	if err := validateUTF8AndControls(field, value, true); err != nil {
		return "", err
	}
	if err := validateRuneLength(field, value, minimumRunes, maximumRunes); err != nil {
		return "", err
	}
	return value, nil
}

func OptionalMultiline(field, rawValue string, maximumRunes int) (string, error) {
	value := strings.TrimSpace(rawValue)
	if value == "" {
		return "", nil
	}
	return Multiline(field, value, 1, maximumRunes)
}

func Email(rawValue string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(rawValue))
	if len(value) == 0 || len(value) > MaxEmailLength || strings.ContainsAny(value, "\r\n\x00") {
		return "", FieldError{Field: "Email", Message: "must contain a valid email address"}
	}
	parsedAddress, err := mail.ParseAddress(value)
	if err != nil || parsedAddress.Address != value {
		return "", FieldError{Field: "Email", Message: "must contain a valid email address"}
	}
	return value, nil
}

func Enum(field, rawValue string, allowedValues ...string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(rawValue))
	for _, allowedValue := range allowedValues {
		if value == strings.ToLower(allowedValue) {
			return value, nil
		}
	}
	return "", FieldError{Field: field, Message: "contains an unsupported value"}
}

func Integer(field, rawValue string, minimum, maximum int) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(rawValue))
	if err != nil || value < minimum || value > maximum {
		return 0, FieldError{
			Field:   field,
			Message: fmt.Sprintf("must be a whole number between %d and %d", minimum, maximum),
		}
	}
	return value, nil
}

func PositiveInt64(field, rawValue string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(rawValue), 10, 64)
	if err != nil || value <= 0 {
		return 0, FieldError{Field: field, Message: "must be a positive identifier"}
	}
	return value, nil
}

func DateTodayOrFuture(field, rawValue string, maximumYears int, now time.Time) (time.Time, error) {
	parsedDate, err := time.Parse("2006-01-02", strings.TrimSpace(rawValue))
	if err != nil {
		return time.Time{}, FieldError{Field: field, Message: "must use the YYYY-MM-DD format"}
	}
	location := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	parsedDate = time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, location)
	if parsedDate.Before(today) {
		return time.Time{}, FieldError{Field: field, Message: "must be today or a future date"}
	}
	if maximumYears > 0 && parsedDate.After(today.AddDate(maximumYears, 0, 0)) {
		return time.Time{}, FieldError{Field: field, Message: fmt.Sprintf("must be within %d years", maximumYears)}
	}
	return parsedDate, nil
}

func OptionalURL(field, rawValue string) (string, error) {
	value := strings.TrimSpace(rawValue)
	if value == "" {
		return "", nil
	}
	if len(value) > MaxURLLength || strings.ContainsAny(value, "\r\n\x00") {
		return "", FieldError{Field: field, Message: "must be a valid HTTP or HTTPS URL"}
	}
	parsedURL, err := url.ParseRequestURI(value)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return "", FieldError{Field: field, Message: "must be a valid HTTP or HTTPS URL"}
	}
	return parsedURL.String(), nil
}

func OptionalPostalCode(rawValue string) (string, error) {
	value := strings.TrimSpace(rawValue)
	if value == "" {
		return "", nil
	}
	if !postalCodePattern.MatchString(value) {
		return "", FieldError{Field: "Postal code", Message: "must contain 3 to 12 letters, numbers, spaces, or hyphens"}
	}
	return value, nil
}

func OpaqueToken(field, rawValue string, minimumLength, maximumLength int) (string, error) {
	value := strings.TrimSpace(rawValue)
	if strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return "", FieldError{Field: field, Message: "contains invalid whitespace"}
	}
	if err := validateRuneLength(field, value, minimumLength, maximumLength); err != nil {
		return "", err
	}
	return value, nil
}

func Secret(field, rawValue string, minimumBytes, maximumBytes int) (string, error) {
	if !utf8.ValidString(rawValue) || strings.ContainsRune(rawValue, '\x00') {
		return "", FieldError{Field: field, Message: "contains invalid characters"}
	}
	byteLength := len([]byte(rawValue))
	if byteLength < minimumBytes || byteLength > maximumBytes {
		return "", FieldError{
			Field:   field,
			Message: fmt.Sprintf("must contain between %d and %d bytes", minimumBytes, maximumBytes),
		}
	}
	return rawValue, nil
}

func validateRuneLength(field, value string, minimumRunes, maximumRunes int) error {
	length := utf8.RuneCountInString(value)
	if length < minimumRunes || length > maximumRunes {
		return FieldError{
			Field:   field,
			Message: fmt.Sprintf("must contain between %d and %d characters", minimumRunes, maximumRunes),
		}
	}
	return nil
}

func validateUTF8AndControls(field, value string, allowMultiline bool) error {
	if !utf8.ValidString(value) {
		return FieldError{Field: field, Message: "contains invalid text encoding"}
	}
	for _, currentRune := range value {
		if !unicode.IsControl(currentRune) {
			continue
		}
		if allowMultiline && (currentRune == '\n' || currentRune == '\r' || currentRune == '\t') {
			continue
		}
		return FieldError{Field: field, Message: "contains unsupported control characters"}
	}
	return nil
}
