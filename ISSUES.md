The following are issues found during testing of the website. Please fix this and ensure tests exists where needed.

- [x] The language changer is not working on some pages
  - Implemented HTMX language switching with HX-Redirect support and preserved href fallback; SetLanguageHandler already handles HX-Redirect.
- [x] The spacing between dark mode, language changer and the login button is only a few pixels, the spacing should be wider
  - Increased header row gutter from g-2 to g-3 and kept gap-2 on controls for clearer separation.
- [x] There is a lot of blank space on the main page, the columns are very thin, can we go for a more full width format to make better use of the space
  - Reworked index.html to full-width sections using container-xl with stacked rows.
- [x] There should be 3 full width sections on the main page, Trending Schematics, Highest Rated Schematics and Latest Schematics
  - Added explicit section headers and layouts for Trending, Highest Rated, and Latest.
- [x] The guides page has a guide called "How to upload a schematic" but this links to /guide instead of opening a guide. We should add a migration to add some default guides.
  - Added migration 1753295400 to update that guide to point to /upload; existing seeds retained.
- [x] The login page does not work, instead it sends the username and password as get parameters
  - Updated login form to method="post" action="/login"; added a unit test to enforce this.
- [x] Login page returned 404 on POST /login
  - Implemented server-side POST /login handler that proxies credentials to PocketBase (users auth-with-password), forwards the auth cookie, and redirects.
  - Supports HTMX via HX-Redirect (204) and normal navigation via 302. Default redirect is "/" when return_to is not provided.
- [x] When creating a collection we should allow the user to upload a banner image, suggest the appropriate size also
  - Template: Added optional banner file input to collections new/edit forms with hint “Recommended 1600x400 (4:1), max 2MB” and set forms to enctype="multipart/form-data". Kept existing banner_url field for fallback.
  - Backend: Accept multipart uploads; validate type/size; center-crop to 4:1 and resize to 1600x400; encode WebP and store as data URL in existing banner_url to avoid schema changes.
  - Display: collections_show.html renders the banner if set; otherwise shows a placeholder card with the recommended size hint.
  - Tests: Template tests for new/edit ensure file input, accept types, hint, and enctype; show template test asserts placeholder when no banner.
- [x] When viewing a schematic, the add collection form should show a dropdown with all available collections and one option should be "Create a new collection". When creating a new collection in this way a new collection should be made with the schematic that it was created from
  - UI: In schematic.html replaced free-text with a select listing the current user’s collections and a “Create new collection…” option; selecting it reveals an inline name field.
  - Backend: Extended SchematicAddToCollectionHandler to support collection="__new__" with new_collection_name, creating the collection (author=current user) and then associating the schematic. Original add-to-existing behavior retained.
  - Tests: Updated template test to assert presence of the select and the “Create new collection…” option.
- [x] Remove the old guide page
  - Redirected legacy /guide to /guides with a 301 in the router to consolidate on the new Guides listing.
- [x] Make it possible to open and close the side menu to allow more space for content, have it be closed by default
  - Added desktop toggle button in header, localStorage-persisted state, and CSS that hides the sidebar and removes page-wrapper margin when closed. Default is closed; preference is remembered across pages (HTMX supported). Added a unit test to assert toggle presence.
- [x] There are no news posts, add the news posts from the old version of the site before we started making changes. These used to be stored directly in news.html
  - Seeded initial posts via migration 1753297200, inserting if missing. News listing now reads from the PocketBase `news` collection (as implemented by handlers).
