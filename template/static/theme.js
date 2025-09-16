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
    } else {
        var storedTheme = localStorage.getItem(themeStorageKey);
        if (storedTheme == null) {
            localStorage.setItem(themeStorageKey, defaultTheme);
        }
        selectedTheme = storedTheme ? storedTheme : defaultTheme;
    }
    console.log("selected theme" + selectedTheme)
    if (selectedTheme === 'dark') {
        document.body.setAttribute("data-bs-theme", selectedTheme);
        // Remove hide-theme-light classes when theme is dark
        document.querySelectorAll('.hide-theme-light').forEach(function(element) {
            element.classList.remove('hide-theme-light');
        });
    } else {
        document.body.setAttribute("data-bs-theme", selectedTheme);
        // Remove hide-theme-dark classes when theme is not dark
        document.querySelectorAll('.hide-theme-dark').forEach(function(element) {
            element.classList.remove('hide-theme-dark');
        });
    }

}));