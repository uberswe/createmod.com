package migrate

import (
	"createmod/query"
	"errors"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"gorm.io/gorm"
	"log"
)

func migrateComments(app *pocketbase.PocketBase, gormdb *gorm.DB, oldUserIDs map[int64]string, oldSchematicIDs map[int64]string) {
	log.Println("Migrating comments.")

	// QeyKryWEcomments
	q := query.Use(gormdb)
	commentsRes, postErr := q.QeyKryWEcomment.Find()
	if postErr != nil {
		panic(postErr)
	}

	schematicCommentsCollection, err := app.FindCollectionByNameOrId("comments")
	if err != nil {
		panic(err)
	}

	totalCount := res{}
	countErr := app.DB().
		NewQuery("SELECT COUNT(id) as c FROM comments").
		One(&totalCount)
	if countErr != nil {
		panic(countErr)
	}

	if totalCount.C >= int64(len(commentsRes)) {
		log.Println("Skipping comments, already migrated.")
		return
	}

	updated := 0
	for _, s := range commentsRes {
		filter, err := app.FindRecordsByFilter(
			schematicCommentsCollection.Id,
			"old_id = {:old_id}",
			"-created",
			1,
			0,
			dbx.Params{"old_id": s.CommentID})
		if !errors.Is(err, gorm.ErrRecordNotFound) && len(filter) != 0 {
			app.Logger().Debug(
				fmt.Sprintf("Comment found or error: %v", err),
				"filter-len", len(filter),
			)
			continue
		}

		newSchematicID := oldSchematicIDs[s.CommentPostID]
		newUserID := oldUserIDs[s.UserID]
		record := core.NewRecord(schematicCommentsCollection)
		record.Set("old_schematic_id", s.CommentPostID)
		record.Set("schematic", newSchematicID)
		record.Set("author", newUserID)
		record.Set("author_url", s.CommentAuthorURL)
		record.Set("author_email", s.CommentAuthorEmail)
		record.Set("author_ip", s.CommentAuthorIP)
		record.Set("published", s.CommentDateGmt)
		record.Set("created", s.CommentDateGmt)
		record.Set("content", s.CommentContent)
		record.Set("karma", s.CommentKarma)
		record.Set("approved", s.CommentApproved)
		record.Set("agent", s.CommentAgent)
		record.Set("type", s.CommentType)
		record.Set("old_parent_id", s.CommentParent)
		record.Set("old_author_id", s.CommentID)
		record.Set("old_id", s.CommentID)
		record.Set("old_schematic_id", s.CommentPostID)

		if err = app.Save(record); err != nil {
			log.Printf("ERROR for %d - %d: %v\n", s.CommentID, s.CommentParent, err)
			continue
		}
		updated++
	}

	// iterate comments again to set parent comments

	comments, commentsErr := app.FindRecordsByFilter(schematicCommentsCollection.Id,
		"1 = 1",
		"-created",
		-1,
		0)
	if commentsErr != nil {
		panic(commentsErr)
	}

	for _, c := range comments {
		if c.GetInt("old_parent_id") > 0 {
			for _, c2 := range comments {
				if c2.GetInt("old_id") == c.GetInt("old_parent_id") {
					c.Set("parent", c2.Id)
					if err = app.Save(c); err != nil {
						log.Printf("ERROR for %d: %v\n", c.GetInt("old_id"), err)
						continue
					}
				}
			}
		}
	}
	log.Printf("%d comments migrated.\n", updated)
}