- [x] The website dark theme should use these styles
  - Applied requested dark theme palette in template/static/style.css under [data-bs-theme=dark] to exactly match the specified variables.
    --tblr-body-color: #dce1e7;
    --tblr-body-color-rgb: 220, 225, 231;
    --tblr-muted: #49566c;
    --tblr-body-bg: #1f2121;
    --tblr-body-bg-rgb: 21, 31, 44;
    --tblr-emphasis-color: #ffffff;
    --tblr-emphasis-color-rgb: 255, 255, 255;
    --tblr-bg-forms: #1f2121;
    --tblr-bg-surface: #353838;
    --tblr-bg-surface-dark: #1f2121;
    --tblr-bg-surface-secondary: #3d4040;
    --tblr-bg-surface-tertiary: #1f2121;
    --tblr-link-color: #ab813d;
    --tblr-link-hover-color: #bf9045;
    --tblr-active-bg: #3d4040;
    --tblr-disabled-color: var(--tblr-gray-700);
    --tblr-border-color: var(--tblr-dark-mode-border-color);
    --tblr-border-color-translucent: var( --tblr-dark-mode-border-color-translucent );
    --tblr-border-dark-color: var(--tblr-dark-mode-border-dark-color);
    --tblr-border-active-color: var( --tblr-dark-mode-border-active-color );
    --tblr-btn-color: #1f2121;
    --tblr-code-color: var(--tblr-body-color);
    --tblr-code-bg: #1f2e41;
    --tblr-primary-lt: #162c43;
    --tblr-primary-lt-rgb: 22, 44, 67;
    --tblr-secondary-lt: #202d3c;
    --tblr-secondary-lt-rgb: 32, 45, 60;
    --tblr-success-lt: #1a3235;
    --tblr-success-lt-rgb: 26, 50, 53;
    --tblr-info-lt: #1c3044;
    --tblr-info-lt-rgb: 28, 48, 68;
    --tblr-warning-lt: #2e2b2f;
    --tblr-warning-lt-rgb: 46, 43, 47;
    --tblr-danger-lt: #2b2634;
    --tblr-danger-lt-rgb: 43, 38, 52;
    --tblr-light-lt: #2e3947;
    --tblr-light-lt-rgb: 46, 57, 71;
    --tblr-dark-lt: #353838;
    --tblr-dark-lt-rgb: 24, 36, 51;
    --tblr-muted-lt: #202d3c;
    --tblr-muted-lt-rgb: 32, 45, 60;
    --tblr-blue-lt: #162c43;
    --tblr-blue-lt-rgb: 22, 44, 67;
    --tblr-azure-lt: #1c3044;
    --tblr-azure-lt-rgb: 28, 48, 68;
    --tblr-indigo-lt: #1c2a45;
    --tblr-indigo-lt-rgb: 28, 42, 69;
    --tblr-purple-lt: #272742;
    --tblr-purple-lt-rgb: 39, 39, 66;
    --tblr-pink-lt: #2b2639;
    --tblr-pink-lt-rgb: 43, 38, 57;
    --tblr-red-lt: #2b2634;
    --tblr-red-lt-rgb: 43, 38, 52;
    --tblr-orange-lt: #2e2b2f;
    --tblr-orange-lt-rgb: 46, 43, 47;
    --tblr-yellow-lt: #2e302e;
    --tblr-yellow-lt-rgb: 46, 48, 46;
    --tblr-lime-lt: #213330;
    --tblr-lime-lt-rgb: 33, 51, 48;
    --tblr-green-lt: #1a3235;
    --tblr-green-lt-rgb: 26, 50, 53;
    --tblr-teal-lt: #17313a;
    --tblr-teal-lt-rgb: 23, 49, 58;
    --tblr-cyan-lt: #183140;
    --tblr-cyan-lt-rgb: 24, 49, 64;
    --tblr-x-lt: #16202e;
    --tblr-x-lt-rgb: 22, 32, 46;
    --tblr-facebook-lt: #182c46;
    --tblr-facebook-lt-rgb: 24, 44, 70;
    --tblr-twitter-lt: #193146;
    --tblr-twitter-lt-rgb: 25, 49, 70;
    --tblr-linkedin-lt: #172b41;
    --tblr-linkedin-lt-rgb: 23, 43, 65;
    --tblr-google-lt: #2c2834;
    --tblr-google-lt-rgb: 44, 40, 52;
    --tblr-youtube-lt: #2f202e;
    --tblr-youtube-lt-rgb: 47, 32, 46;
    --tblr-vimeo-lt: #183345;
    --tblr-vimeo-lt-rgb: 24, 51, 69;
    --tblr-dribbble-lt: #2d283c;
    --tblr-dribbble-lt-rgb: 45, 40, 60;
    --tblr-github-lt: #182330;
    --tblr-github-lt-rgb: 24, 35, 48;
    --tblr-instagram-lt: #2c2737;
    --tblr-instagram-lt-rgb: 44, 39, 55;
    --tblr-pinterest-lt: #292131;
    --tblr-pinterest-lt-rgb: 41, 33, 49;
    --tblr-vk-lt: #202e3f;
    --tblr-vk-lt-rgb: 32, 46, 63;
    --tblr-rss-lt: #2f312e;
    --tblr-rss-lt-rgb: 47, 49, 46;
    --tblr-flickr-lt: #162a44;
    --tblr-flickr-lt-rgb: 22, 42, 68;
    --tblr-bitbucket-lt: #162942;
    --tblr-bitbucket-lt-rgb: 22, 41, 66;
    --tblr-tabler-lt: #162c43;
    --tblr-tabler-lt-rgb: 22, 44, 67;
