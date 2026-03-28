package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var e164Regex = regexp.MustCompile(`^\+[1-9]\d{7,14}$`)

// NormalizePhone normalizes a phone number to E.164 format.
// Handles country-specific rules for BR, MX, AR.
func NormalizePhone(phone string) string {
	// Strip all non-digit chars except leading +
	hasPlus := strings.HasPrefix(phone, "+")
	digits := stripNonDigits(phone)

	if len(digits) == 0 {
		return phone
	}

	// Brazil (55): if 13 digits and DDD >= 31, remove the 9 after DDD (position 4)
	if len(digits) == 13 && strings.HasPrefix(digits, "55") {
		ddd := digits[2:4]
		if dddNum := atoi(ddd); dddNum >= 31 {
			digits = digits[:4] + digits[5:]
		}
	}

	// Mexico (52): if 13 digits, remove digit at position 2
	if len(digits) == 13 && strings.HasPrefix(digits, "52") {
		digits = digits[:2] + digits[3:]
	}

	// Argentina (54): if 13 digits, remove digit at position 2
	if len(digits) == 13 && strings.HasPrefix(digits, "54") {
		digits = digits[:2] + digits[3:]
	}

	if !hasPlus {
		return "+" + digits
	}
	return "+" + digits
}

// ValidateE164 validates a phone number is in E.164 format.
func ValidateE164(phone string) bool {
	return e164Regex.MatchString(phone)
}

// PhoneValidationMiddleware returns a gin middleware that normalizes phone numbers in request body.
func PhoneValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body == nil || c.Request.ContentLength == 0 {
			c.Next()
			return
		}

		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut && c.Request.Method != http.MethodPatch {
			c.Next()
			return
		}

		contentType := c.GetHeader("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			c.Next()
			return
		}

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}
		c.Request.Body.Close()

		var body map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			// Not valid JSON, restore body and continue
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Next()
			return
		}

		modified := false
		for _, key := range []string{"phone", "number"} {
			if val, ok := body[key]; ok {
				if strVal, ok := val.(string); ok && strVal != "" {
					normalized := NormalizePhone(strVal)
					if normalized != strVal {
						body[key] = normalized
						modified = true
					}
				}
			}
		}

		if modified {
			newBody, err := json.Marshal(body)
			if err == nil {
				bodyBytes = newBody
			}
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		c.Request.ContentLength = int64(len(bodyBytes))
		c.Next()
	}
}

func stripNonDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		}
	}
	return n
}
