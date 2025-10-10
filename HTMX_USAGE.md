# HTMX Usage in CreateMod.com

This project uses HTMX (via CDN) for progressive enhancement without any npm/bundling. The server returns full HTML pages rendered with Go templates. HTMX swaps partials out of those full-page responses.

Key files:
- template/include/head.html: loads HTMX, enables hx-boost globally, adds a simple request indicator.
- template/include/foot.html: basic HTMX error handler and lazy-load helper.
- template/include/header.html: header search form enhanced via HTMX.
- template/search.html: advanced search form enhanced via HTMX with partial updates.

## Global conventions
- Use standard links and forms for baseline behavior (SEO, no-JS).
- Enhance with HTMX attributes:
  - hx-boost="true" is applied on <body> to intercept navigations and form submissions by default.
  - Use hx-push-url="true" where you want history updates for HTMX requests.
  - Add a request indicator with hx-indicator="#htmx-indicator" on <body> (see head.html for markup and CSS).
  - Basic error handling is attached via `window.addEventListener('htmx:responseError', ...)` in foot.html.

## Full-page vs partial updates
- Full-page replace: set hx-target="body" and hx-swap="outerHTML" to replace the entire document with the response. Example: header search form (header.html).
- Partial update of a specific container extracted from a full-page response:
  - Use hx-select="#container-id" so HTMX pulls only that element from the response.
  - Use hx-target="#container-id" and hx-swap="outerHTML" to replace the container on the current page.
  - Example: in search.html, the advanced search form updates only `#search-results` while the server still returns the whole search page.

## Search patterns
- Header search (global):
  - form: action="/search", method="post", hx-post="/search", hx-target="body", hx-swap="outerHTML", hx-push-url="true".
- Advanced search (search.html):
  - form: action="/search", method="post", hx-post="/search", hx-target="#search-results", hx-select="#search-results", hx-swap="outerHTML", hx-push-url="true".
  - Results container must have id="search-results".

## Auth/CSRF
- The backend uses cookie-based auth (see internal/router/main.go cookieAuth). HTMX requests automatically send cookies for server-rendered pages.
- For PocketBase API POSTs that require an Authorization header, the footer adds a small htmx:configRequest listener that injects the current JWT from pb.authStore.token when present.
- If a CSRF token becomes necessary, add a meta tag and set hx-headers on forms or via htmx:configRequest.

## Future conversions
- Pagination: Convert pagination links to hx-get with hx-target and hx-select to update only the results container, preserving normal anchors for SEO. If infinite scrolling is desired, use hx-trigger="revealed" and hx-swap="afterend".
- Schematic comments: A partial comments endpoint exists at /schematics/{name}/comments that renders only #comments-list. Use hx-get with hx-target="#comments-list" and hx-swap="outerHTML" to refresh comments in place. Next, replace current PocketBase JS submission with a standard form posting to a server endpoint that returns updated comments and trigger an OOB swap (hx-swap-oob) or target #comments-list.
- Settings/profile: Delete account via hx-delete to PocketBase (Authorization header is injected globally). Use hx-post for other forms; render validation errors as partial snippets.

## Error handling and UX
- A simple global handler shows alert on response errors. Replace with a toast later.
- Loading indicator: a slim top bar animates during HTMX requests.

## Notes
- Keep progressive enhancement: everything should still work with JS disabled (full page loads). HTMX only optimizes UX.
- Do not depend on npm; use CDN for third-party assets and keep small local overrides in template/static/app.css.


## Logout and redirects (HTMX)
- Logout route: GET /logout clears the auth cookie on the server. For HTMX requests (HX-Request header present), the server sets `HX-Redirect: "/"` so the client performs a full navigation. For normal requests, the server issues a 302 redirect to "/".
- Use standard links for logout: set all logout UI to `<a href="/logout">` so it works without JS and with HTMX.
- Full-page vs. partial after logout: Prefer full-page navigations after auth transitions so header and sidebar reflect the correct state. If you use `hx-target="body" hx-swap="outerHTML"`, ensure the response renders the full page with common includes so auth UI is consistent.
- Cookies: The auth cookie is configured as Path=/, SameSite=Lax, HttpOnly, and Secure in production. In development (DEV=true in .env), Secure is disabled to allow http://localhost. HTMX requests automatically include cookies for same-origin requests.
