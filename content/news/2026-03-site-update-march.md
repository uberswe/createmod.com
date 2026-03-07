---
title: "March 2026 Site Update"
date: 2026-03-08
slug: march-2026-site-update
excerpt: "A big update is here: faster page loads with HTMX, multi-language support, a new upload flow, auto-translated content, and much more."
---

CreateMod.com has received its biggest update since the [full rewrite last year](/news/major-website-rewrite). Here's what's new.

## Faster, Smoother Navigation

The entire site now uses [HTMX](https://htmx.org/) for page transitions. Clicking links and submitting forms no longer triggers full page reloads — only the content that changes gets swapped in. Everything still works without JavaScript, but with it enabled pages feel significantly snappier.

<!-- TODO: Add screenshot of the new page transition in action -->

## New Homepage

The homepage has been redesigned with tabbed sections for **Trending**, **New**, and **Top** schematics so you can quickly find what's popular or just uploaded.

<!-- TODO: Add screenshot of the new tabbed homepage -->

## Better Search

Search has been overhauled with new filtering options:

- Filter by **Minecraft version** and **Create mod version**
- Filter by **category** and **tags**
- **Search suggestions** appear as you type
- Results can be sorted by relevance, views, downloads, rating, or date

<!-- TODO: Add screenshot of search with filters expanded -->

## Multi-Language Support

The site now supports **8 languages**: English, Portuguese (Brazil & Portugal), Spanish, German, Polish, Russian, and Simplified Chinese. Your language is detected automatically from your browser, and you can switch anytime using the flag icon in the navigation bar.

Schematic descriptions, guides, and collection descriptions are **automatically translated** so content is accessible regardless of what language the author wrote it in. Translated pages show a notice with a link to view the original.

## New Upload Flow

Uploading schematics has been redesigned into a clearer step-by-step process with progress indicators. You can now upload multiple NBT files at once and preview parsed schematic stats (block count, dimensions, detected mods) before publishing.

<!-- TODO: Add screenshot of the new upload publish page -->

## Guides & Collections Improvements

- Guide and collection descriptions are now auto-translated alongside schematics
- Collections have a cleaner detail page layout

## Videos Page

Schematics with video links now appear on a dedicated [Videos](/videos) page, making it easy to browse video showcases.

## Explore & Mods

- The [Explore](/explore) page lets you discover random schematics
- The [Mods](/mods) page shows which mods are used across uploaded schematics, with links to Modrinth and CurseForge

## CreateMod Servers

A new **Servers** link has been added to the sidebar, connecting you to [CreateModServers.com](https://createmodservers.com) where you can find Create mod multiplayer servers.

## Under the Hood

For those interested in the technical side:

- **Database migrated from SQLite to PostgreSQL** for better performance and reliability
- **Session-based authentication** replaces the previous token-based system
- **S3 file storage** for schematic files and images
- **Background job system** for search indexing, trending calculations, sitemap generation, AI descriptions, and translations
- **Dark mode flash fix** — the theme now loads instantly without a white flash
- Improved **schematic moderation** with AI-assisted content review
- Admin tools for managing schematics, tags, and categories

## What's Next

This update lays the groundwork for more features. If you have ideas or run into issues, feel free to reach out on the [contact page](/contact) or open an issue on [GitHub](https://github.com/uberswe/createmod.com).

Thanks for being part of the CreateMod.com community!
