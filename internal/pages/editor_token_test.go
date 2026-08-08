package pages

import (
	"testing"
	"time"
)

func TestEditorToken_MintAndAllow(t *testing.T) {
	const id = "11111111-2222-3333-4444-555555555555"
	now := time.Unix(1_700_000_000, 0)

	edit := mintEditorToken(id, editorScopeEdit, now)
	view := mintEditorToken(id, editorScopeView, now)

	// Edit token authorizes both edit and view; view authorizes only view.
	if !editorTokenAllows(id, edit, editorScopeEdit, now) {
		t.Fatal("edit token rejected for edit scope")
	}
	if !editorTokenAllows(id, edit, editorScopeView, now) {
		t.Fatal("edit token rejected for view scope (edit should satisfy view)")
	}
	if !editorTokenAllows(id, view, editorScopeView, now) {
		t.Fatal("view token rejected for view scope")
	}
	if editorTokenAllows(id, view, editorScopeEdit, now) {
		t.Fatal("view token wrongly satisfied edit scope — privilege escalation")
	}
}

func TestEditorToken_Expiry(t *testing.T) {
	const id = "abcdef01-2345-6789-abcd-ef0123456789"
	now := time.Unix(1_700_000_000, 0)
	tok := mintEditorToken(id, editorScopeView, now)

	if !editorTokenAllows(id, tok, editorScopeView, now.Add(59*time.Minute)) {
		t.Fatal("token rejected before expiry")
	}
	if editorTokenAllows(id, tok, editorScopeView, now.Add(editorTokenTTL+time.Second)) {
		t.Fatal("expired token accepted")
	}
}

func TestEditorToken_Rejections(t *testing.T) {
	const id = "abcdef01-2345-6789-abcd-ef0123456789"
	now := time.Unix(1_700_000_000, 0)
	tok := mintEditorToken(id, editorScopeEdit, now)

	if editorTokenAllows(id, "", editorScopeView, now) {
		t.Fatal("empty token accepted")
	}
	if editorTokenAllows(id, "e.deadbeef", editorScopeView, now) {
		t.Fatal("malformed token accepted")
	}
	// A token minted for one session must not unlock another — the property
	// that stops a leaked worklist of bare ids from being usable.
	if editorTokenAllows("00000000-0000-0000-0000-000000000001", tok, editorScopeView, now) {
		t.Fatal("token validated against a different session")
	}
	// Tampering with the signature must fail.
	if editorTokenAllows(id, tok[:len(tok)-1]+"0", editorScopeEdit, now) {
		t.Fatal("tampered signature accepted")
	}
	// Attacker rewrites the scope char from view->edit: the signature (which
	// covers the scope) no longer matches.
	view := mintEditorToken(id, editorScopeView, now)
	forged := "e" + view[1:]
	if editorTokenAllows(id, forged, editorScopeEdit, now) {
		t.Fatal("scope-upgraded token accepted — signature must bind the scope")
	}
}
