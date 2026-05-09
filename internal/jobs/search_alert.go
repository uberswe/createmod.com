package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/mail"
	"os"
	"strings"
	"time"

	"createmod/internal/mailer"
	"createmod/internal/search"
	"createmod/internal/store"

	"github.com/meilisearch/meilisearch-go"
	"github.com/riverqueue/river"
)

type SearchAlertCheckArgs struct{}

func (SearchAlertCheckArgs) Kind() string { return "search_alert_check" }

type SearchAlertCheckWorker struct {
	river.WorkerDefaults[SearchAlertCheckArgs]
	deps Deps
}

func (w *SearchAlertCheckWorker) Work(ctx context.Context, job *river.Job[SearchAlertCheckArgs]) error {
	slog.Info("search alert check started")
	if w.deps.Store == nil {
		return nil
	}

	alerts, err := w.deps.Store.SearchAlerts.ListActive(ctx, 100)
	if err != nil {
		return err
	}
	if len(alerts) == 0 {
		return nil
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://createmod.com"
	}

	notified := 0
	for _, alert := range alerts {
		if !w.shouldNotify(alert) {
			_ = w.deps.Store.SearchAlerts.UpdateLastChecked(ctx, alert.ID)
			continue
		}

		newIDs := w.findNewResults(ctx, alert)

		_ = w.deps.Store.SearchAlerts.UpdateLastChecked(ctx, alert.ID)

		if len(newIDs) == 0 {
			continue
		}

		if w.deps.Mail == nil {
			continue
		}

		user, err := w.deps.Store.Users.GetUserByID(ctx, alert.UserID)
		if err != nil || user.Email == "" {
			continue
		}

		schematics, err := w.deps.Store.Schematics.ListByIDs(ctx, newIDs)
		if err != nil || len(schematics) == 0 {
			continue
		}

		subject := fmt.Sprintf("New results for \"%s\"", alert.Query)
		var lines []string
		lines = append(lines, fmt.Sprintf("We found %d new schematics matching your search alert:\n", len(schematics)))
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

		unsubLink := fmt.Sprintf("%s/unsubscribe-alert?token=%s", baseURL, alert.UnsubscribeToken)
		lines = append(lines, fmt.Sprintf("\nTo unsubscribe from this alert: %s", unsubLink))

		bodyText := strings.Join(lines, "\n")
		searchURL := fmt.Sprintf("%s/search?q=%s", baseURL, alert.Query)
		htmlBody := mailer.EmailHTML(subject, "", searchURL, "View Search Results", bodyText)
		msg := &mailer.Message{
			From:    w.deps.Mail.DefaultFrom(),
			To:      []mail.Address{{Address: user.Email}},
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := w.deps.Mail.Send(msg); err != nil {
			slog.Warn("search alert check: send failed", "alert", alert.ID, "err", err)
		} else {
			notified++
			_ = w.deps.Store.SearchAlerts.UpdateLastNotified(ctx, alert.ID)
		}
	}

	slog.Info("search alert check completed", "alerts_checked", len(alerts), "notified", notified)
	return nil
}

func (w *SearchAlertCheckWorker) shouldNotify(alert store.SearchAlert) bool {
	now := time.Now().UTC()
	switch alert.Frequency {
	case "immediate":
		return true
	case "daily":
		if alert.LastNotified == nil {
			return true
		}
		return now.Sub(*alert.LastNotified) >= 24*time.Hour
	case "weekly":
		if alert.LastNotified == nil {
			return true
		}
		return now.Sub(*alert.LastNotified) >= 7*24*time.Hour
	default:
		return true
	}
}

func (w *SearchAlertCheckWorker) findNewResults(ctx context.Context, alert store.SearchAlert) []string {
	if w.deps.MeiliClient == nil {
		return nil
	}

	filter := ""
	if alert.LastNotified != nil {
		filter = fmt.Sprintf("created_unix > %d", alert.LastNotified.Unix())
	} else if alert.LastChecked != nil {
		filter = fmt.Sprintf("created_unix > %d", alert.LastChecked.Unix())
	}

	var alertFilters struct {
		Category string `json:"category"`
		Tags     []string `json:"tags"`
	}
	if len(alert.Filters) > 0 {
		_ = json.Unmarshal(alert.Filters, &alertFilters)
	}

	var filterParts []string
	if filter != "" {
		filterParts = append(filterParts, filter)
	}
	if alertFilters.Category != "" {
		filterParts = append(filterParts, fmt.Sprintf(`categories = "%s"`, alertFilters.Category))
	}

	combinedFilter := strings.Join(filterParts, " AND ")

	index := w.deps.MeiliClient.Index(search.MeiliIndex)
	result, err := index.SearchWithContext(ctx, alert.Query, &meilisearch.SearchRequest{
		Limit:  20,
		Filter: combinedFilter,
	})
	if err != nil {
		slog.Warn("search alert check: meili search failed", "alert", alert.ID, "err", err)
		return nil
	}

	ids := make([]string, 0, len(result.Hits))
	for _, hit := range result.Hits {
		var doc struct {
			ID string `json:"id"`
		}
		if err := hit.DecodeInto(&doc); err != nil || doc.ID == "" {
			continue
		}
		ids = append(ids, doc.ID)
	}
	return ids
}
