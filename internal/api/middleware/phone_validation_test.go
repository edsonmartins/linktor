package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePhone_CleanNumber(t *testing.T) {
	result := NormalizePhone("5511999999999")
	assert.Equal(t, "+5511999999999", result)
}

func TestNormalizePhone_WithPlus(t *testing.T) {
	result := NormalizePhone("+5511999999999")
	assert.Equal(t, "+5511999999999", result)
}

func TestNormalizePhone_Brazil_RemoveNine(t *testing.T) {
	// 13 digits, country=55, DDD=31 (>=31), has extra 9 after DDD
	// +55 31 9 1234 5678 → +55 31 1234 5678
	result := NormalizePhone("+5531912345678")
	assert.Equal(t, "+553112345678", result)
}

func TestNormalizePhone_Brazil_DDD_Below31_NoChange(t *testing.T) {
	// DDD=21 (<31), should NOT remove the 9
	result := NormalizePhone("+5521912345678")
	assert.Equal(t, "+5521912345678", result)
}

func TestNormalizePhone_Mexico_Remove1(t *testing.T) {
	// 13 digits starting with 52: remove digit at position 2
	// +52 1 55 1234 5678 → +52 55 1234 5678
	result := NormalizePhone("+5215512345678")
	assert.Equal(t, "+525512345678", result)
}

func TestNormalizePhone_Argentina_Remove1(t *testing.T) {
	// 13 digits starting with 54: remove digit at position 2
	// +54 9 11 1234 5678 → +54 11 1234 5678
	result := NormalizePhone("+5491112345678")
	assert.Equal(t, "+541112345678", result)
}

func TestNormalizePhone_AlreadyClean(t *testing.T) {
	result := NormalizePhone("+14155551234")
	assert.Equal(t, "+14155551234", result)
}

func TestNormalizePhone_WithSpacesAndDashes(t *testing.T) {
	result := NormalizePhone("+1 (415) 555-1234")
	assert.Equal(t, "+14155551234", result)
}

func TestValidateE164_Valid(t *testing.T) {
	tests := []string{
		"+14155551234",
		"+5511999999999",
		"+442071234567",
		"+12345678",
	}
	for _, phone := range tests {
		assert.True(t, ValidateE164(phone), "expected valid: %s", phone)
	}
}

func TestValidateE164_Invalid(t *testing.T) {
	tests := []struct {
		phone  string
		reason string
	}{
		{"14155551234", "no plus sign"},
		{"+1234567", "too short (7 digits)"},
		{"+1234567890123456", "too long (16 digits)"},
		{"+1abc2345678", "contains letters"},
		{"", "empty"},
		{"+0123456789", "starts with 0 after +"},
	}
	for _, tt := range tests {
		assert.False(t, ValidateE164(tt.phone), "expected invalid (%s): %s", tt.reason, tt.phone)
	}
}

func TestPhoneValidationMiddleware_NormalizesPhone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(PhoneValidationMiddleware())
	router.POST("/test", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		var data map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &data))
		c.JSON(http.StatusOK, data)
	})

	payload := map[string]interface{}{
		"phone": "55 31 9 1234-5678",
		"name":  "Test",
	}
	bodyBytes, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "+553112345678", resp["phone"])
	assert.Equal(t, "Test", resp["name"])
}

func TestPhoneValidationMiddleware_NormalizesNumber(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(PhoneValidationMiddleware())
	router.POST("/test", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		var data map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &data))
		c.JSON(http.StatusOK, data)
	})

	payload := map[string]interface{}{
		"number": "+1 (415) 555-1234",
	}
	bodyBytes, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "+14155551234", resp["number"])
}
