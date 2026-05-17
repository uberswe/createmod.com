---
title: "Security Update, Rotation Images & More"
date: 2026-05-17
slug: security-update-rotation-images
excerpt: "A major security hardening pass, rotation image support for schematics, improved search filters, and performance optimizations."
image: "/assets/x/news/security-and-features.webp"
---

We have been busy the past few weeks with a large update focused on security, new features, and performance.

## Security

A comprehensive security review has been completed and all findings have been addressed. This includes improvements to authentication, access control, rate limiting, and input validation across the site.

## Rotation Images

Schematics now support 360-degree rotation images. When uploading a schematic you can add multiple angle shots that visitors can cycle through on the schematic page.

## Short Codes

Every schematic now gets a unique short code for easy sharing. These are short alphanumeric identifiers that make it simpler to reference a specific schematic.

## Search and Mods

The search page has been updated with improved filter controls including tag and mod dropdowns, a per-page selector, and better pagination. The mods section has been expanded with detailed mod pages showing compatible schematics.

## Top Creators

The leaderboard page now shows 100 creators per page in a two-column layout with avatars and badges. If you are logged in you can see your own rank as well.

## Stats and Performance

- Site stats now include total schematics, drafts, and a daily uploads chart
- User stats page loads faster with parallelized queries and better caching
- Fixed a bug where hourly view counts were inflated

## Other Improvements

- Passkey registration UX fix
- Dark mode styling fixes
- Hull generator now supports ship presets
- New placeholder images for schematics without thumbnails
- Added Meilisearch to local development setup
- Dockerfile now runs as a non-root user

If you have feedback or run into issues, reach out on the [contact page](/contact).
