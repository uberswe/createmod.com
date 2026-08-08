package pages

import "testing"

func TestEditorSessionToken(t *testing.T) {
	const id = "11111111-2222-3333-4444-555555555555"

	tok := editorSessionToken(id)
	if len(tok) != 32 {
		t.Fatalf("token length = %d, want 32", len(tok))
	}
	if editorSessionToken(id) != tok {
		t.Fatal("token is not deterministic for the same id")
	}
	if editorSessionToken("66666666-7777-8888-9999-000000000000") == tok {
		t.Fatal("different ids produced the same token")
	}
}

func TestEditorTokenValid(t *testing.T) {
	const id = "abcdef01-2345-6789-abcd-ef0123456789"
	tok := editorSessionToken(id)

	if !editorTokenValid(id, tok) {
		t.Fatal("valid token rejected")
	}
	if editorTokenValid(id, "") {
		t.Fatal("empty token accepted")
	}
	if editorTokenValid(id, tok[:len(tok)-1]+"0") {
		t.Fatal("tampered token accepted")
	}
	// A token minted for one session must not unlock another — this is the
	// property that stops a leaked worklist of bare ids from being usable.
	other := "00000000-0000-0000-0000-000000000001"
	if editorTokenValid(other, tok) {
		t.Fatal("token for one session validated against another")
	}
}
