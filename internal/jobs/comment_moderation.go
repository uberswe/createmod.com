package jobs

import (
	"context"
	"createmod/internal/mailer"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/riverqueue/river"
)

// CommentModerationArgs are the arguments for the async comment moderation job.
type CommentModerationArgs struct {
	CommentID   string `json:"comment_id"`
	Content     string `json:"content"`
	AuthorID    string `json:"author_id"`
	SchematicID string `json:"schematic_id"`
}

func (CommentModerationArgs) Kind() string { return "comment_moderation" }

// CommentModerationWorker runs OpenAI moderation on new comments.
type CommentModerationWorker struct {
	river.WorkerDefaults[CommentModerationArgs]
	deps Deps
}

func (w *CommentModerationWorker) Work(ctx context.Context, job *river.Job[CommentModerationArgs]) error {
	args := job.Args
	slog.Info("running comment moderation", "comment_id", args.CommentID, "schematic_id", args.SchematicID)

	if w.deps.Store == nil {
		slog.Warn("comment moderation job skipped: missing store")
		return nil
	}

	// Verify comment still exists
	comment, err := w.deps.Store.Comments.GetByID(ctx, args.CommentID)
	if err != nil || comment == nil {
		slog.Warn("comment moderation job: comment not found, skipping", "comment_id", args.CommentID)
		return nil
	}

	// Run moderation check
	if w.deps.Moderation != nil {
		result, err := w.deps.Moderation.CheckContent(args.Content)
		if err != nil {
			// Return the error so River retries on the slow schedule —
			// swallowing it would leave the comment permanently unmoderated
			// (this hid a moderation outage on 2026-07-23).
			slog.Warn("comment moderation job: moderation check unavailable, will retry",
				"error", err, "comment_id", args.CommentID, "attempt", job.Attempt)
			return fmt.Errorf("comment moderation check: %w", err)
		} else if !result.Approved {
			slog.Warn("comment moderation job: comment failed moderation",
				"comment_id", args.CommentID, "reason", result.Reason)

			// Disapprove the comment
			if err := w.deps.Store.Comments.Disapprove(ctx, args.CommentID); err != nil {
				slog.Error("comment moderation job: failed to disapprove comment", "error", err)
				return fmt.Errorf("disapproving comment: %w", err)
			}

			// Create a report
			report := &store.Report{
				TargetType: "comment",
				TargetID:   args.CommentID,
				Reason:     fmt.Sprintf("Auto-moderation: %s", result.Reason),
				Reporter:   "system",
			}
			if err := w.deps.Store.Reports.Create(ctx, report); err != nil {
				slog.Error("comment moderation job: failed to create report", "error", err)
			}

			// Email admins
			if w.deps.Mail != nil {
				baseURL := os.Getenv("BASE_URL")
				if baseURL == "" {
					baseURL = "https://createmod.com"
				}
				// Load schematic for URL
				schem, _ := w.deps.Store.Schematics.GetByID(ctx, args.SchematicID)
				schematicURL := baseURL + "/admin/reports"
				if schem != nil {
					schematicURL = fmt.Sprintf("%s/schematics/%s", baseURL, schem.Name)
				}

				to := moderationAdminRecipients(w.deps.Store, w.deps.Mail)
				if len(to) > 0 {
					subject := "Comment Blocked by Moderation"
					bodyText := fmt.Sprintf("A comment was blocked by automated moderation.\n\nReason: %s\n\nContent: %.200s", result.Reason, args.Content)
					htmlBody := mailer.EmailHTML(subject, "", schematicURL, "View Schematic", bodyText)
					msg := &mailer.Message{
						From:    w.deps.Mail.DefaultFrom(),
						To:      to,
						Subject: subject,
						HTML:    htmlBody,
					}
					if err := w.deps.Mail.Send(msg); err != nil {
						slog.Error("comment moderation job: failed to send admin email", "error", err)
					}
				}
			}

			slog.Info("comment moderation complete: blocked", "comment_id", args.CommentID)
			return nil
		}
	}

	// Comment passed moderation — translate to all languages
	if w.deps.Translation != nil {
		w.deps.Translation.TranslateComment(args.CommentID)
	}

	slog.Info("comment moderation complete: approved", "comment_id", args.CommentID)
	return nil
}

// NextRetry applies the slow retry schedule (30m doubling to a 24h ceiling):
// comment moderation failures usually mean OpenAI or the database is down.
func (w *CommentModerationWorker) NextRetry(job *river.Job[CommentModerationArgs]) time.Time {
	return slowRetryAt(job.Attempt, time.Now())
}
