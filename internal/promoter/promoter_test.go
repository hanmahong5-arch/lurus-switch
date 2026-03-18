package promoter

import "testing"

func TestGenerateShareLink(t *testing.T) {
	tests := []struct {
		name    string
		affCode string
		want    string
	}{
		{"with code", "ABC123", "https://lurus.cn/switch?ref=ABC123"},
		{"empty code", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateShareLink(tt.affCode)
			if got != tt.want {
				t.Errorf("GenerateShareLink(%q) = %q, want %q", tt.affCode, got, tt.want)
			}
		})
	}
}
