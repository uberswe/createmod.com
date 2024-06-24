package migrate

import (
	"createmod/query"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	"gorm.io/gorm"
)

func migrateComments(app *pocketbase.PocketBase, gormdb *gorm.DB, oldUserIDs map[int64]string, oldSchematicIDs map[int64]string) {
	// TODO check if comment exists, if it does we skip

	// QeyKryWEcomments
	q := query.Use(gormdb)
	postViewRes, postErr := q.QeyKryWEcomment.Find()
	if postErr != nil {
		panic(postErr)
	}

	schematicCommentsCollection, err := app.Dao().FindCollectionByNameOrId("comments")
	if err != nil {
		panic(err)
	}

	for _, s := range postViewRes {
		newSchematicID := oldSchematicIDs[s.CommentPostID]
		newUserID := oldUserIDs[s.UserID]
		record := models.NewRecord(schematicCommentsCollection)
		record.Set("old_schematic_id", s.CommentPostID)
		record.Set("schematic", newSchematicID)
		record.Set("author", newUserID)
		record.Set("author_url", s.CommentAuthorURL)
		record.Set("author_email", s.CommentAuthorEmail)
		record.Set("author_ip", s.CommentAuthorIP)
		record.Set("published", s.CommentDateGmt)
		record.Set("content", s.CommentContent)
		record.Set("karma", s.CommentKarma)
		record.Set("approved", s.CommentApproved)
		record.Set("agent", s.CommentAgent)
		record.Set("type", s.CommentType)
		record.Set("old_parent_id", s.CommentParent)
		record.Set("old_id", s.UserID)
		record.Set("old_schematic_id", s.CommentPostID)

		if err = app.Dao().SaveRecord(record); err != nil {
			panic(err)
		}
	}

	// iterate comments again to set parent comments

	comments, commentsErr := app.Dao().FindRecordsByFilter(schematicCommentsCollection.Id,
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
					c.Set("parent", c2.GetId())
					if err = app.Dao().SaveRecord(c); err != nil {
						panic(err)
					}
				}
			}
		}
	}
}
