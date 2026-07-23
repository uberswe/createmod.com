package jobs

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"createmod/internal/mailer"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

const (
	// failingJobsAttemptThreshold is how many failed attempts a job needs
	// before it appears in the admin digest.
	failingJobsAttemptThreshold = 5

	// failingJobsLookback bounds which failures count as "new" for a digest
	// run. Slightly over the periodic interval so failures on the boundary
	// aren't missed; a job that keeps failing reappears in a later digest
	// only after its next failed attempt.
	failingJobsLookback = failingJobsDigestInterval + 30*time.Minute

	// failingJobsDigestInterval is how often the digest job runs.
	failingJobsDigestInterval = 6 * time.Hour
)

// FailingJobsDigestArgs are the arguments for the failing-jobs digest job.
type FailingJobsDigestArgs struct{}

func (FailingJobsDigestArgs) Kind() string { return "failing_jobs_digest" }

// FailingJobsDigestWorker emails admins ONE summary of background jobs that
// have failed at least failingJobsAttemptThreshold times and are still
// retrying (or were discarded). One email covers all such jobs — deliberately
// batched so an outage affecting many jobs doesn't flood the inbox.
type FailingJobsDigestWorker struct {
	river.WorkerDefaults[FailingJobsDigestArgs]
	deps Deps
}

func (w *FailingJobsDigestWorker) Work(ctx context.Context, job *river.Job[FailingJobsDigestArgs]) error {
	if w.deps.Mail == nil || w.deps.Store == nil {
		return nil
	}
	client, err := river.ClientFromContextSafely[pgx.Tx](ctx)
	if err != nil {
		slog.Warn("failing jobs digest: no river client in context", "error", err)
		return nil
	}

	res, err := client.JobList(ctx, river.NewJobListParams().
		States(rivertype.JobStateRetryable, rivertype.JobStateDiscarded).
		OrderBy(river.JobListOrderByTime, river.SortOrderDesc).
		First(200))
	if err != nil {
		return fmt.Errorf("listing failing jobs: %w", err)
	}

	cutoff := time.Now().Add(-failingJobsLookback)
	var failing []*rivertype.JobRow
	for _, j := range res.Jobs {
		if j.Kind == (FailingJobsDigestArgs{}).Kind() {
			continue
		}
		if j.Attempt < failingJobsAttemptThreshold || len(j.Errors) == 0 {
			continue
		}
		// Only report jobs whose latest failure happened since the previous
		// digest window, so a long-retrying job produces one digest entry
		// per failed attempt, not one per digest run.
		if j.Errors[len(j.Errors)-1].At.Before(cutoff) {
			continue
		}
		failing = append(failing, j)
	}
	if len(failing) == 0 {
		return nil
	}

	to := moderationAdminRecipients(w.deps.Store, w.deps.Mail)
	if len(to) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString("<p>The following background jobs have failed at least ")
	fmt.Fprintf(&b, "%d times and need attention:</p>", failingJobsAttemptThreshold)
	b.WriteString("<table border=\"1\" cellpadding=\"6\" cellspacing=\"0\" style=\"border-collapse:collapse\">")
	b.WriteString("<tr><th>Job</th><th>ID</th><th>Attempts</th><th>Status</th><th>Last error</th></tr>")
	for _, j := range failing {
		status := "gave up (discarded)"
		if j.State == rivertype.JobStateRetryable {
			status = "next retry " + j.ScheduledAt.UTC().Format("2006-01-02 15:04 MST")
		}
		lastErr := j.Errors[len(j.Errors)-1].Error
		if len(lastErr) > 200 {
			lastErr = lastErr[:200] + "…"
		}
		fmt.Fprintf(&b, "<tr><td>%s</td><td>%d</td><td>%d</td><td>%s</td><td>%s</td></tr>",
			html.EscapeString(j.Kind), j.ID, j.Attempt, html.EscapeString(status), html.EscapeString(lastErr))
	}
	b.WriteString("</table>")

	subject := fmt.Sprintf("CreateMod: %d background job(s) failing", len(failing))
	msg := &mailer.Message{
		From:    w.deps.Mail.DefaultFrom(),
		To:      to,
		Subject: subject,
		HTML:    mailer.EmailHTMLRaw(subject, "", "", "", b.String()),
	}
	if err := w.deps.Mail.Send(msg); err != nil {
		return fmt.Errorf("sending failing jobs digest: %w", err)
	}
	slog.Info("failing jobs digest sent", "jobs", len(failing))
	return nil
}
