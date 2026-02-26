package news

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadAll_SortsByDateDesc(t *testing.T) {
	fsys := fstest.MapFS{
		"news/older.md": &fstest.MapFile{Data: []byte(`---
title: "Older Post"
date: 2025-01-01
slug: older
excerpt: "Old"
---

Old content.
`)},
		"news/newer.md": &fstest.MapFile{Data: []byte(`---
title: "Newer Post"
date: 2025-02-15
slug: newer
excerpt: "New"
---

New content.
`)},
	}

	posts, err := LoadAll(fsys, "news")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}

	if posts[0].Slug != "newer" {
		t.Errorf("expected newest first, got slug %q", posts[0].Slug)
	}
	if posts[1].Slug != "older" {
		t.Errorf("expected oldest second, got slug %q", posts[1].Slug)
	}
}

func TestLoadAll_RendersMarkdownToHTML(t *testing.T) {
	fsys := fstest.MapFS{
		"news/test.md": &fstest.MapFile{Data: []byte(`---
title: "Test"
date: 2025-03-01
slug: test
excerpt: "A test"
---

This is **bold** text.
`)},
	}

	posts, err := LoadAll(fsys, "news")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}

	body := string(posts[0].Body)
	if !strings.Contains(body, "<strong>bold</strong>") {
		t.Errorf("expected <strong>bold</strong> in body, got: %s", body)
	}
}

func TestLoadAll_SetsURL(t *testing.T) {
	fsys := fstest.MapFS{
		"news/post.md": &fstest.MapFile{Data: []byte(`---
title: "Post"
date: 2025-01-01
slug: my-post
excerpt: "Excerpt"
---

Body.
`)},
	}

	posts, err := LoadAll(fsys, "news")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if posts[0].URL != "/news/my-post" {
		t.Errorf("expected URL /news/my-post, got %s", posts[0].URL)
	}
}

func TestLoadBySlug_Found(t *testing.T) {
	fsys := fstest.MapFS{
		"news/a.md": &fstest.MapFile{Data: []byte(`---
title: "A"
date: 2025-01-01
slug: alpha
excerpt: "A"
---

A content.
`)},
		"news/b.md": &fstest.MapFile{Data: []byte(`---
title: "B"
date: 2025-01-02
slug: beta
excerpt: "B"
---

B content.
`)},
	}

	post, err := LoadBySlug(fsys, "news", "alpha")
	if err != nil {
		t.Fatalf("LoadBySlug: %v", err)
	}
	if post == nil {
		t.Fatal("expected post, got nil")
	}
	if post.Title != "A" {
		t.Errorf("expected title 'A', got %q", post.Title)
	}
}

func TestLoadBySlug_NotFound(t *testing.T) {
	fsys := fstest.MapFS{
		"news/a.md": &fstest.MapFile{Data: []byte(`---
title: "A"
date: 2025-01-01
slug: alpha
excerpt: "A"
---

A content.
`)},
	}

	post, err := LoadBySlug(fsys, "news", "nonexistent")
	if err != nil {
		t.Fatalf("LoadBySlug: %v", err)
	}
	if post != nil {
		t.Errorf("expected nil for nonexistent slug, got %+v", post)
	}
}

func TestLoadAll_SkipsNonMarkdown(t *testing.T) {
	fsys := fstest.MapFS{
		"news/post.md": &fstest.MapFile{Data: []byte(`---
title: "Post"
date: 2025-01-01
slug: post
excerpt: "E"
---

Body.
`)},
		"news/readme.txt": &fstest.MapFile{Data: []byte(`not markdown`)},
	}

	posts, err := LoadAll(fsys, "news")
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(posts) != 1 {
		t.Errorf("expected 1 post (skipping .txt), got %d", len(posts))
	}
}

func TestLoadAll_MissingSlug(t *testing.T) {
	fsys := fstest.MapFS{
		"news/bad.md": &fstest.MapFile{Data: []byte(`---
title: "Bad"
date: 2025-01-01
slug: ""
excerpt: "E"
---

Body.
`)},
	}

	_, err := LoadAll(fsys, "news")
	if err == nil {
		t.Fatal("expected error for missing slug")
	}
}

func TestLoadAll_InvalidDate(t *testing.T) {
	fsys := fstest.MapFS{
		"news/bad.md": &fstest.MapFile{Data: []byte(`---
title: "Bad"
date: not-a-date
slug: bad
excerpt: "E"
---

Body.
`)},
	}

	_, err := LoadAll(fsys, "news")
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}
