package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"os"
	"strings"
	"time"

	"createmod/internal/mailer"
	"createmod/internal/store"

	"github.com/riverqueue/river"
)

type SectionUpdateArgs struct{}

func (SectionUpdateArgs) Kind() string { return "section_update" }

type SectionUpdateWorker struct {
	river.WorkerDefaults[SectionUpdateArgs]
	deps Deps
}

func (w *SectionUpdateWorker) Work(ctx context.Context, job *river.Job[SectionUpdateArgs]) error {
	slog.Info("section update digest started")
	if w.deps.Store == nil || w.deps.Mail == nil {
		return nil
	}

	now := time.Now().UTC()
	isMonday := now.Weekday() == time.Monday

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://createmod.com"
	}

	dailyFollows, err := w.deps.Store.Follows.ListByFrequency(ctx, "daily")
	if err != nil {
		return fmt.Errorf("section update: list daily follows: %w", err)
	}

	var weeklyFollows []store.UserFollow
	if isMonday {
		weeklyFollows, err = w.deps.Store.Follows.ListByFrequency(ctx, "weekly")
		if err != nil {
			return fmt.Errorf("section update: list weekly follows: %w", err)
		}
	}

	allFollows := append(dailyFollows, weeklyFollows...)
	if len(allFollows) == 0 {
		slog.Info("section update: no follows with email frequency due")
		return nil
	}

	sent := 0
	for _, follow := range allFollows {
		var since time.Time
		switch follow.EmailFrequency {
		case "daily":
			since = now.Add(-24 * time.Hour)
		case "weekly":
			since = now.Add(-7 * 24 * time.Hour)
		default:
			continue
		}

		schematics := w.findNewSchematics(ctx, follow, since)
		if len(schematics) == 0 {
			continue
		}

		user, err := w.deps.Store.Users.GetUserByID(ctx, follow.UserID)
		if err != nil || user.Email == "" {
			continue
		}

		sectionName := w.resolveFollowName(ctx, follow)
		subject := fmt.Sprintf("New schematics in %s", sectionName)

		var lines []string
		lines = append(lines, fmt.Sprintf("%d new schematics were added:\n", len(schematics)))
		for i, s := range schematics {
			if i >= 5 {
				lines = append(lines, fmt.Sprintf("... and %d more", len(schematics)-5))
				break
			}
			title := s.Title
			if title == "" {
				title = s.Name
			}
			lines = append(lines, fmt.Sprintf("- %s\n  %s/schematics/%s", title, baseURL, s.Name))
		}

		unsubLink := fmt.Sprintf("%s/unsubscribe?token=%s", baseURL, follow.UnsubscribeToken)
		lines = append(lines, fmt.Sprintf("\nTo unsubscribe: %s", unsubLink))

		bodyText := strings.Join(lines, "\n")
		htmlBody := mailer.EmailHTML(subject, "", baseURL, "Browse Schematics", bodyText)
		msg := &mailer.Message{
			From:    w.deps.Mail.DefaultFrom(),
			To:      []mail.Address{{Address: user.Email}},
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := w.deps.Mail.Send(msg); err != nil {
			slog.Warn("section update: send failed", "user", follow.UserID, "err", err)
		} else {
			_ = w.deps.Store.Follows.UpdateLastNotified(ctx, follow.ID)
			sent++
		}
	}

	slog.Info("section update digest completed", "follows", len(allFollows), "sent", sent)
	return nil
}

func (w *SectionUpdateWorker) findNewSchematics(ctx context.Context, follow store.UserFollow, since time.Time) []store.Schematic {
	switch follow.FollowType {
	case "category":
		schematics, err := w.deps.Store.Schematics.ListByCategoryIDs(ctx, []string{follow.TargetID}, nil, 10)
		if err != nil {
			return nil
		}
		var recent []store.Schematic
		for _, s := range schematics {
			if s.Created.After(since) {
				recent = append(recent, s)
			}
		}
		return recent
	case "user":
		schematics, err := w.deps.Store.Schematics.ListByAuthor(ctx, follow.TargetID, 10, 0)
		if err != nil {
			return nil
		}
		var recent []store.Schematic
		for _, s := range schematics {
			if s.Created.After(since) {
				recent = append(recent, s)
			}
		}
		return recent
	case "latest":
		schematics, err := w.deps.Store.Schematics.ListApproved(ctx, 10, 0)
		if err != nil {
			return nil
		}
		var recent []store.Schematic
		for _, s := range schematics {
			if s.Created.After(since) {
				recent = append(recent, s)
			}
		}
		return recent
	case "tag":
		tag, err := w.deps.Store.Tags.GetByID(ctx, follow.TargetID)
		if err != nil || tag == nil {
			return nil
		}
		schematics, err := w.deps.Store.Schematics.ListApproved(ctx, 50, 0)
		if err != nil {
			return nil
		}
		var recent []store.Schematic
		for _, s := range schematics {
			if !s.Created.After(since) {
				continue
			}
			tagIDs, err := w.deps.Store.Schematics.GetTagIDs(ctx, s.ID)
			if err != nil {
				continue
			}
			for _, tid := range tagIDs {
				if tid == follow.TargetID {
					recent = append(recent, s)
					break
				}
			}
			if len(recent) >= 10 {
				break
			}
		}
		return recent
	default:
		return nil
	}
}

func (w *SectionUpdateWorker) resolveFollowName(ctx context.Context, follow store.UserFollow) string {
	switch follow.FollowType {
	case "category":
		cat, err := w.deps.Store.Categories.GetByID(ctx, follow.TargetID)
		if err == nil && cat != nil {
			return cat.Name
		}
		return "your subscribed category"
	case "tag":
		tag, err := w.deps.Store.Tags.GetByID(ctx, follow.TargetID)
		if err == nil && tag != nil {
			return tag.Name
		}
		return "your subscribed tag"
	case "user":
		user, err := w.deps.Store.Users.GetUserByID(ctx, follow.TargetID)
		if err == nil {
			return user.Username
		}
		return "a creator you follow"
	case "latest":
		return "Latest Schematics"
	case "trending":
		return "Trending Schematics"
	case "highest_rated":
		return "Highest Rated Schematics"
	case "mod":
		return "a mod you follow"
	case "search":
		return "a saved search"
	default:
		return "your subscription"
	}
}
