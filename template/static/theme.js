// theme.js — runs after body is loaded.
// window.setTheme is already defined inline in head.html so it is available
// before any onclick handler can fire. This script handles:
//   1. ?theme= query parameter override
//   2. Initial display-toggle of theme buttons (needs body to exist)
//   3. Clearing the anti-FOUC inline background
(function () {
    'use strict';

    var themeStorageKey = "createmodTheme";
    var defaultTheme = "dark";

    // Handle ?theme= query parameter
    var params = new URLSearchParams(window.location.search);
    var paramTheme = params.get('theme');
    if (paramTheme) {
        localStorage.setItem(themeStorageKey, paramTheme);
        // Remove ?theme= from URL to avoid crawlable duplicate
        var url = new URL(window.location);
        url.searchParams.delete('theme');
        window.history.replaceState({}, '', url.pathname + url.search + url.hash);
    }

    // Determine the active theme
    var selectedTheme = paramTheme || localStorage.getItem(themeStorageKey) || defaultTheme;
    if (!localStorage.getItem(themeStorageKey)) {
        localStorage.setItem(themeStorageKey, defaultTheme);
    }

    // Apply theme via the inline-defined setTheme (guaranteed to exist)
    if (typeof window.setTheme === 'function') {
        window.setTheme(selectedTheme);
    }

    // Clear the inline background set by the head script now that CSS has loaded.
    document.documentElement.style.background = "";
})();
