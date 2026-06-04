package utils

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
)

func TestIsCodeField(t *testing.T) {
	tests := []struct {
		name  string
		field ExtractedField
		want  bool
	}{
		{
			name:  "snake code suffix",
			field: ExtractedField{OriginalName: "carrier_code", JavaField: "carrierCode", Comment: "承运商"},
			want:  true,
		},
		{
			name:  "comment contains Chinese code keyword",
			field: ExtractedField{OriginalName: "carrier_no", JavaField: "carrierNo", Comment: "承运商编码"},
			want:  true,
		},
		{
			name:  "non code field",
			field: ExtractedField{OriginalName: "carrier_name", JavaField: "carrierName", Comment: "承运商名称"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCodeField(tt.field); got != tt.want {
				t.Fatalf("isCodeField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodeGenTemplateFuncsRenderLeafRule(t *testing.T) {
	tpl := `{[< range .field_lines >]}{[< if isCodeField . >]}name={[< leafRuleName . >]};prefix={[< leafRulePrefix $.moduleName . >]};remark={[< .Comment >]}
{[< end >]}{[< end >]}`

	parsed, err := template.New("leaf-rule").
		Delims("{[<", ">]}").
		Funcs(codeGenTemplateFuncs()).
		Parse(tpl)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	data := map[string]interface{}{
		"moduleName": "carrier",
		"field_lines": []ExtractedField{
			{OriginalName: "carrier_code", JavaField: "carrierCode", Comment: "承运商编码"},
			{OriginalName: "carrier_name", JavaField: "carrierName", Comment: "承运商名称"},
		},
	}

	var buf bytes.Buffer
	if err := parsed.Execute(&buf, data); err != nil {
		t.Fatalf("execute template: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "name=CARRIER_CODE;prefix=CARRIER;remark=承运商编码") {
		t.Fatalf("rendered template missing leaf rule values: %q", got)
	}
	if strings.Contains(got, "carrier_name") || strings.Contains(got, "承运商名称") {
		t.Fatalf("rendered template should skip non-code fields: %q", got)
	}
}
