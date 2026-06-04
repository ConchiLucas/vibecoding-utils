package system

import (
	"encoding/json"
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
