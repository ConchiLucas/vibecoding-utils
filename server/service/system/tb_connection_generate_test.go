package system

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/flipped-aurora/easy-deploy/server/utils"
)

func TestParseGeneratedIntRoundsFractionalValues(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  int64
	}{
		{name: "json number low fraction", value: json.Number("219.1"), want: 219},
		{name: "json number high fraction", value: json.Number("219.6"), want: 220},
		{name: "float fraction", value: float64(10.5), want: 11},
		{name: "string fraction", value: "42.2", want: 42},
		{name: "negative fraction", value: "-1.4", want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseGeneratedInt(tt.value)
			if !ok {
				t.Fatalf("parseGeneratedInt(%v) ok = false, want true", tt.value)
			}
			if got != tt.want {
				t.Fatalf("parseGeneratedInt(%v) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestNormalizeGeneratedInsertValueCoercesIntegerDecimal(t *testing.T) {
	got, err := normalizeGeneratedInsertValue("pgsql", utils.ClientColumnVO{
		Name:       "sort_no",
		ColumnType: "integer",
	}, json.Number("219.1"))
	if err != nil {
		t.Fatalf("normalizeGeneratedInsertValue returned error: %v", err)
	}
	if got != int64(219) {
		t.Fatalf("normalizeGeneratedInsertValue = %#v, want int64(219)", got)
	}
}

func TestNormalizeGeneratedInsertValueRejectsBadInteger(t *testing.T) {
	_, err := normalizeGeneratedInsertValue("pgsql", utils.ClientColumnVO{
		Name:       "sort_no",
		ColumnType: "integer",
	}, "not-a-number")
	if err == nil {
		t.Fatal("normalizeGeneratedInsertValue error = nil, want error")
	}
}

func TestNormalizeGeneratedInsertColumnValueFillsRequiredTemporalNull(t *testing.T) {
	got, useColumn, err := normalizeGeneratedInsertColumnValue("pgsql", utils.ClientColumnVO{
		Name:       "ship_date",
		ColumnType: "timestamp without time zone",
		NotNull:    true,
	}, nil, true, 4)
	if err != nil {
		t.Fatalf("normalizeGeneratedInsertColumnValue returned error: %v", err)
	}
	if !useColumn {
		t.Fatal("useColumn = false, want true")
	}
	text, ok := got.(string)
	if !ok || strings.TrimSpace(text) == "" {
		t.Fatalf("generated fallback = %#v, want non-empty timestamp string", got)
	}
	if _, err := parseTableUpdateTime(text); err != nil {
		t.Fatalf("fallback timestamp %q is not parseable: %v", text, err)
	}
}

func TestNormalizeGeneratedInsertColumnValueOmitsDefaultNull(t *testing.T) {
	got, useColumn, err := normalizeGeneratedInsertColumnValue("pgsql", utils.ClientColumnVO{
		Name:         "created_at",
		ColumnType:   "timestamp without time zone",
		NotNull:      true,
		HasDefault:   true,
		DefaultValue: "now()",
	}, nil, true, 0)
	if err != nil {
		t.Fatalf("normalizeGeneratedInsertColumnValue returned error: %v", err)
	}
	if useColumn {
		t.Fatalf("useColumn = true, want false; value = %#v", got)
	}
}

func TestParseGeneratedIntAcceptsQuotedNumberText(t *testing.T) {
	tests := []interface{}{`"2"`, `'2'`, "`2`", "“2”", "1,234"}
	for _, value := range tests {
		got, ok := parseGeneratedInt(value)
		if !ok {
			t.Fatalf("parseGeneratedInt(%#v) ok = false, want true", value)
		}
		if value == "1,234" {
			if got != 1234 {
				t.Fatalf("parseGeneratedInt(%#v) = %d, want 1234", value, got)
			}
			continue
		}
		if got != 2 {
			t.Fatalf("parseGeneratedInt(%#v) = %d, want 2", value, got)
		}
	}
}

func TestParseGeneratedTableRowsRepairsParenthesizedValues(t *testing.T) {
	rows, err := parseGeneratedTableRows(`[
		{
			"id": (1),
			"route_code": ("RT_001"),
			"deleted": (0),
			"active": (true),
			"remark": (null)
		}
	]`)
	if err != nil {
		t.Fatalf("parseGeneratedTableRows returned error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}
	if rows[0]["id"] != json.Number("1") {
		t.Fatalf("id = %#v, want json.Number(1)", rows[0]["id"])
	}
	if rows[0]["route_code"] != "RT_001" {
		t.Fatalf("route_code = %#v, want RT_001", rows[0]["route_code"])
	}
	if rows[0]["deleted"] != json.Number("0") {
		t.Fatalf("deleted = %#v, want json.Number(0)", rows[0]["deleted"])
	}
	if rows[0]["active"] != true {
		t.Fatalf("active = %#v, want true", rows[0]["active"])
	}
	if rows[0]["remark"] != nil {
		t.Fatalf("remark = %#v, want nil", rows[0]["remark"])
	}
}

func TestParseGeneratedTableRowsRepairsBarePseudoNumbers(t *testing.T) {
	rows, err := parseGeneratedTableRows(`[
		{
			"cost_price": pi,
			"est_price": waste24.50
		}
	]`)
	if err != nil {
		t.Fatalf("parseGeneratedTableRows returned error: %v", err)
	}
	if rows[0]["cost_price"] != json.Number("3.14") {
		t.Fatalf("cost_price = %#v, want json.Number(3.14)", rows[0]["cost_price"])
	}
	if rows[0]["est_price"] != json.Number("24.50") {
		t.Fatalf("est_price = %#v, want json.Number(24.50)", rows[0]["est_price"])
	}
}

func TestParseGeneratedTableRowsAcceptsRowsObject(t *testing.T) {
	rows, err := parseGeneratedTableRows(`这是前缀，会被忽略。
{
  "rows": [
    {
      "id": 1,
      "route_name": "京沪线",
      "remark": "包含 ] 和 } 的普通字符串"
    }
  ]
}
这是后缀，也会被忽略。`)
	if err != nil {
		t.Fatalf("parseGeneratedTableRows returned error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}
	if rows[0]["id"] != json.Number("1") {
		t.Fatalf("id = %#v, want json.Number(1)", rows[0]["id"])
	}
	if rows[0]["route_name"] != "京沪线" {
		t.Fatalf("route_name = %#v, want 京沪线", rows[0]["route_name"])
	}
}

func TestParseGeneratedTableRowsRejectsBareChineseValue(t *testing.T) {
	_, err := parseGeneratedTableRows(`[
		{
			"id":在我们输入第一笔资料时发生错误。首先我们对数据进行一些规范：1,
			"demand_no": "DEM20241128001"
		}
	]`)
	if err == nil {
		t.Fatal("parseGeneratedTableRows error = nil, want invalid JSON error")
	}
	if !strings.Contains(err.Error(), "AI 返回的数据不是合法 JSON 数组") {
		t.Fatalf("error = %q, want invalid JSON array message", err.Error())
	}
}
