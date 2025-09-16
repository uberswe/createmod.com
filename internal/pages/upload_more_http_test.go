package pages

import (
	"bytes"
	"compress/gzip"
	"createmod/internal/testutil"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
)

// helper to build multipart with raw bytes
func buildMultipartBytes(t *testing.T, field, filename string, data []byte) (string, io.Reader) {
	t.Helper()
	pr, pw := io.Pipe()
	w := multipart.NewWriter(pw)
	go func() {
		defer pw.Close()
		defer w.Close()
		fw, err := w.CreateFormFile(field, filename)
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		_, err = fw.Write(data)
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
	}()
	return w.FormDataContentType(), pr
}

func Test_Upload_HTTP_Rejects_WrongExtension(t *testing.T) {
	baseURL, cleanup := testutil.NewTestServer(t)
	defer cleanup()
	client := testutil.NewHTTPClient(t)

	ctype, body := buildMultipartBytes(t, "nbt", "notnbt.txt", []byte("hello"))
	req, _ := http.NewRequest("POST", baseURL+"/upload/nbt", body)
	req.Header.Set("Content-Type", ctype)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for wrong extension, got %d", resp.StatusCode)
	}
}

func Test_Upload_HTTP_GzipParsedSummary(t *testing.T) {
	baseURL, cleanup := testutil.NewTestServer(t)
	defer cleanup()
	client := testutil.NewHTTPClient(t)

	// prepare gzip-compressed payload
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write([]byte("hello world"))
	_ = gz.Close()

	ctype, body := buildMultipartBytes(t, "nbt", "gzip.nbt", buf.Bytes())
	req, _ := http.NewRequest("POST", baseURL+"/upload/nbt", body)
	req.Header.Set("Content-Type", ctype)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}

	// extract preview url
	s := string(b)
	// find trailing " url=/u/{token}"
	idx := bytes.LastIndex([]byte(s), []byte(" url="))
	if idx < 0 {
		t.Fatalf("missing url in response: %q", s)
	}
	preview := s[idx+5:]

	// fetch preview and assert it contains gzip summary
	resp2, err := client.Get(baseURL + preview)
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	pb, _ := io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on preview, got %d", resp2.StatusCode)
	}
	if !bytes.Contains(pb, []byte("nbt=gzip")) {
		t.Fatalf("expected preview to mention nbt=gzip, got: %s", string(pb))
	}
}
