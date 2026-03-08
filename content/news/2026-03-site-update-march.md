---
title: "March 2026 Site Update"
date: 2026-03-08
slug: march-2026-site-update
excerpt: "A big update is here: faster page loads with HTMX, multi-language support, a new upload flow, auto-translated content, and much more."
---

CreateMod.com has received its biggest update since the [full rewrite last year](/news/major-website-rewrite). Here's what's new.

## Faster, Smoother Navigation

The entire site now uses [HTMX](https://htmx.org/) for page transitions. Clicking links and submitting forms no longer triggers full page reloads, only the content that changes gets swapped in. Everything still works without JavaScript, but with it enabled pages feel significantly snappier.

## New Homepage

The homepage has been redesigned with tabbed sections for **Trending**, **Latest**, and **Highest Rated** schematics so you can quickly find what's popular or just uploaded.

![New homepage design](/assets/x/news/homepage.webp)

## Better Search

Search has been overhauled with new filtering options:

- Filter by **Minecraft version** and **Create mod version**
- Filter by **category** and **tags**
- **Search suggestions** appear as you type
- Results can be sorted by relevance, views, downloads, rating, or date
- When using best match the search results should be more relevant,

![Improved search with filters](/assets/x/news/search.webp)

## Guides

There is now a guides section to let anyone write and post useful written guides for other members to learn and benefit from. This can be anything Create Mod related and it is possible to include a video with the guide. Initially we will add some default guides on how to upload and download schematics from CreateMod.com.

## Collections

You can now create both private and public collections of schematics. This can be useful if you want to favorite some schematics of different types. By adding a description and an optional banner you can make your collection public for others to see. You might make a collection of your favorite trains or maybe a collection of a subset of your own builds.

## Multi-Language Support

The site now supports **8 languages**: English, Portuguese (Brazil & Portugal), Spanish, German, Polish, Russian, and Simplified Chinese. You can switch anytime using the flag icon in the navigation bar.

Schematic descriptions, guides, and collection descriptions are **automatically translated** so content is accessible regardless of what language the author wrote it in. Translated pages show a notice with a link to view the original.

## Private Schematics & New Upload Flow

When you upload a schematic it will be private by default. You can then choose to make it public by adding a description, images and other relevant info.

Uploading schematics has been redesigned into a clearer step-by-step process with progress indicators. You can now upload multiple NBT files at once and preview parsed schematic stats (block count, dimensions, detected mods) before publishing.

![New upload flow](/assets/x/news/upload.webp)

## Videos Page

Schematics with video links now appear on a dedicated [Videos](/videos) page, making it easy to browse video showcases.

## Explore & Mods

- The [Explore](/explore) page lets you discover random schematics
- The [Mods](/mods) page shows which mods are used across uploaded schematics, with links to Modrinth and CurseForge

## CreateMod Servers

A new **Servers** link has been added to the sidebar, connecting you to [CreateModServers.com](https://createmodservers.com) where you can find Create mod multiplayer servers.

## Under the Hood

For those interested in the technical side:

- **Database migrated from SQLite to PostgreSQL** for better scalability
- **Session-based authentication** replaces the previous token-based system
- **Background job system** for search indexing, trending calculations, sitemap generation, and translations
- **Dark mode flash fix** — the theme now loads instantly without a white flash
- Improved **schematic moderation** with AI-assisted content review
- Full pagination overhaul
- Various bugfixes and improvements

## What's Next

This update lays the groundwork for more features. If you have ideas or run into issues, feel free to reach out on the [contact page](/contact) or open an issue on [GitHub](https://github.com/uberswe/createmod.com).

Thanks for being part of the CreateMod.com community!
