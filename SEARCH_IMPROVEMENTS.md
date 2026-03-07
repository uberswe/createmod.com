# CreateMod.com Search Improvements — Claude Code Prompt

## Context

CreateMod.com is an open-source Go web application (github.com/uberswe/createmod) that hosts Minecraft Create mod schematics. It uses **Bleve** for full-text search and serves HTML templates with a Tabler UI framework. The site has ~100+ schematics across 17 categories and 53 tags.

This document specifies search improvements based on competitive analysis of praised platforms (Modrinth, Printables, Nexus Mods) and common failures on CurseForge, Thingiverse, Steam Workshop, and DeviantArt.

---

## Current State (What Exists)

**Search page:** `/search/` with these query parameters:
- `sort` — integer (0=best match, 1=newest, 2=oldest, 3=highest rating, 4=lowest rating, 5=least views, 6=most views)
- `rating` — minimum star filter (-1=any, 1-5)
- `category` — slug or "all"
- `tag` — slug or "all"
- `q` — search text (when provided)

**Current result cards show:** title, thumbnail, Create version, Minecraft version, category label. That's it.

**Detail pages contain but search cards DO NOT show:** star rating, view count, author name, upload date, tags, download count.

**AI image descriptions exist in the database** but are NOT currently indexed by Bleve. These are AI-generated text descriptions of schematic screenshots that describe visible components, layout, and purpose. They are a high-value untapped search signal, especially for schematics with vague or minimal titles/descriptions.

