package news

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"gopkg.in/yaml.v3"
)

// Post represents a single news article parsed from a markdown file.
type Post struct {
	Title   string
	Slug    string
	Date    time.Time
	Excerpt string
	Body    template.HTML // rendered HTML from markdown
	URL     string        // "/news/{slug}"
}

// frontMatter holds the YAML metadata at the top of each markdown file.
type frontMatter struct {
	Title   string `yaml:"title"`
	Date    string `yaml:"date"`
	Slug    string `yaml:"slug"`
	Excerpt string `yaml:"excerpt"`
}

// LoadAll reads all *.md files from dir within fsys, parses front matter and
// markdown, and returns posts sorted by date descending.
func LoadAll(fsys fs.FS, dir string) ([]Post, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("reading news directory: %w", err)
	}

	var posts []Post
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		p, err := parseFile(fsys, filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}
		posts = append(posts, *p)
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})

	return posts, nil
}

// LoadBySlug reads all files and returns the post with the matching slug.
// Returns nil, nil if no post matches.
func LoadBySlug(fsys fs.FS, dir, slug string) (*Post, error) {
	posts, err := LoadAll(fsys, dir)
	if err != nil {
		return nil, err
	}
	for i := range posts {
		if posts[i].Slug == slug {
			return &posts[i], nil
		}
	}
	return nil, nil
}

// parseFile reads a single markdown file and returns a Post.
func parseFile(fsys fs.FS, path string) (*Post, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}

	fm, body, err := splitFrontMatter(data)
	if err != nil {
		return nil, err
	}

	var meta frontMatter
	if err := yaml.Unmarshal(fm, &meta); err != nil {
		return nil, fmt.Errorf("invalid front matter YAML: %w", err)
	}

	if meta.Slug == "" {
		return nil, fmt.Errorf("missing slug in front matter")
	}

	date, err := time.Parse("2006-01-02", meta.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q: %w", meta.Date, err)
	}

	var htmlBuf bytes.Buffer
	if err := goldmark.Convert(body, &htmlBuf); err != nil {
		return nil, fmt.Errorf("converting markdown: %w", err)
	}

	return &Post{
		Title:   meta.Title,
		Slug:    meta.Slug,
		Date:    date,
		Excerpt: meta.Excerpt,
		Body:    template.HTML(htmlBuf.String()),
		URL:     "/news/" + meta.Slug,
	}, nil
}

// splitFrontMatter splits a markdown file into YAML front matter and body.
// Front matter is delimited by "---" lines.
func splitFrontMatter(data []byte) (frontMatter []byte, body []byte, err error) {
	content := string(data)
	content = strings.TrimSpace(content)

	if !strings.HasPrefix(content, "---") {
		return nil, nil, fmt.Errorf("missing front matter delimiter")
	}

	// Find the closing ---
	rest := content[3:] // skip opening ---
	rest = strings.TrimLeft(rest, "\r\n")
	idx := strings.Index(rest, "---")
	if idx < 0 {
		return nil, nil, fmt.Errorf("missing closing front matter delimiter")
	}

	fm := rest[:idx]
	bodyStr := strings.TrimLeft(rest[idx+3:], "\r\n")

	return []byte(fm), []byte(bodyStr), nil
}
