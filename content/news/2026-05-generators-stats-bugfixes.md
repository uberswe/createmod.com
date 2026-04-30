---
title: "Generators, Stats and Bugfixes"
date: 2026-05-01
slug: generators-stats-bugfixes
excerpt: "New schematic generators, detailed analytics for your uploads, a public API spec, and a batch of bug fixes."
image: "/assets/x/news/generators.png"
---

It's May and it's time for a few new updates since the last [March update](/news/march-2026-site-update).

## Schematic Generators

You can now **generate schematics** directly on the site. Three generators are available specifically for airships:

- **Propeller** — configure blade count, radius, and pitch
- **Balloon** — set size and shape
- **Hull** — define length, beam, and profile

Each generator produces a downloadable `.nbt` file you can drop straight into your world. Find them under the new [Generators](/generators) section. You can also try the new layer by layer guide if you want to build things by hand in-game.

We have also made the layer guide available for all current schematics.

## Analytics for Your Schematics

Every schematic you upload now has a **Stats** page showing hourly data over the last 30 days:

- **Views and downloads** — see how your builds are performing
- **Video plays and YouTube clicks** — if your schematic has a video, track engagement
- **Average time on page** — how long visitors spend looking at your work
- **Layer viewer tracking** — usage of the building guide

A **view-to-download ratio** compares your schematic against the site average so you can tell how compelling your thumbnails and descriptions are.

There is also a unified [Statistics](/settings/statistics) page in your account settings with aggregate charts and a sortable list of all your schematics.

## Public API & OpenAPI Spec

Two new authenticated API endpoints let you pull your analytics data programmatically:

- `GET /api/schematics/{name}/stats` — hourly stats for a single schematic
- `GET /api/user/stats` — aggregate stats and paginated schematic list

A full [OpenAPI 3.0.3 specification](/api/openapi.json) is now available covering all public endpoints, including HMAC-authenticated mod download routes. You can find details on the [API documentation](/api) page.

## Performance Improvements

- **Page caching** for anonymous visitors on high-traffic pages cuts response times significantly
- **Batch database queries** reduce the number of round-trips on listing pages
- **S3-based index cache** speeds up pod startup so new deployments serve traffic faster

## What's Next

We're continuing to refine the analytics features and looking at adding more generator types. If you have feedback or run into issues, reach out on the [contact page](/contact) or open an issue on [GitHub](https://github.com/uberswe/createmod.com).