**Known limitations:**
- Advanced filters (category, tag, MC version, Create version) are hidden behind a collapsed "Show advanced options" panel
- Results hard-cap at ~100 items with NO pagination
- Only one tag can be selected at a time
- No autocomplete/search suggestions
- Default sort is "Most views" which buries new content (and doesn't switch to "Best match" when a text query is entered)
- No URL-based pagination state
- A trending algorithm already exists in the codebase and can be reused

---

## Changes to Implement

### 1. Enrich Search Result Cards

**Priority: HIGHEST — biggest UX win for least effort**

Each search result card currently shows only title + thumbnail + version info. Add the following metadata to every card:

- **Star rating** — show as filled/empty stars or a numeric rating (e.g., "4.2 ★")
- **View count** — formatted with abbreviation for large numbers (e.g., "12.3k views")
- **Author name** — linked to their author profile page
- **Upload date** — relative format (e.g., "3 months ago") with ISO date in a `title` attribute for hover
- **Tags** — show first 2-3 tags as small badges on the card

This data already exists on detail pages and in the database — it just needs to be included in the search results template and the search query/response struct.

### 2. Expose Filters — Remove the Collapsed Panel

**Priority: HIGH**

The "Show advanced options" collapsible hides the most useful filters. Users don't discover them.

- **Remove the collapse behavior** — all filters should be visible by default
- **Desktop layout:** Move filters to a left sidebar. Search bar and sort stay above the results grid. Sidebar contains: Category, Tag (multi-select), Minecraft Version, Create Version, Minimum Rating
- **Mobile layout:** Keep filters behind a "Filters" button that opens a drawer/modal, but show active filter count on the button (e.g., "Filters (3)")
- **Show result counts next to filter values** where feasible (e.g., "Farms (23)", "Iron (7)") — this helps users predict which filters are useful before clicking

### 3. Add Pagination with SEO-Optimized URLs

**Priority: HIGH**

Currently results cap at ~100 with no way to access more content.

**URL structure — use path-based pagination for SEO:**

```
/search/                          → page 1 (default)
/search/page/2/                   → page 2
/search/page/3/                   → page 3
```

All filter and sort state MUST be preserved in query parameters:

```
/search/page/2/?sort=6&category=farms&tag=iron&rating=3&q=automatic
```

**SEO requirements:**
- Each page must have a unique, descriptive `<title>` tag: `"Farms Schematics - Page 2 | CreateMod.com"` or `"Search results for 'automatic' - Page 2 | CreateMod.com"`
- Add `<link rel="canonical">` pointing to the current page URL
- Add `<link rel="prev">` and `<link rel="next">` for pagination (even though Google has deprecated these, other crawlers use them and they don't hurt)
- Page 1 canonical should be `/search/?...` (without `/page/1/`)
- Add `<meta name="robots" content="noindex">` on pages beyond a reasonable depth (e.g., page 20+) to avoid thin content indexing
- Pagination parameters must NOT use hash fragments (`#page=2`) — they must be real URLs that work without JavaScript

**Implementation details:**
- Default to 24 results per page (fits a 4-column grid nicely)
- Show pagination controls at both top and bottom of results
- Pagination UI: First, Prev, [page numbers with ellipsis], Next, Last
- Preserve all filter/sort/query state when navigating pages
- Show total result count: "Showing 25-48 of 342 schematics"
- When filters change, reset to page 1

### 4. Add Trending Sort Option + Smart Sort Defaults

**Priority: MEDIUM**

A trending algorithm already exists in the codebase.

- Add "Trending" as a new sort option (consider making it the new default sort instead of "Most views")
- Trending should factor in recent views/downloads over a time window (e.g., last 7 or 30 days) to give new popular content a chance to surface
- The existing trending algorithm should be wired into the sort dropdown

**Smart sort default behavior:**

The sort order should automatically adapt based on whether the user is browsing or searching:

- **No search query (`q` is empty):** Default sort = "Most views" (sort=6). This is the browse/explore mode — users want to see popular content.
- **Search query present (`q` has a value):** Default sort = "Best match" (sort=0). When someone types a search, relevance should be the default — that's the whole point of searching.
- **User explicitly selects a sort:** If the user manually picks a sort option from the dropdown, that choice takes priority regardless of whether `q` is set. The auto-switch only applies when no explicit sort has been chosen.

**Implementation approach:**
- If the URL has a `sort` parameter, always respect it (the user chose it explicitly)
- If the URL has NO `sort` parameter: use sort=0 (best match) when `q` is non-empty, use sort=6 (most views) when `q` is empty
- The sort dropdown should visually reflect the active sort at all times
- When a user types a query and hits search, if they haven't explicitly set a sort, the dropdown should update to show "Best match" as the active selection

### 5. Add Multi-Tag Selection

**Priority: MEDIUM**

Currently only one tag can be selected. Allow selecting multiple tags simultaneously.

**URL format for multi-tag:**
```
/search/?tag=iron,train,farm
```

Use comma-separated tag slugs in the `tag` parameter.

**UI behavior:**
- Change the tag dropdown to a multi-select with checkboxes
- Selected tags appear as removable badges/chips above or within the filter area
- Multiple tags use AND logic (results must match ALL selected tags)
- Show a "Clear all" option when multiple tags are selected

### 6. Improve Search Relevance

**Priority: MEDIUM**

Configure Bleve to weight fields appropriately:

- **Title matches: highest weight** (e.g., 10x) — searching "iron farm" should always return schematics with "iron farm" in the title first
- **Tag matches: high weight** (e.g., 5x)
- **AI image description matches: medium weight** (e.g., 3x) — see below
- **Description/body matches: base weight** (1x)

**Index AI-generated image descriptions:**

The database already contains AI-generated descriptions of schematic images. These should be added to the Bleve index as a new searchable field (e.g., `ai_description` or `image_description`). This is especially valuable because many schematics have uninformative titles ("My Build", "Factory v2") or empty/minimal descriptions — the AI descriptions capture what's actually visible in the screenshot (conveyor belts, trains, mechanical crafters, farm layouts, etc.) and will match user search queries that the human-written metadata misses.

Implementation:
- Add the AI description field to the Bleve document mapping
- Weight it at ~3x (higher than user-written descriptions since they tend to be more detailed and accurate, but lower than title and tags which represent intentional labeling by the creator)
- When re-indexing, populate this field from the existing database column
- If a schematic has no AI description, the field should simply be empty/omitted — it won't affect scoring for that document
- Consider running a one-time re-index after adding this field so all existing schematics benefit immediately

Additional relevance improvements:
- Exact phrase matches should rank above individual word matches
- Multi-word queries should use AND logic by default (all words must appear), not OR
- If no AND results are found, fall back to OR with a "showing results containing any of your terms" notice

### 7. Add Autocomplete / Search Suggestions

**Priority: LOW (can be a follow-up)**

When users type in the search box, show a dropdown with:

- **Schematic title matches** (top 5) — show thumbnail + title + category
- **Tag suggestions** — if the typed text matches a tag name, suggest it as a filter (e.g., typing "iro" suggests adding "Iron" tag filter)
- **Category suggestions** — similarly for categories

Implementation notes:
- Create an API endpoint like `/api/search/suggest?q=...` that returns JSON
- Debounce input by 200-300ms before querying
- Show suggestions after 2+ characters
- Target sub-200ms response time
- Keyboard navigation (arrow keys + enter) should work in the dropdown

### 8. Add Grid/List View Toggle

**Priority: LOW**

Add a toggle button near the sort dropdown to switch between:

- **Grid view** (current default) — large thumbnails in a card grid
- **List view** — compact rows with small thumbnail, title, author, rating, views, date, tags all on one line

Store the preference in a cookie or localStorage so it persists across visits.

### 9. Modernize the Search Page Design

**Priority: HIGH**

The current search page uses basic Tabler defaults and feels utilitarian. Modernize the design to feel polished and purpose-built for a creative community site. Take inspiration from Modrinth's clean, spacious design and Printables' visual-first approach.

**Overall layout — 3-column desktop design:**

```
┌─────────────────────────────────────────────────────────────────┐
│  Search bar (full width)              [Sort dropdown] [Grid/List]│
├──────────┬──────────────────────────────────┬───────────────────┤
│          │                                  │                   │
│  Filters │     Results grid                 │   Ad spot         │
│  sidebar │     (cards with rich metadata)   │   (sticky)        │
│          │                                  │                   │
│  Category│     ┌──────┐ ┌──────┐ ┌──────┐  │   ┌───────────┐  │
│  Tags    │     │ Card │ │ Card │ │ Card │  │   │           │  │
│  MC ver  │     └──────┘ └──────┘ └──────┘  │   │  Ad unit   │  │
│  Create  │     ┌──────┐ ┌──────┐ ┌──────┐  │   │  300x250   │  │
│  Rating  │     │ Card │ │ Card │ │ Card │  │   │  or        │  │
│          │     └──────┘ └──────┘ └──────┘  │   │  160x600   │  │
│          │                                  │   │           │  │
│          │     [Pagination controls]        │   └───────────┘  │
├──────────┴──────────────────────────────────┴───────────────────┤
│  Footer                                                         │
└─────────────────────────────────────────────────────────────────┘
```

**Column widths (approximate):**
- Left sidebar (filters): ~220px fixed
- Center (results): fluid, takes remaining space
- Right sidebar (ad): ~300px fixed

**Specific design improvements:**

**Search bar area:**
- Make the search input larger and more prominent — it should feel like the primary action on the page
- Add a subtle search icon inside the input field
- The sort dropdown and view toggle should sit on the right side of the same row, aligned with the search bar
- Show the result count and active filters summary in a bar between the search input and the results (e.g., "342 schematics in Farms tagged Iron — sorted by Most views")

**Result cards:**
- Use subtle hover effects — slight elevation/shadow increase on hover, smooth transition (150-200ms)
- Thumbnail should have a consistent aspect ratio (16:9) with `object-fit: cover` to prevent stretching
- Rating stars should use a warm accent color (amber/gold)
- View count and date should be slightly muted (secondary text color) to not compete with the title
- Author name in a distinct style (could be a small avatar + name if available)
- Tags as small rounded pills/badges with a subtle background color
- Version compatibility badges should be compact and use color coding (green for latest MC version, gray for older)

**Filter sidebar:**
- Each filter group should have a clear heading with a subtle separator
- Collapsible filter groups (but all expanded by default) — users can collapse ones they don't need to save space
- Active filters should be visually distinct (highlighted background or checkmark)
- A "Clear all filters" link at the top of the sidebar when any filter is active
- Smooth transitions when filter counts update

**Typography and spacing:**
- Use consistent vertical rhythm — even spacing between cards, between filter groups, between sections
- Card titles should be 1-2 lines max with text truncation (ellipsis) for long titles
- Ensure sufficient padding inside cards — the current cards may feel cramped with all the new metadata

**Color and visual polish:**
- Maintain the existing site's color scheme but ensure sufficient contrast
- Use subtle background color differentiation between the sidebar, results area, and ad column
- Add a subtle top border or background to the active sort option
- Pagination controls should feel clickable — clear hover states, adequate touch targets (min 44px)

**Responsive breakpoints:**
- **Desktop (>1200px):** Full 3-column layout — filters sidebar, results grid (3 columns of cards), ad sidebar
- **Tablet (768-1200px):** 2-column layout — filters collapse to top horizontal bar or drawer, results grid (2-3 columns of cards), ad moves to below results or inline between result rows
- **Mobile (<768px):** Single column — filters behind drawer button, results grid (1-2 columns), ad moves inline between results (e.g., after every 6th result)

### 10. Sticky Ad Spot — Right Sidebar

**Priority: HIGH**

Reserve a fixed-width right sidebar column for advertising that follows the user as they scroll.

**Desktop implementation:**
- The right sidebar should be ~300px wide (standard ad sizes: 300x250 medium rectangle, 300x600 half page, or 160x600 wide skyscraper)
- Use `position: sticky; top: 20px;` on the ad container so it floats with the viewport as the user scrolls through results
- The sticky behavior should stop before the footer — the ad should not overlap the footer. Use a wrapper with a defined bottom boundary so the sticky element stops at the right place
- Add a clear visual boundary between the results area and the ad column (subtle border or background color difference) so the ad doesn't feel like part of the content
- The ad container should have a minimum height placeholder (e.g., a light gray background with "Advertisement" label) even when no ad is loaded, to prevent layout shift

**CSS approach for sticky with footer boundary:**
```css
.search-layout {
  display: grid;
  grid-template-columns: 220px 1fr 300px;
  gap: 24px;
  align-items: start; /* important for sticky to work in grid */
}

.ad-sidebar {
  position: sticky;
  top: 20px;
  /* The sticky element naturally stops when its parent ends,
     so the grid cell height = results column height = natural boundary */
}
```

**Ad container HTML structure:**
```html
<aside class="ad-sidebar">
  <div class="ad-unit" id="search-sidebar-ad">
    <!-- Ad code injected here -->
  </div>
</aside>
```

**Responsive behavior:**
- **Desktop (>1200px):** Sticky right sidebar, always visible alongside results
- **Tablet (768-1200px):** Ad moves to a horizontal banner between result rows (e.g., after row 2) or below the results grid. No sticky behavior.
- **Mobile (<768px):** Ad appears inline within the results feed, after every 6th-8th result card. Native-feeling placement, not sticky.

**Important constraints:**
- The ad column must NOT compress the results grid. The results area should have a fluid min-width that ensures at least 2 card columns on desktop before the ad column is hidden
- If the viewport is between 1000-1200px and 3 columns (filters + results + ad) don't fit comfortably, collapse the filter sidebar to a top bar first, keeping the ad visible
- The ad container should be a simple empty `<div>` with an ID — the actual ad script/content will be inserted separately (don't hardcode any ad network code)
- Ensure the sticky ad doesn't cause content jumps or reflow when it becomes fixed

---

## URL Design Summary

The final URL structure should look like:

```
# Browse mode (no query) — defaults to most views
/search/

# Search mode (has query) — defaults to best match automatically
/search/?q=iron+farm

# User explicitly chose a sort — always respected
/search/?q=iron+farm&sort=6&category=farms&tag=iron,train&rating=3&mc=1.20.X&create=0.5.1

# Paginated
/search/page/2/?q=iron+farm&sort=6&category=farms&tag=iron,train&rating=3

# Category browse (no search text) — defaults to most views
/search/?category=farms

# Trending
/search/?sort=7
```

**Key rules:**
- Page parameter is path-based: `/search/page/N/`
- All other parameters are query strings
- Page 1 never shows `/page/1/` in the URL
- Changing any filter resets to page 1
- All parameters are optional — bare `/search/` works with defaults
- Parameter names should be short but readable: `q`, `sort`, `category`, `tag`, `rating`, `mc`, `create`, optionally `per_page`
- **Sort defaults:** When `sort` param is absent, use best match (0) if `q` is non-empty, most views (6) if `q` is empty. When `sort` param is present, always use it.

---

## Technical Notes

- The site uses Go with HTML templates and Bleve for search
- Tabler CSS framework is used for the UI — leverage its existing components (badges, cards, pagination, sidebar, form elements)
- The search handler likely lives in a Go handler function that processes the query parameters, runs a Bleve search, and renders the template
- Bleve supports field boosting via `SetBoost()` on query fields — use this for the relevance weighting
- For pagination, the Bleve `Search` request accepts `From` and `Size` parameters — use `From = (page - 1) * perPage` and `Size = perPage`
- For multi-tag support, build a conjunction query (AND) of multiple tag match queries
- **AI image descriptions** are stored in the database alongside schematics. Add an `ai_description` (or similar) text field to the Bleve document mapping. Use `fieldMapping.SetBoost(3.0)` for this field. After updating the mapping, trigger a full re-index so existing schematics pick up the new field. The AI descriptions tend to mention specific Create mod components (gears, belts, mechanical crafters, deployers) and layout details that users search for but creators don't always include in titles.
- The existing trending algorithm should already be available as a sort/scoring function — add it as sort option 7
- **Sort default logic** should live in the Go handler: check if `sort` param exists in the URL. If not, check if `q` is non-empty → use sort=0, else use sort=6. Pass the resolved sort value to the template so the dropdown reflects it.
- **Layout uses CSS Grid** — the 3-column search layout (filters | results | ad) should use `display: grid; grid-template-columns: 220px 1fr 300px;` with `align-items: start` to enable sticky positioning within grid cells
- **Sticky ad** uses `position: sticky; top: 20px;` — no JavaScript needed. The sticky element naturally stops when its grid cell parent ends (which aligns with the results column height), preventing footer overlap.
- The ad container should be a plain `<aside>` with an ID — do not hardcode any ad network scripts. The ad code will be injected separately.

---

## Testing Checklist

After implementation, verify:

**Search result cards:**
- [ ] Cards show: title, thumbnail, rating, views, author, date, tags, version info
- [ ] Hover effects work smoothly on cards (elevation/shadow transition)
- [ ] Thumbnails maintain 16:9 aspect ratio without stretching
- [ ] Long titles truncate with ellipsis

**Filters:**
- [ ] All filters visible by default on desktop (left sidebar layout)
- [ ] Filter groups are collapsible but expanded by default
- [ ] "Clear all filters" link appears when any filter is active
- [ ] Result counts show next to filter values where feasible

**Pagination:**
- [ ] Pagination works and URLs are bookmarkable
- [ ] Navigating to `/search/page/3/?category=farms&sort=6` directly works (server-side rendered)
- [ ] Page titles and canonical URLs are correct for each paginated page
- [ ] `<link rel="prev">` and `<link rel="next">` are present
- [ ] Page 1 canonical has no `/page/1/` in the URL
- [ ] When filters change, user is sent back to page 1

**Sort behavior:**
- [ ] Browse mode (no `q`): defaults to "Most views" when `sort` param is absent
- [ ] Search mode (has `q`): defaults to "Best match" when `sort` param is absent
- [ ] Explicit sort param in URL always takes priority over defaults
- [ ] Sort dropdown visually reflects the active sort at all times
- [ ] "Trending" appears in the sort dropdown

**Search quality:**
- [ ] Searching an exact schematic title returns it as the first result
- [ ] Multi-word searches use AND logic (all terms must appear)
- [ ] Multi-tag selection works with comma-separated URL params
- [ ] Empty search with no filters shows all schematics (paginated)
- [ ] AI image descriptions are indexed — searching for a component (e.g., "mechanical crafter") returns schematics that show one in their screenshot even if the title/description don't mention it
- [ ] A full re-index has been run after adding the AI description field

**Layout and design:**
- [ ] 3-column layout on desktop: filters (left) + results (center) + ad (right)
- [ ] Grid/list toggle works and persists across page navigations
- [ ] Mobile: filters accessible via drawer, pagination works
- [ ] Tablet: graceful degradation — ad repositions, filters collapse

**Ad spot:**
- [ ] Right sidebar ad container is ~300px wide on desktop
- [ ] Ad is sticky and follows viewport scroll
- [ ] Sticky ad stops before overlapping the footer
- [ ] Ad container has placeholder styling when empty (no layout shift)
- [ ] Ad repositions to inline on tablet/mobile
- [ ] Ad column never compresses the results grid below 2 card columns

**Backward compatibility:**
- [ ] Existing search URLs (`/search/?sort=6&rating=-1&category=all&tag=all`) still work
- [ ] Category nav links in the sidebar (`/search?category=builds&sort=6`) still work
