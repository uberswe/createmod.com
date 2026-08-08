package pages

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"createmod/internal/server"
)

// Editor sessions are a capability: the session id is an unguessable 122-bit
// UUID, but holding it must not be enough to read the unpublished in-editor
// state behind it. Access is instead gated by a signed, time-limited token.
//
// There are two scopes:
//   - edit ("e"): full access (read + mutate). This is the editor client's own
//     token. It is delivered in JSON responses and sent back only in the
//     X-Editor-Token *header* — never placed in a URL, so it cannot leak to
//     analytics/ad scripts via the page's address bar, nor to third parties via
//     Referer or an outbound link.
//   - view ("v"): read-only preview access. This is the token that rides in the
//     ?t= query of the preview URLs handed to external viewers (Bloxelizer,
//     Shulkr) and the download links. It cannot mutate the session, so sharing
//     it — deliberately or by leaking through a third party's logs — grants only
//     what a preview link is meant to grant.
//
// Both scopes expire after editorTokenTTL. The client is re-issued fresh tokens
// in every editor response, so an actively edited session rolls forward
// indefinitely; an idle session and any handed-out view URL stop working once
// the TTL elapses. The signature covers the scope and expiry, so a view token
// cannot be edited into an edit token nor have its expiry extended.

const (
	editorScopeEdit = "e"
	editorScopeView = "v"

	// editorTokenTTL bounds how long a minted token is accepted. Kept short so
	// a leaked/handed-out view URL self-expires quickly; active editors are
	// unaffected because every response re-mints.
	editorTokenTTL = time.Hour
)

// editorTokenKey derives a namespaced signing subkey from the app security
// secret. The version tag rotates every editor token independently of other
// SECURITY_SECRET uses; bump it (v2 -> v3) to force-invalidate all live tokens.
func editorTokenKey() []byte {
	mac := hmac.New(sha256.New, []byte(securitySecret()))
	mac.Write([]byte("editor-preview-token-v2"))
	return mac.Sum(nil)
}

// editorTokenSig is the 128-bit hex HMAC binding scope+id+expiry.
func editorTokenSig(scope, id, expHex string) string {
	mac := hmac.New(sha256.New, editorTokenKey())
	mac.Write([]byte(scope + "|" + id + "|" + expHex))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// mintEditorToken issues a "<scope>.<expUnixHex>.<sig>" token for id, valid for
// editorTokenTTL from now.
func mintEditorToken(id, scope string, now time.Time) string {
	expHex := strconv.FormatInt(now.Add(editorTokenTTL).Unix(), 16)
	return scope + "." + expHex + "." + editorTokenSig(scope, id, expHex)
}

// editorTokenAllows reports, in constant time on the signature, whether tok
// authorizes need-scope access to id at time now: signature valid, not expired,
// and scope sufficient (an edit token satisfies a view requirement; a view
// token does not satisfy an edit requirement).
func editorTokenAllows(id, tok, need string, now time.Time) bool {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return false
	}
	scope, expHex, sig := parts[0], parts[1], parts[2]
	if scope != editorScopeEdit && scope != editorScopeView {
		return false
	}
	if !hmac.Equal([]byte(sig), []byte(editorTokenSig(scope, id, expHex))) {
		return false
	}
	exp, err := strconv.ParseInt(expHex, 16, 64)
	if err != nil || now.Unix() >= exp {
		return false
	}
	if need == editorScopeEdit && scope != editorScopeEdit {
		return false
	}
	return true
}

// editorTokenFromRequest reads the token from the X-Editor-Token header (the
// editor client's edit token) or the ?t= query param (view tokens on download
// links and external-viewer URLs, which cannot set headers).
func editorTokenFromRequest(e *server.RequestEvent) string {
	if t := e.Request.Header.Get("X-Editor-Token"); t != "" {
		return t
	}
	return e.Request.URL.Query().Get("t")
}
