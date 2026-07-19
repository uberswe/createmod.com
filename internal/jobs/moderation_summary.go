package jobs

import (
	"context"
	"createmod/internal/mailer"
	"createmod/internal/store"
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/riverqueue/river"
)

// moderationSummaryTimezone is the wall-clock timezone the summary schedule
// is anchored to. The job runs hourly and only sends at the summary hours.
const moderationSummaryTimezone = "Europe/Stockholm"

// ModerationSummaryArgs are the arguments for the twice-daily moderation
// summary email job.
type ModerationSummaryArgs struct{}

func (ModerationSummaryArgs) Kind() string { return "moderation_summary" }

// ModerationSummaryWorker sends admins a twice-daily digest (12:00 and 22:00
// Stockholm time) of auto-approved schematics, published collections, new
// guides, and new reports since the previous summary, plus the current list
// of schematics pending approval. It replaces the per-event admin emails.
type ModerationSummaryWorker struct {
	river.WorkerDefaults[ModerationSummaryArgs]
	deps Deps
}

// moderationSummaryWindow returns the reporting window ending at the summary
// hour that now falls in, or ok=false when now is not a summary hour. Windows
// have fixed boundaries (22:00→12:00 and 12:00→22:00 local time) so consecutive
// summaries never overlap or leave gaps, regardless of when within the hour
// the job actually runs.
func moderationSummaryWindow(now time.Time) (since, until time.Time, ok bool) {
	// Boundaries are built with time.Date rather than duration arithmetic so
	// they stay at 12:00/22:00 wall clock across DST transitions.
	at := func(t time.Time, hour int) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), hour, 0, 0, 0, now.Location())
	}
	switch now.Hour() {
	case 12:
		return at(now.AddDate(0, 0, -1), 22), at(now, 12), true
	case 22:
		return at(now, 12), at(now, 22), true
	}
	return time.Time{}, time.Time{}, false
}

func (w *ModerationSummaryWorker) Work(ctx context.Context, job *river.Job[ModerationSummaryArgs]) error {
	if w.deps.Store == nil || w.deps.Mail == nil {
		return nil
	}

	loc, err := time.LoadLocation(moderationSummaryTimezone)
	if err != nil {
		return fmt.Errorf("loading timezone: %w", err)
	}
	now := time.Now().In(loc)
	since, until, ok := moderationSummaryWindow(now)
	if !ok {
		return nil
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://createmod.com"
	}

	autoApproved, err := w.deps.Store.ModerationLog.ListAutoApprovedSince(ctx, since, until)
	if err != nil {
		slog.Warn("moderation summary: listing auto-approved schematics failed", "err", err)
	}
	collections, err := w.deps.Store.Collections.ListPublishedSince(ctx, since, until)
	if err != nil {
		slog.Warn("moderation summary: listing published collections failed", "err", err)
	}
	guides, err := w.deps.Store.Guides.ListCreatedSince(ctx, since, until)
	if err != nil {
		slog.Warn("moderation summary: listing new guides failed", "err", err)
	}
	reports, err := w.deps.Store.Reports.ListSince(ctx, since, until)
	if err != nil {
		slog.Warn("moderation summary: listing reports failed", "err", err)
	}
	pendingCount, err := w.deps.Store.Schematics.CountForAdmin(ctx, "pending", "")
	if err != nil {
		slog.Warn("moderation summary: counting pending schematics failed", "err", err)
	}
	const pendingLimit = 20
	var pending []store.Schematic
	if pendingCount > 0 {
		pending, err = w.deps.Store.Schematics.ListForAdmin(ctx, "pending", "", pendingLimit, 0)
		if err != nil {
			slog.Warn("moderation summary: listing pending schematics failed", "err", err)
		}
	}

	if len(autoApproved) == 0 && len(collections) == 0 && len(guides) == 0 && len(reports) == 0 && pendingCount == 0 {
		slog.Info("moderation summary: nothing to report, skipping email", "since", since, "until", until)
		return nil
	}

	body := w.buildBody(ctx, baseURL, loc, since, until, autoApproved, collections, guides, reports, pending, pendingCount)

	to := moderationAdminRecipients(w.deps.Store, w.deps.Mail)
	if len(to) == 0 {
		slog.Warn("moderation summary: no admin recipients, skipping email")
		return nil
	}

	subject := moderationSummarySubject(len(autoApproved), int(pendingCount), len(reports))
	msg := &mailer.Message{
		From:    w.deps.Mail.DefaultFrom(),
		To:      to,
		Subject: subject,
		HTML:    mailer.EmailHTMLRaw("Moderation Summary", "", baseURL+"/admin/schematics", "Open Moderation Queue", body),
	}
	if err := w.deps.Mail.Send(msg); err != nil {
		return fmt.Errorf("sending moderation summary email: %w", err)
	}

	slog.Info("moderation summary sent",
		"since", since, "until", until,
		"auto_approved", len(autoApproved), "collections", len(collections),
		"guides", len(guides), "reports", len(reports), "pending", pendingCount)
	return nil
}

