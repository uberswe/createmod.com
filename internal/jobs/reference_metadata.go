package jobs

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/riverqueue/river"
	"golang.org/x/net/html"
)

type ReferenceMetadataArgs struct{}

func (ReferenceMetadataArgs) Kind() string { return "reference_metadata_fetch" }

type ReferenceMetadataWorker struct {
	river.WorkerDefaults[ReferenceMetadataArgs]
	deps Deps
}

func (w *ReferenceMetadataWorker) Work(ctx context.Context, job *river.Job[ReferenceMetadataArgs]) error {
	slog.Info("reference metadata fetch started")
	if w.deps.Store == nil {
		return nil
	}

	stale, err := w.deps.Store.References.ListStale(ctx, 50)
	if err != nil {
		return err
	}

	if len(stale) == 0 {
		slog.Info("reference metadata: no stale references")
		return nil
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	updated := 0

	for _, ref := range stale {
		req, err := http.NewRequestWithContext(ctx, "GET", ref.URL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "createmod.com/1.0 (metadata fetcher)")

		resp, err := client.Do(req)
		if err != nil {
			slog.Warn("reference metadata: fetch failed", "id", ref.ID, "err", err)
			continue
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()

		if resp.StatusCode != 200 {
			continue
		}

		title, image, author := parseOGMeta(string(body))
		if err := w.deps.Store.References.UpdateMetadata(ctx, ref.ID, title, image, author); err != nil {
			slog.Warn("reference metadata: update failed", "id", ref.ID, "err", err)
		} else {
			updated++
		}
	}

	slog.Info("reference metadata fetch completed", "checked", len(stale), "updated", updated)
	return nil
}

func parseOGMeta(htmlContent string) (title, image, author string) {
	tokenizer := html.NewTokenizer(strings.NewReader(htmlContent))
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}
		t := tokenizer.Token()
		if t.Data != "meta" && t.Data != "title" {
			continue
		}
		if t.Data == "title" && title == "" {
			tokenizer.Next()
			title = strings.TrimSpace(tokenizer.Token().Data)
			continue
		}

		var property, name, content string
		for _, a := range t.Attr {
			switch a.Key {
			case "property":
				property = a.Val
			case "name":
				name = a.Val
			case "content":
				content = a.Val
			}
		}

		switch {
		case property == "og:title" && content != "":
			title = content
		case property == "og:image" && content != "":
			image = content
		case (property == "article:author" || name == "author") && content != "" && author == "":
			author = content
		}
	}
	return title, image, author
}
