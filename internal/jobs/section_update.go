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

	subs, err := w.deps.Store.SectionSubscriptions.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("section update: list subscriptions: %w", err)
	}
	if len(subs) == 0 {
		slog.Info("section update: no subscriptions")
		return nil
	}

	now := time.Now().UTC()
	isMonday := now.Weekday() == time.Monday

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://createmod.com"
	}

	sent := 0
	for _, sub := range subs {
		if sub.Frequency == "weekly" && !isMonday {
			continue
		}

		var since time.Time
		switch sub.Frequency {
		case "daily":
			since = now.Add(-24 * time.Hour)
		case "weekly":
			since = now.Add(-7 * 24 * time.Hour)
		default:
			continue
		}

		schematics := w.findNewSchematics(ctx, sub, since)
		if len(schematics) == 0 {
			continue
		}

		user, err := w.deps.Store.Users.GetUserByID(ctx, sub.UserID)
		if err != nil || user.Email == "" {
			continue
		}

		sectionName := w.resolveSectionName(ctx, sub)
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

		unsubLink := fmt.Sprintf("%s/unsubscribe-section?token=%s", baseURL, sub.UnsubscribeToken)
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
			slog.Warn("section update: send failed", "user", sub.UserID, "err", err)
		} else {
			sent++
		}
	}

	slog.Info("section update digest completed", "subscriptions", len(subs), "sent", sent)
	return nil
}

func (w *SectionUpdateWorker) findNewSchematics(ctx context.Context, sub store.SectionSubscription, since time.Time) []store.Schematic {
	switch sub.SubscriptionType {
	case "category":
		schematics, err := w.deps.Store.Schematics.ListByCategoryIDs(ctx, []string{sub.TargetID}, nil, 10)
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
		tag, err := w.deps.Store.Tags.GetByID(ctx, sub.TargetID)
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
				if tid == sub.TargetID {
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

func (w *SectionUpdateWorker) resolveSectionName(ctx context.Context, sub store.SectionSubscription) string {
	switch sub.SubscriptionType {
	case "category":
		cat, err := w.deps.Store.Categories.GetByID(ctx, sub.TargetID)
		if err == nil && cat != nil {
			return cat.Name
		}
		return "your subscribed category"
	case "tag":
		tag, err := w.deps.Store.Tags.GetByID(ctx, sub.TargetID)
		if err == nil && tag != nil {
			return tag.Name
		}
		return "your subscribed tag"
	default:
		return "your subscribed section"
	}
}