func moderationSummarySubject(autoApproved, pending, reports int) string {
	parts := []string{}
	if autoApproved > 0 {
		parts = append(parts, fmt.Sprintf("%d auto-approved", autoApproved))
	}
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", pending))
	}
	if reports == 1 {
		parts = append(parts, "1 report")
	} else if reports > 1 {
		parts = append(parts, fmt.Sprintf("%d reports", reports))
	}
	if len(parts) == 0 {
		return "Moderation Summary"
	}
	return "Moderation Summary: " + strings.Join(parts, ", ")
}

func (w *ModerationSummaryWorker) buildBody(ctx context.Context, baseURL string, loc *time.Location, since, until time.Time,
	autoApproved []store.AutoApprovedSchematic, collections []store.Collection, guides []store.Guide,
	reports []store.Report, pending []store.Schematic, pendingCount int64) string {

	var b strings.Builder
	link := func(href, label string) string {
		return fmt.Sprintf(`<a href="%s" style="color:#206bc4;">%s</a>`, href, html.EscapeString(label))
	}
	section := func(title string) {
		fmt.Fprintf(&b, `<h3 style="margin:24px 0 8px;font-size:16px;">%s</h3>`, html.EscapeString(title))
	}
	openList := func() { b.WriteString(`<ul style="margin:0;padding-left:20px;">`) }
	closeList := func() { b.WriteString(`</ul>`) }

	fmt.Fprintf(&b, `<p style="margin:0 0 8px;color:#666;">Activity between %s and %s (%s).</p>`,
		html.EscapeString(since.In(loc).Format("Jan 2 15:04")),
		html.EscapeString(until.In(loc).Format("Jan 2 15:04")),
		html.EscapeString(moderationSummaryTimezone))

	section(fmt.Sprintf("Auto-approved schematics (%d)", len(autoApproved)))
	if len(autoApproved) == 0 {
		b.WriteString(`<p style="margin:0;color:#666;">None.</p>`)
	} else {
		openList()
		for _, s := range autoApproved {
			fmt.Fprintf(&b, `<li>%s <span style="color:#666;">— %s</span></li>`,
				link(baseURL+"/schematics/"+url.PathEscape(s.Name), s.Title),
				html.EscapeString(s.ApprovedAt.In(loc).Format("Jan 2 15:04")))
		}
		closeList()
	}

	section(fmt.Sprintf("Published collections (%d)", len(collections)))
	if len(collections) == 0 {
		b.WriteString(`<p style="margin:0;color:#666;">None.</p>`)
	} else {
		openList()
		for _, c := range collections {
			slug := c.Slug
			if slug == "" {
				slug = c.ID
			}
			b.WriteString("<li>" + link(baseURL+"/collections/"+url.PathEscape(slug), c.Title) + "</li>")
		}
		closeList()
	}

	section(fmt.Sprintf("New guides (%d)", len(guides)))
	if len(guides) == 0 {
		b.WriteString(`<p style="margin:0;color:#666;">None.</p>`)
	} else {
		openList()
		for _, g := range guides {
			b.WriteString("<li>" + link(baseURL+"/guides/"+url.PathEscape(g.Slug), g.Title) + "</li>")
		}
		closeList()
	}

	section(fmt.Sprintf("New reports (%d)", len(reports)))
	if len(reports) == 0 {
		b.WriteString(`<p style="margin:0;color:#666;">None.</p>`)
	} else {
		openList()
		for _, r := range reports {
			label := r.TargetType + " " + r.TargetID
			if r.TargetType == "schematic" {
				if s, err := w.deps.Store.Schematics.GetByID(ctx, r.TargetID); err == nil && s != nil {
					label = "schematic: " + s.Title
				}
			}
			reporter := "anonymous"
			if r.Reporter != "" {
				reporter = r.Reporter
				if u, err := w.deps.Store.Users.GetUserByID(ctx, r.Reporter); err == nil && u != nil {
					reporter = u.Username
				}
			}
			fmt.Fprintf(&b, `<li>%s <span style="color:#666;">reported by %s</span><br><span style="color:#666;">%s</span></li>`,
				link(baseURL+"/admin/reports", label),
				html.EscapeString(reporter),
				html.EscapeString(r.Reason))
		}
		closeList()
	}

	section(fmt.Sprintf("Schematics pending approval (%d)", pendingCount))
	if pendingCount == 0 {
		b.WriteString(`<p style="margin:0;color:#666;">None.</p>`)
	} else {
		openList()
		for _, s := range pending {
			reason := ""
			if s.ModerationReason != "" {
				reason = fmt.Sprintf(`<br><span style="color:#666;">%s</span>`, html.EscapeString(s.ModerationReason))
			}
			fmt.Fprintf(&b, `<li>%s <span style="color:#666;">(%s)</span>%s</li>`,
				link(baseURL+"/admin/schematics/"+url.PathEscape(s.ID), s.Title),
				html.EscapeString(s.ModerationState),
				reason)
		}
		if int64(len(pending)) < pendingCount {
			fmt.Fprintf(&b, `<li style="color:#666;">… and %d more</li>`, pendingCount-int64(len(pending)))
		}
		closeList()
	}

	return b.String()
}
