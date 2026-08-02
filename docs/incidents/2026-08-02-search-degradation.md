# Incident Report — createmod.com search degradation

**Date:** 2026-08-02
**Severity:** SEV-3 (degraded, not down)
**Status:** Resolved — self-recovered
**User impact:** ~2 minutes
**Data loss:** None
**Environment:** production (`createmod-com-prod`)

---

## Summary

Between roughly **22:10 and 22:14 UTC**, external monitors reported `main-index`
unhealthy from Global, Western Europe, Eastern NA, and Western NA. During that
window the app logged a rapid burst of Meilisearch index rebuilds on the
`schematics_mods` index — each rebuild reprocessing **7,506 documents**.
Repeated back-to-back rebuilds raised search latency, and because the homepage
is search-backed, its health check timed out until the rebuilds settled.

Nothing crashed. There were **no OOM kills and no pod restarts** — the earlier
memory/CPU headroom increase absorbed the spike, so this was a latency blip
rather than the pod cascade seen in the previous incident. The autoscaler added
capacity and the site returned to normal within about two minutes.

## Key facts

| | |
|---|---|
| Impact window | ~2 min (22:10–22:14 UTC) |
| Trigger | Meilisearch index rebuild storm |
| Pod restarts (6 h) | 0 |
| OOM kills in window | 0 |
| Memory in use | ~1.0–1.4 GiB / pod vs 4 GiB limit (ample headroom) |
| Health now | HTTP 200 on `/` and `/api/ready`, 8/8 pods ready |

## Timeline (UTC)

| Time | Event |
|------|-------|
| 21:53:57 | First **0-result broad query** enqueues a rebuild of `schematics_mods`. |
| 21:58–22:01 | Four more broad-query triggers enqueue additional rebuilds in a ~3 min span. |
| 22:01:43 | **Rebuild completes** — 7,506 documents reindexed. |
| 22:06:28 | Trigger fires again — new rebuild enqueued. |
| 22:10:05 | Another rebuild enqueued. **External health checks begin failing** for `main-index` across regions. |
| 22:13:16 | Rebuild enqueued again — latency stays elevated; autoscaler adds pods (8→9). |
| 22:14:31 | Final trigger of the burst. |
| ~22:15 | **Recovered** — health checks pass, latency back to baseline. No restarts, no OOM. |

## Evidence

The app logs name the mechanism directly. A single index takes ~7,500 documents
to rebuild, and the trigger fired five times in roughly eight minutes:

```
22:01:39Z  INFO  meili: 0-result broad query detected, enqueueing index rebuild  index=schematics_mods
22:01:43Z  INFO  search index rebuild complete  count=7506
22:06:28Z  INFO  meili: 0-result broad query detected, enqueueing index rebuild  index=schematics_mods
22:10:05Z  INFO  meili: 0-result broad query detected, enqueueing index rebuild  index=schematics_mods
22:13:16Z  INFO  meili: 0-result broad query detected, enqueueing index rebuild  index=schematics_mods
22:14:31Z  INFO  meili: 0-result broad query detected, enqueueing index rebuild  index=schematics_mods
```

## Ruled out: the dev-domain migration did not cause this

Production runs image `master-93ca5f8` (built 31 Jul), unchanged through the
incident. The `dev.createmod.com` change (commit `18d21f0a`) touched only
`k8s/createmod/dev/*`; production manifests and image were untouched, and no
prod deploy or rollout occurred at 22:12.

## Root cause

A search that returns zero results for a *broad* query is treated as a signal
that the index is stale, so the app enqueues a full rebuild of `schematics_mods`.
In practice, ordinary user queries return zero results all the time (typos,
nonsense phrases, terms with no matches). During this window several such
queries landed close together, each enqueuing its own rebuild. The rebuilds ran
nearly back-to-back — five triggers in ~8 minutes, ~7,500 documents each — and
the sustained reindex load raised query latency enough that the search-backed
homepage failed its health check.

