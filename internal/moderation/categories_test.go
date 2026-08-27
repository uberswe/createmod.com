package moderation

import (
	"reflect"
	"testing"
)

func TestBlockingSchematicCategories(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"violence only is cleared", []string{"violence"}, []string{}},
		{"graphic violence only is cleared", []string{"graphic violence"}, []string{}},
		{"both violence variants cleared", []string{"violence", "graphic violence"}, []string{}},
		{"real category survives", []string{"sexual content"}, []string{"sexual content"}},
		{"violence stripped, rest kept", []string{"violence", "hate speech"}, []string{"hate speech"}},
		{"nothing flagged", nil, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := blockingSchematicCategories(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("blockingSchematicCategories(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
