package pages

import (
    "strings"
    "testing"
)

func Test_Header_Has_Sidebar_Mobile_Toggle_Button(t *testing.T) {
    d := DefaultData{IsAuthenticated: false, Language: "en", Title: "Home"}
    html := renderTemplate(t, "template/include/header.html", d)
    if !strings.Contains(html, "sidebar-mobile-toggle") {
        t.Fatalf("header should include a sidebar mobile toggle button with class=sidebar-mobile-toggle")
    }
}
