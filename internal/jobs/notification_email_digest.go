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

	"github.com/riverqueue/river"
)

type NotificationEmailDigestArgs struct{}

func (NotificationEmailDigestArgs) Kind() string { return "notification_email_digest" }

type NotificationEmailDigestWorker struct {
	river.WorkerDefaults[NotificationEmailDigestArgs]
	deps Deps
}

func (w *NotificationEmailDigestWorker) Work(ctx context.Context, job *river.Job[NotificationEmailDigestArgs]) error {
	slog.Info("notification email digest started")
	if w.deps.Store == nil || w.deps.Mail == nil {
		return nil
	}

	now := time.Now().UTC()

	var frequencies []string
	if now.Hour() == 9 {
		frequencies = append(frequencies, "daily")
		if now.Weekday() == time.Monday {
			frequencies = append(frequencies, "weekly")
		}
	}
	if len(frequencies) == 0 {
		slog.Info("notification email digest: not a digest hour, skipping")
		return nil
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://createmod.com"
	}

	sent := 0
	for _, freq := range frequencies {
		userIDs, err := w.deps.Store.Notifications.ListUsersWithDigestPreference(ctx, freq)
		if err != nil {
			slog.Warn("notification email digest: list users failed", "frequency", freq, "err", err)
			continue
		}

		var since time.Time
		switch freq {
		case "daily":
			since = now.Add(-24 * time.Hour)
		case "weekly":
			since = now.Add(-7 * 24 * time.Hour)
		}

		for _, userID := range userIDs {
			notifications, err := w.deps.Store.Notifications.ListUnreadSince(ctx, userID, since)
			if err != nil || len(notifications) == 0 {
				continue
			}

			user, err := w.deps.Store.Users.GetUserByID(ctx, userID)
			if err != nil || user.Email == "" {
				continue
			}

			subject := fmt.Sprintf("Your %s notification digest", freq)
			var lines []string
			lines = append(lines, fmt.Sprintf("You have %d unread notifications:\n", len(notifications)))
			for i, n := range notifications {
				if i >= 10 {
					lines = append(lines, fmt.Sprintf("... and %d more", len(notifications)-10))
					break
				}
				line := fmt.Sprintf("- %s", n.Title)
				if n.Body != "" {
					line += ": " + n.Body
				}
				lines = append(lines, line)
			}

			bodyText := strings.Join(lines, "\n")
			htmlBody := mailer.EmailHTML(subject, "", baseURL+"/notifications", "View Notifications", bodyText)
			msg := &mailer.Message{
				From:    w.deps.Mail.DefaultFrom(),
				To:      []mail.Address{{Address: user.Email}},
				Subject: subject,
				HTML:    htmlBody,
			}
			if err := w.deps.Mail.Send(msg); err != nil {
				slog.Warn("notification email digest: send failed", "user", userID, "err", err)
			} else {
				sent++
			}
		}
	}

	slog.Info("notification email digest completed", "sent", sent)
	return nil
}