This is the same underlying behavior as the earlier OOM incident. The queue that
replaced the old synchronous resync (`enqueueing index rebuild` vs. the previous
`triggering background resync`) successfully prevented an out-of-memory cascade
this time — but it does not prevent the *latency* storm, because the triggers
themselves are neither de-duplicated nor rate-limited.

## Recommended fixes (app-side)

1. **Guard the trigger condition.** Don't treat every 0-result broad query as
   "index is stale." Rebuild only when the index is genuinely empty or its
   document count has actually dropped below a threshold. A user searching an
   unmatched phrase should never kick off a reindex.

2. **Debounce & coalesce rebuilds.** Collapse repeated triggers into at most one
   rebuild per cooldown window (e.g. 15–30 min), and drop enqueues while a
   rebuild is already in flight. Here, five triggers should have produced one
   rebuild, not a back-to-back series.

3. **Rebuild into a shadow index, then swap.** Build into a new index and use
   Meilisearch's atomic `swap-indexes`, so live queries keep hitting the current
   index at full speed and never see the reindex load.

4. **Cap broad-query result size.** Empty/broad queries currently return very
   large result sets (up to ~5,000). Cap the page size so a broad query is cheap
   regardless of how the rebuild logic evolves.

### Already in place (platform side)

- Memory/CPU headroom raised (4 GiB limit) — why this was a blip, not an OOM cascade.
- Horizontal autoscaling 4–16 replicas on CPU.
- Postgres read replica + PgBouncer pooling to keep DB load off the write path.

**Suggested guardrail:** alert on rebuild frequency (e.g. >2 rebuilds in 15 min)
so a trigger storm pages before it becomes user-visible.

---

## Resolution (implemented 2026-08-03)

**Fix:** guard the rebuild trigger on the index's *actual* state
(`internal/search/meili_engine.go`, `triggerResyncIfEmpty`). Before enqueuing a
rebuild, the trigger now queries Meilisearch for the live document count and
proceeds only when the index is genuinely empty (0 docs). A broad query
returning 0 hits while the index holds ~7,500 documents is treated as a
transient blip and ignored.

This breaks the feedback loop that drove the storm: a rebuild slowed
Meilisearch → broad queries transiently returned 0 → each 0-result enqueued
another rebuild. Because Meilisearch held 7,506 documents throughout the
incident, the new guard would have blocked all five triggers. If the count
can't be confirmed (stats call errors/times out) the trigger does nothing —
fail-safe, never rebuild on uncertainty. The existing per-pod cooldown and
cluster-wide River de-duplication remain as secondary defenses. Covered by a
regression test (`TestTriggerResync_SkipsWhenIndexHasDocuments`).

### On the "standalone worker for index building" idea

Considered, and not adopted as the fix, because it does not address this
incident's bottleneck. The rebuild job's Go-side work (reading the DB, mapping
documents) does run on a web pod, and a dedicated worker would move that off the
serving pods — a reasonable cleanup for autoscaler noise. But the user-visible
latency came from **Meilisearch itself** being saturated by the reindex writes:
the shared Meili instance slows *queries* while it indexes, and the homepage
health check is a query. Moving the job to another pod doesn't change how busy
Meilisearch is, so a standalone worker would not have prevented the health-check
timeouts. The trigger guard removes the storm entirely, which is the actual
problem here.

The right tool for making *legitimate* rebuilds (the hourly periodic sync, or a
genuine empty-index recovery) invisible to query latency is recommendation #3 —
**build into a shadow index and atomically `swap-indexes`** — so live queries
keep hitting the current index at full speed during any reindex. That is a
larger change, not required to close this incident, and recommended as a
follow-up if periodic-rebuild latency ever becomes noticeable.

### Follow-ups (not blocking)

- **Shadow index + atomic swap** for the periodic/recovery rebuild (rec. #3).
- **Cap broad-query result size** (rec. #4) — currently up to 5,000 IDs; needs
  care around the search page's result count / pagination.
- **Alert on rebuild frequency** (>2 rebuilds in 15 min) as an early warning.
