(function (factory) {
    typeof define === 'function' && define.amd ? define(factory) :
        factory();
})((function () { 'use strict';

    var themeStorageKey = "createmodTheme";
    var defaultTheme = "dark";
    var selectedTheme;
    var params = new Proxy(new URLSearchParams(window.location.search), {
        get: function get(searchParams, prop) {
            return searchParams.get(prop);
        }
    });
    if (!!params.theme) {
        localStorage.setItem(themeStorageKey, params.theme);
        selectedTheme = params.theme;
        // Remove ?theme= from URL to avoid crawlable duplicate
        var url = new URL(window.location);
        url.searchParams.delete('theme');
        window.history.replaceState({}, '', url.pathname + url.search + url.hash);
    } else {
        var storedTheme = localStorage.getItem(themeStorageKey);
        if (storedTheme == null) {
            localStorage.setItem(themeStorageKey, defaultTheme);
        }
        selectedTheme = storedTheme ? storedTheme : defaultTheme;
    }

    function applyTheme(theme) {
        document.documentElement.setAttribute("data-bs-theme", theme);
        document.body.setAttribute("data-bs-theme", theme);
        if (theme === 'dark') {
            document.querySelectorAll('.hide-theme-light').forEach(function(el) {
                el.style.display = '';
            });
            document.querySelectorAll('.hide-theme-dark').forEach(function(el) {
                el.style.display = 'none';
            });
        } else {
            document.querySelectorAll('.hide-theme-dark').forEach(function(el) {
                el.style.display = '';
            });
            document.querySelectorAll('.hide-theme-light').forEach(function(el) {
                el.style.display = 'none';
            });
        }
    }

    applyTheme(selectedTheme);

    // Global function for theme toggle buttons
    window.setTheme = function(theme) {
        localStorage.setItem(themeStorageKey, theme);
        applyTheme(theme);
    };

}));
