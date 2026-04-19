package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ginContextWithQuery(t *testing.T, query string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/templates?"+query, nil)
	return c
}

func TestTemplateListParamsFromQuery_AllFilters(t *testing.T) {
	c := ginContextWithQuery(t, "category=UTILITY&sub_category=ORDER_STATUS&language=pt_BR&status=APPROVED&quality_score=GREEN&name=welcome&content=hello&since=1700000000&until=1700086400&page=3&page_size=25")
	p := templateListParamsFromQuery(c)

	require.NotNil(t, p)
	assert.Equal(t, 3, p.Page)
	assert.Equal(t, 25, p.PageSize)
	assert.Equal(t, "UTILITY", p.Filters["category"])
	assert.Equal(t, "ORDER_STATUS", p.Filters["sub_category"])
	assert.Equal(t, "pt_BR", p.Filters["language"])
	assert.Equal(t, "APPROVED", p.Filters["status"])
	assert.Equal(t, "GREEN", p.Filters["quality_score"])
	assert.Equal(t, "welcome", p.Filters["name"])
	assert.Equal(t, "hello", p.Filters["content"])
	assert.Equal(t, int64(1700000000), p.Filters["since"])
	assert.Equal(t, int64(1700086400), p.Filters["until"])
}

func TestTemplateListParamsFromQuery_EmptyQuery_Defaults(t *testing.T) {
	c := ginContextWithQuery(t, "")
	p := templateListParamsFromQuery(c)

	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 50, p.PageSize)
	assert.Empty(t, p.Filters)
}

func TestTemplateListParamsFromQuery_MalformedSinceIgnored(t *testing.T) {
	c := ginContextWithQuery(t, "since=not-a-number&category=UTILITY")
	p := templateListParamsFromQuery(c)

	_, hasSince := p.Filters["since"]
	assert.False(t, hasSince, "malformed since must be dropped, not rejected")
	assert.Equal(t, "UTILITY", p.Filters["category"])
}

func TestTemplateListParamsFromQuery_PageSizeCapped(t *testing.T) {
	c := ginContextWithQuery(t, "page_size=9999")
	p := templateListParamsFromQuery(c)

	assert.Equal(t, 50, p.PageSize, "excessive page_size must fall back to default, not explode the query")
}

func TestTemplateListParamsFromQuery_InvalidPageIgnored(t *testing.T) {
	c := ginContextWithQuery(t, "page=abc")
	p := templateListParamsFromQuery(c)
	assert.Equal(t, 1, p.Page)
}
