package pages

import (
    "createmod/internal/models"
    "path/filepath"
    "strings"
    "testing"
    "time"
)

func Test_News_Template_Renders_With_Posts(t *testing.T) {
    r := NewTestRegistry()

    files := append([]string{
        "./template/news.html",
    }, commonTemplates...)

    // Resolve to absolute paths from project root
    root := projectRootFromThisFile(t)
    var paths []string
    for _, f := range files {
        paths = append(paths, filepath.Join(root, f))
    }

    d := NewsData{
        DefaultData: DefaultData{Language: "en", Title: "News", Slug: "/news"},
        Posts: []models.NewsPostListItem{
            {
                ID:       "abc123",
                Title:    "Hello World",
                Excerpt:  "Welcome excerpt",
                URL:      "/news/abc123",
                PostDate: time.Now(),
            },
        },
        HourlyViews: []HourlyStat{{Hour: "12:00", Count: 5}},
        HourlyDL:    []HourlyStat{{Hour: "12:00", Count: 3}},
    }

    out, err := r.LoadFiles(paths...).Render(d)
    if err != nil {
        t.Fatalf("render news.html failed: %v", err)
    }
    if len(out) == 0 {
        t.Fatalf("rendered html is empty")
    }
    if !strings.Contains(out, "Read more") {
        t.Fatalf("expected translated button text 'Read more' in output")
    }
}
