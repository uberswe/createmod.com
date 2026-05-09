-- name: CreateNewsletterSubscriber :one
INSERT INTO newsletter_subscribers (email, user_id, type, frequency, confirm_token, unsubscribe_token)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (email, type) DO UPDATE SET
    frequency = EXCLUDED.frequency,
    updated = NOW()
RETURNING *;

-- name: ConfirmNewsletterSubscriber :exec
UPDATE newsletter_subscribers SET confirmed = true, updated = NOW()
WHERE confirm_token = $1;

-- name: UnsubscribeNewsletter :exec
UPDATE newsletter_subscribers SET confirmed = false, updated = NOW()
WHERE unsubscribe_token = $1;

-- name: ListConfirmedSubscribers :many
SELECT * FROM newsletter_subscribers
WHERE type = $1 AND confirmed = true
ORDER BY created ASC;

-- name: ListConfirmedSubscribersByFrequency :many
SELECT * FROM newsletter_subscribers
WHERE type = $1 AND frequency = $2 AND confirmed = true
ORDER BY created ASC;

-- name: GetNewsletterSubscriberByEmail :one
SELECT * FROM newsletter_subscribers
WHERE email = $1 AND type = $2
LIMIT 1;

-- name: CreateNewsletterIssue :one
INSERT INTO newsletter_issues (type, subject, html_body, slug, sent_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetNewsletterIssueBySlug :one
SELECT * FROM newsletter_issues WHERE slug = $1 LIMIT 1;

-- name: ListNewsletterIssues :many
SELECT * FROM newsletter_issues
WHERE type = $1
ORDER BY created DESC
LIMIT $2 OFFSET $3;

-- name: CreateSearchAlert :one
INSERT INTO search_alerts (user_id, query, filters, frequency, unsubscribe_token)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListSearchAlertsByUser :many
SELECT * FROM search_alerts
WHERE user_id = $1 AND active = true
ORDER BY created DESC;

-- name: ListActiveSearchAlerts :many
SELECT * FROM search_alerts
WHERE active = true
ORDER BY last_checked ASC NULLS FIRST
LIMIT $1;

-- name: DeleteSearchAlert :exec
DELETE FROM search_alerts WHERE id = $1 AND user_id = $2;

-- name: UnsubscribeSearchAlert :exec
UPDATE search_alerts SET active = false, updated = NOW()
WHERE unsubscribe_token = $1;

-- name: UpdateSearchAlertLastChecked :exec
UPDATE search_alerts SET last_checked = NOW(), updated = NOW()
WHERE id = $1;

-- name: UpdateSearchAlertLastNotified :exec
UPDATE search_alerts SET last_notified = NOW(), updated = NOW()
WHERE id = $1;

-- name: CreateSectionSubscription :one
INSERT INTO section_subscriptions (user_id, subscription_type, target_id, frequency, unsubscribe_token)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, subscription_type, target_id) DO UPDATE SET
    frequency = EXCLUDED.frequency,
    updated = NOW()
RETURNING *;

-- name: ListSectionSubscriptionsByUser :many
SELECT * FROM section_subscriptions
WHERE user_id = $1
ORDER BY created DESC;

-- name: ListSectionSubscriptionsByTarget :many
SELECT * FROM section_subscriptions
WHERE subscription_type = $1 AND target_id = $2
ORDER BY created ASC;

-- name: DeleteSectionSubscription :exec
DELETE FROM section_subscriptions WHERE id = $1 AND user_id = $2;

-- name: UnsubscribeSectionSubscription :exec
UPDATE section_subscriptions SET frequency = 'off', updated = NOW()
WHERE unsubscribe_token = $1;

-- name: UpdateNewsletterIssueSentAt :exec
UPDATE newsletter_issues SET sent_at = NOW() WHERE id = $1;

-- name: ListAllSectionSubscriptions :many
SELECT * FROM section_subscriptions
WHERE frequency != 'off'
ORDER BY subscription_type, target_id;
