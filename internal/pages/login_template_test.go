package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Login_Template_Uses_POST(t *testing.T) {
	path := filepath.Join("..", "..", "template", "login.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	if !strings.Contains(s, "<form id=\"login-form\"") {
		t.Fatalf("login.html missing login form tag")
	}
	if !strings.Contains(s, "method=\"post\"") {
		t.Fatalf("login.html login form must use method=post to avoid GET leakage")
	}
	if !strings.Contains(s, "action=\"/login\"") {
		t.Fatalf("login.html login form must post to /login action")
	}
}
