package pages

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"createmod/internal/server"
	"createmod/internal/session"
	"createmod/internal/store"
)

// fakeEditorSessions returns one fixed session by id.
type fakeEditorSessions struct{ sess *store.EditorSession }

func (f fakeEditorSessions) Create(context.Context, string, string, string) (string, error) {
	return "", nil
}
func (f fakeEditorSessions) GetByID(_ context.Context, id string) (*store.EditorSession, error) {
	if f.sess != nil && f.sess.ID == id {
		return f.sess, nil
	}
	return nil, nil
}
func (f fakeEditorSessions) UpdateOps(context.Context, string, []byte, int) error { return nil }
func (f fakeEditorSessions) DeleteExpired(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func ownerTestReq(id, tok, authUserID string) *server.RequestEvent {
	req := httptest.NewRequest("GET", "/api/editor/"+id+"/preview.nbt", nil)
	req.SetPathValue("id", id)
	if tok != "" {
		req.Header.Set("X-Editor-Token", tok)
	}
	if authUserID != "" {
		req = req.WithContext(session.ContextWithSession(req.Context(),
			&session.Session{User: &session.SessionUser{ID: authUserID}}))
	}
	return &server.RequestEvent{Request: req}
}

// A logged-in user's session (UserID set) must still be viewable by an
// unauthenticated external viewer holding a valid VIEW token — that is the
// Bloxelizer/Shulkr flow. Edit scope still requires the authenticated owner.
func TestLoadEditorSession_ViewTokenBypassesOwnership(t *testing.T) {
	const id = "d7fa0c0b-6a71-4226-81e1-f30d8b1be28c"
	appStore := &store.Store{EditorSessions: fakeEditorSessions{
		sess: &store.EditorSession{ID: id, UserID: "owner-1"},
	}}
	now := time.Now()
	view := mintEditorToken(id, editorScopeView, now)
	edit := mintEditorToken(id, editorScopeEdit, now)

	// Unauthenticated view-token read of a logged-in-owned session → allowed.
	if _, err := loadEditorSession(ownerTestReq(id, view, ""), appStore, editorScopeView); err != nil {
		t.Fatalf("unauthenticated view-token read rejected: %v", err)
	}
	// Unauthenticated edit-token access to a logged-in-owned session → rejected.
	if _, err := loadEditorSession(ownerTestReq(id, edit, ""), appStore, editorScopeEdit); err == nil {
		t.Fatal("edit access to owned session allowed without authenticated owner")
	}
	// The authenticated owner can edit.
	if _, err := loadEditorSession(ownerTestReq(id, edit, "owner-1"), appStore, editorScopeEdit); err != nil {
		t.Fatalf("owner edit rejected: %v", err)
	}
	// A different authenticated user cannot edit.
	if _, err := loadEditorSession(ownerTestReq(id, edit, "intruder"), appStore, editorScopeEdit); err == nil {
		t.Fatal("non-owner edit allowed")
	}
}