- [x] Can you redo the trending algorithm so that it slowly makes old schematics receive a penalty and are less likely to show in trending sort of like how reddit posts work?
  - Implemented decayed trending score in backend: score = (views_48h + 2.0*log1p(ratings_sum)) / pow(age_hours+2, 1.8).
  - Aggregates recent views (last 48h) and ratings sum per schematic; scoring and sorting done in Go; results cached via existing cache service.
  - Added unit tests (internal/pages/trending_test.go) to assert newer items with similar engagement outrank older ones, and that substantially higher engagement can overcome age penalty.
  - Refactored getTrendingSchematics in internal/pages/index.go to use the new score with a safe fallback behavior.
- [x] I get an error viewing the news page
  - Fixed template/news.html to use root context for Language inside the posts range; added regression test internal/pages/news_template_test.go to prevent regressions. News page renders correctly.
  ERROR GET /news                                                                                                                                                                                                                                                                                
  └─ template: news.html:22:89: executing "news.html" at <.Language>: can't evaluate field Language in type models.NewsPostListItem
  [0.00ms] SELECT `users`.* FROM `users` WHERE `users`.`id`='30pfwjgehl9ymdi' LIMIT 1
  [0.00ms] SELECT `_collections`.* FROM `_collections` WHERE `id`='schematics' OR LOWER(`name`)='schematics' LIMIT 1                                                                                                                                                                             
  [18.00ms] SELECT `schematics`.* FROM `schematics` WHERE ((`schematics`.`deleted` = '' OR `schematics`.`deleted` IS NULL) AND `schematics`.`moderated` = 1 AND ((`schematics`.`scheduled_at` = '' OR `schematics`.`scheduled_at` IS NULL) OR `schematics`.`scheduled_at` <= '2025-10-13 07:27:01.049089 +0200 CEST m=+228.364381292')) ORDER BY `schematics`.`created` DESC LIMIT 50                                                                                                                                                                                                           
  [231.00ms] SELECT `schematics`.*, avg(schematic_views.count) AS `avg_views` FROM `schematic_views` LEFT JOIN `schematics` ON schematic_views.schematic = schematics.id WHERE (schematic_views.type = 0) AND (schematic_views.created > (SELECT DATETIME('now', '-2 day'))) GROUP BY `schematics`.`id` ORDER BY `avg_views` DESC LIMIT 10                                                                                                                                                                                                                                                      
  [0.00ms] SELECT `schematics`.*, avg(schematic_ratings.rating) AS `avg_rating`, count(schematic_ratings.rating) AS `total_rating` FROM `schematics` LEFT JOIN `schematic_ratings` ON schematic_ratings.schematic = schematics.id WHERE schematics.deleted = null AND schematics.moderated = true AND (schematics.scheduled_at IS NULL OR schematics.scheduled_at <= DATETIME('now')) GROUP BY `schematics`.`id` ORDER BY `avg_rating` DESC, `total_rating` DESC LIMIT 10                                                                                                                       
  [0.00ms] SELECT `_collections`.* FROM `_collections` WHERE `id`='schematic_tags' OR LOWER(`name`)='schematic_tags' LIMIT 1                                                                                                                                                                     
  [0.00ms] SELECT `schematic_tags`.* FROM `schematic_tags` WHERE 1 = 1 ORDER BY `schematic_tags`.`name` ASC                                                                                                                                                                                      
  [0.00ms] SELECT `schematics`.`tags` FROM `schematics`                                                                                                                                                                                                                                          
  [0.00ms] SELECT `schematics`.* FROM `schematics` WHERE ((`schematics`.`deleted` = '' OR `schematics`.`deleted` IS NULL) AND `schematics`.`author` = '30pfwjgehl9ymdi') ORDER BY `schematics`.`created` DESC LIMIT 1                                                                            
  INFO GET /.well-known/appspecific/com.chrome.devtools.json                                                                                                                                                                                                                                     



## 2025-10-13 – Planning notes (historical, all items completed above)

All items from the original planning notes have been fully implemented. The collection banner upload, add-to-collection dropdown, and trending algorithm are complete. The "caching" sub-item for collections was reviewed and found to be a non-issue: collections handlers query PocketBase directly without caching, so there is no stale cache to invalidate.


## 2026-02-23 – Verification pass

Ran Go tests (`go test ./...`) and Playwright E2E tests against the running server.

- [x] Fixed `cmd/moderate` build error: `client.CheckMinecraftBuildImage` was called but not defined on the `openai.Client` type. Added the missing method to `internal/openai/client.go` using the Responses API.
- [x] Added Playwright `page_health.spec.ts` with 28 tests covering all major pages (200 status), login form POST method, HTMX search, news template rendering, banner upload input, sidebar toggle, dark mode toggle, and language changer presence.
- [x] Verified all 15 original issues are fully resolved via screenshots and automated tests.
- All Go tests pass. All Playwright tests pass (31 active, 5 scaffolded/skipped).
