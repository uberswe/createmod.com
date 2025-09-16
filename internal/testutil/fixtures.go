package testutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// randHex returns a random hex string of n bytes.
func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// SeedUser attempts to create a minimal user record in PocketBase if the
// users collection exists with common fields. This is best-effort and will
// silently no-op if the collection/fields are not present. Returns the record
// id when created, or an empty string otherwise.
func SeedUser(app *pocketbase.PocketBase, username string) (string, error) {
	if app == nil {
		return "", nil
	}
	coll, err := app.FindCollectionByNameOrId("users")
	if err != nil || coll == nil {
		return "", nil
	}
	r := core.NewRecord(coll)
	uname := username
	if strings.TrimSpace(uname) == "" {
		uname = "user_" + randHex(4)
	}
	// Try common auth fields (ignore errors on missing fields at Save time)
	r.Set("username", uname)
	r.Set("email", fmt.Sprintf("%s@example.test", uname))
	r.Set("password", randHex(8))
	r.Set("passwordConfirm", r.Get("password"))
	if err := app.Save(r); err != nil {
		// silently ignore schema mismatch
		return "", nil
	}
	return r.Id, nil
}

// SeedSchematic attempts to create a schematic record if the schematics
// collection exists. Best-effort and schema-tolerant.
func SeedSchematic(app *pocketbase.PocketBase, title string, checksum string, ownerId string) (string, error) {
	if app == nil {
		return "", nil
	}
	coll, err := app.FindCollectionByNameOrId("schematics")
	if err != nil || coll == nil {
		return "", nil
	}
	r := core.NewRecord(coll)
	name := title
	if strings.TrimSpace(name) == "" {
		name = "Schematic " + randHex(3)
	}
	r.Set("title", name)
	if checksum != "" {
		r.Set("checksum", checksum)
	}
	if ownerId != "" {
		r.Set("owner", ownerId)
	}
	if err := app.Save(r); err != nil {
		return "", nil
	}
	return r.Id, nil
}

// SeedComment attempts to create a comment for a schematic if the comments
// collection exists. Best-effort and schema-tolerant.
func SeedComment(app *pocketbase.PocketBase, schematicId string, userId string, body string) (string, error) {
	if app == nil {
		return "", nil
	}
	coll, err := app.FindCollectionByNameOrId("comments")
	if err != nil || coll == nil {
		return "", nil
	}
	r := core.NewRecord(coll)
	msg := body
	if strings.TrimSpace(msg) == "" {
		msg = "Nice build!"
	}
	r.Set("body", msg)
	if schematicId != "" {
		r.Set("schematic", schematicId)
	}
	if userId != "" {
		r.Set("author", userId)
	}
	if err := app.Save(r); err != nil {
		return "", nil
	}
	return r.Id, nil
}
