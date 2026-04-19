package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildTemplateFilters_Empty(t *testing.T) {
	clause, args := buildTemplateFilters(nil, 2)
	assert.Empty(t, clause)
	assert.Empty(t, args)

	clause, args = buildTemplateFilters(map[string]interface{}{}, 2)
	assert.Empty(t, clause)
	assert.Empty(t, args)
}

func TestBuildTemplateFilters_AllSimpleKeys(t *testing.T) {
	clause, args := buildTemplateFilters(map[string]interface{}{
		"category":      "UTILITY",
		"sub_category":  "ORDER_STATUS",
		"language":      "pt_BR",
		"status":        "APPROVED",
		"quality_score": "GREEN",
	}, 2)
	assert.Contains(t, clause, "category = $")
	assert.Contains(t, clause, "sub_category = $")
	assert.Contains(t, clause, "language = $")
	assert.Contains(t, clause, "status = $")
	assert.Contains(t, clause, "quality_score = $")
	assert.Len(t, args, 5)
}

func TestBuildTemplateFilters_SkipsEmptyStrings(t *testing.T) {
	clause, args := buildTemplateFilters(map[string]interface{}{
		"category": "",
		"language": "en_US",
	}, 2)
	assert.NotContains(t, clause, "category")
	assert.Contains(t, clause, "language")
	assert.Equal(t, []interface{}{"en_US"}, args)
}

func TestBuildTemplateFilters_NameAndContentUseILIKE(t *testing.T) {
	clause, args := buildTemplateFilters(map[string]interface{}{
		"name":    "welcome",
		"content": "hello",
	}, 2)
	assert.Contains(t, clause, "name ILIKE $")
	assert.Contains(t, clause, "components::text ILIKE $")
	assert.Contains(t, args, "%welcome%")
	assert.Contains(t, args, "%hello%")
}

func TestBuildTemplateFilters_SinceUntilUseTimestamp(t *testing.T) {
	clause, args := buildTemplateFilters(map[string]interface{}{
		"since": int64(1700000000),
		"until": int64(1700086400),
	}, 2)
	assert.Contains(t, clause, "created_at >= to_timestamp($")
	assert.Contains(t, clause, "created_at <= to_timestamp($")
	assert.Len(t, args, 2)
	assert.Equal(t, int64(1700000000), args[0])
}

func TestBuildTemplateFilters_UnknownKeysIgnored(t *testing.T) {
	clause, args := buildTemplateFilters(map[string]interface{}{
		"bogus": "value",
	}, 2)
	assert.Empty(t, clause)
	assert.Empty(t, args)
}

func TestBuildTemplateFilters_PlaceholderNumbering(t *testing.T) {
	// When startArg=5 the first filter uses $5, not $1.
	clause, _ := buildTemplateFilters(map[string]interface{}{
		"category": "UTILITY",
	}, 5)
	assert.Contains(t, clause, "$5")
	assert.NotContains(t, clause, "$1")
}

func TestToUnix_AcceptsInt64IntAndFloat64(t *testing.T) {
	ts, ok := toUnix(int64(123))
	assert.True(t, ok)
	assert.Equal(t, int64(123), ts)

	ts, ok = toUnix(456)
	assert.True(t, ok)
	assert.Equal(t, int64(456), ts)

	ts, ok = toUnix(float64(789))
	assert.True(t, ok)
	assert.Equal(t, int64(789), ts)

	_, ok = toUnix("not a number")
	assert.False(t, ok)
}
