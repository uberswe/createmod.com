package pages

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"createmod/internal/server"
)

// Editor sessions are a capability: the session id is an unguessable 122-bit
// UUID and, historically, holding it was the entire access control — the
// preview endpoints were even CORS-open so external viewers could fetch them.
// That made the id a bearer secret, and once a batch of ids leaked (scrape,
// enumeration of a listing, or a dump) a third party could drain every
// session's unpublished contents by proxying /api/editor/<id>/preview.nbt
// through an external viewer, so the fetch reached us with the viewer's IP and
// headers instead of theirs.
//
// The token below re-binds access to a secret the id alone does not reveal.
// It is an HMAC over the session id under a subkey derived from the app
// security secret, so it cannot be computed from the id without the secret. A
// bare (leaked/enumerated) id therefore no longer grants access — the holder
// also needs the token, which is only ever handed to the session's own client.

// editorTokenKey derives a namespaced signing subkey from the app security
// secret. Bumping the version tag rotates every editor token independently of
// other SECURITY_SECRET uses (e.g. TOTP encryption).
func editorTokenKey() []byte {
	mac := hmac.New(sha256.New, []byte(securitySecret()))
	mac.Write([]byte("editor-preview-token-v1"))
	return mac.Sum(nil)
}

// editorSessionToken returns the capability token for a session id (128-bit,
// hex). Handed to the client at session creation and required on every read or
// mutation of that session.
func editorSessionToken(id string) string {
	mac := hmac.New(sha256.New, editorTokenKey())
	mac.Write([]byte(id))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// editorTokenValid reports, in constant time, whether tok is the token for id.
func editorTokenValid(id, tok string) bool {
	if tok == "" {
		return false
	}
	return hmac.Equal([]byte(tok), []byte(editorSessionToken(id)))
}

// editorTokenFromRequest reads the token from the X-Editor-Token header
// (same-origin XHR from the editor) or the ?t= query param (download links and
// external-viewer URLs, which cannot set headers).
func editorTokenFromRequest(e *server.RequestEvent) string {
	if t := e.Request.Header.Get("X-Editor-Token"); t != "" {
		return t
	}
	return e.Request.URL.Query().Get("t")
}
