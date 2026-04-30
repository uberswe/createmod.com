package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_GeneratorPropeller_Template(t *testing.T) {
	path := filepath.Join("..", "..", "template", "generator-propeller.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)

	if !strings.Contains(s, "Propeller Generator") {
		t.Error("expected page title")
	}
	if !strings.Contains(s, "generator-viewport") {
		t.Error("expected viewport div")
	}
	if !strings.Contains(s, "/api/generators/propeller") {
		t.Error("expected API URL")
	}
	if !strings.Contains(s, "gen-download-btn") {
		t.Error("expected download button")
	}
	if !strings.Contains(s, "gen-preset") {
		t.Error("expected preset buttons")
	}
}

func Test_GeneratorBalloon_Template(t *testing.T) {
	path := filepath.Join("..", "..", "template", "generator-balloon.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)

	if !strings.Contains(s, "Airship Balloon Generator") {
		t.Error("expected page title")
	}
	if !strings.Contains(s, "/api/generators/balloon") {
		t.Error("expected API URL")
	}
	if !strings.Contains(s, "generator-viewport") {
		t.Error("expected viewport div")
	}
}

func Test_GeneratorHull_Template(t *testing.T) {
	path := filepath.Join("..", "..", "template", "generator-hull.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)

	if !strings.Contains(s, "Ship Hull Generator") {
		t.Error("expected page title")
	}
	if !strings.Contains(s, "/api/generators/hull") {
		t.Error("expected API URL")
	}
	if !strings.Contains(s, "generator-viewport") {
		t.Error("expected viewport div")
	}
	if !strings.Contains(s, "sternStyle") {
		t.Error("expected stern style parameter")
	}
}

func Test_Sidebar_HasGeneratorsLink(t *testing.T) {
	path := filepath.Join("..", "..", "template", "include", "sidebar.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)

	if !strings.Contains(s, "/generators/propeller") {
		t.Error("sidebar should link to generators")
	}
}

func Test_Header_HasGeneratorsDropdown(t *testing.T) {
	path := filepath.Join("..", "..", "template", "include", "header.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)

	if !strings.Contains(s, "/generators/propeller") {
		t.Error("header should link to propeller generator")
	}
	if !strings.Contains(s, "/generators/balloon") {
		t.Error("header should link to balloon generator")
	}
	if !strings.Contains(s, "/generators/hull") {
		t.Error("header should link to hull generator")
	}
}
