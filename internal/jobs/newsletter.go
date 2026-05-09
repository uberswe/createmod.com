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

type TrendingNewsletterArgs struct{}

func (TrendingNewsletterArgs) Kind() string { return "trending_newsletter" }

type TrendingNewsletterWorker struct {
	river.WorkerDefaults[TrendingNewsletterArgs]
	deps Deps
}

func (w *TrendingNewsletterWorker) Work(ctx context.Context, job *river.Job[TrendingNewsletterArgs]) error {
	slog.Info("trending newsletter started")
	if w.deps.Store == nil || w.deps.Mail == nil {
		return nil
	}

	now := time.Now().UTC()
	isMonday := now.Weekday() == time.Monday

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://createmod.com"
	}

	since := now.Add(-7 * 24 * time.Hour)
	topSchematics, err := w.deps.Store.SearchTracking.ListTopViewedSchematicsSince(ctx, since, 10)
	if err != nil {
		return fmt.Errorf("trending newsletter: fetch trending: %w", err)
	}
	if len(topSchematics) == 0 {
		slog.Info("trending newsletter: no trending schematics")
		return nil
	}

	subject := fmt.Sprintf("Trending on CreateMod.com — %s", now.Format("Jan 2, 2006"))
	bodyText := buildNewsletterBody(topSchematics, baseURL)

	htmlBody := mailer.EmailHTML(subject, "", baseURL, "Browse All Schematics", bodyText)

	issue := &store.NewsletterIssue{
		Type:     "trending",
		Subject:  subject,
		HTMLBody: htmlBody,
		Slug:     fmt.Sprintf("trending-%s", now.Format("2006-01-02")),
	}
	if err := w.deps.Store.Newsletters.CreateIssue(ctx, issue); err != nil {
		slog.Warn("trending newsletter: create issue failed", "err", err)
	}

	sent := 0
	for _, freq := range []string{"daily", "weekly"} {
		if freq == "weekly" && !isMonday {
			continue
		}

		subscribers, err := w.deps.Store.Newsletters.ListConfirmedByFrequency(ctx, "trending", freq)
		if err != nil {
			slog.Warn("trending newsletter: list subscribers failed", "frequency", freq, "err", err)
			continue
		}

		for _, sub := range subscribers {
			unsubLink := fmt.Sprintf("%s/unsubscribe?token=%s", baseURL, sub.UnsubscribeToken)
			personalBody := bodyText + fmt.Sprintf("\n\nTo unsubscribe: %s", unsubLink)
			personalHTML := mailer.EmailHTML(subject, "", baseURL, "Browse All Schematics", personalBody)

			msg := &mailer.Message{
				From:    w.deps.Mail.DefaultFrom(),
				To:      []mail.Address{{Address: sub.Email}},
				Subject: subject,
				HTML:    personalHTML,
			}
			if err := w.deps.Mail.Send(msg); err != nil {
				slog.Warn("trending newsletter: send failed", "email", sub.Email, "err", err)
			} else {
				sent++
			}
		}
	}

	if issue.ID != "" {
		_ = w.deps.Store.Newsletters.UpdateIssueSentAt(ctx, issue.ID)
	}

	slog.Info("trending newsletter completed", "schematics", len(topSchematics), "sent", sent)
	return nil
}

func buildNewsletterBody(schematics []store.TopViewedSchematic, baseURL string) string {
	var lines []string
	lines = append(lines, "Here are this week's most popular schematics:\n")
	for i, s := range schematics {
		title := s.Title
		if title == "" {
			title = s.Name
		}
		url := fmt.Sprintf("%s/schematics/%s", baseURL, s.Name)
		lines = append(lines, fmt.Sprintf("%d. %s (%d views)\n   %s", i+1, title, s.TotalViews, url))
	}
	return strings.Join(lines, "\n")
}
