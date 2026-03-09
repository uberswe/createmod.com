package modmeta

import (
	"testing"
)

func TestExpandNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		want      []string // expected variants (order matters)
		mustHave  []string // variants that must be present (order doesn't matter)
	}{
		{
			name:      "concatenated create addon (all lowercase)",
			namespace: "createbigcannons",
			mustHave:  []string{"create big cannons", "Create: big cannons", "createbigcannons"},
		},
		{
			name:      "underscore separated",
			namespace: "design_decor",
			mustHave:  []string{"design decor", "design_decor"},
		},
		{
			name:      "base create mod",
			namespace: "create",
			want:      []string{"create"},
		},
		{
			name:      "no create prefix, no camelCase",
			namespace: "kubejs",
			mustHave:  []string{"kubejs"},
		},
		{
			name:      "camelCase with Create prefix",
			namespace: "CreateBigCannons",
			mustHave:  []string{"create big cannons", "Create: big cannons", "CreateBigCannons"},
		},
		{
			name:      "short create addon",
			namespace: "creategoggles",
			mustHave:  []string{"create goggles", "Create: goggles", "creategoggles"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandNamespace(tt.namespace)

			if tt.want != nil {
				if len(got) != len(tt.want) {
					t.Errorf("expandNamespace(%q) = %v, want %v", tt.namespace, got, tt.want)
					return
				}
				for i, v := range tt.want {
					if got[i] != v {
						t.Errorf("expandNamespace(%q)[%d] = %q, want %q", tt.namespace, i, got[i], v)
					}
				}
			}

			if tt.mustHave != nil {
				gotSet := make(map[string]bool)
				for _, v := range got {
					gotSet[v] = true
				}
				for _, v := range tt.mustHave {
					if !gotSet[v] {
						t.Errorf("expandNamespace(%q) = %v, missing expected variant %q", tt.namespace, got, v)
					}
				}
			}
		})
	}
}

func TestSplitCamelCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CreateBigCannons", "Create Big Cannons"},
		{"alreadylowercase", "alreadylowercase"},
		{"already split", "already split"},
		{"HTMLParser", "HTML Parser"},
		{"create", "create"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitCamelCase(tt.input)
			if got != tt.want {
				t.Errorf("splitCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
