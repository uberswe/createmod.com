package pages

import (
	"bufio"
	"createmod/internal/testutil"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

// helper to build a minimal multipart form with a single .nbt file
func buildMultipartNBTPayload(t *testing.T, field, filename string, content string) (contentType string, bodyReader io.Reader) {
	t.Helper()
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	go func() {
		defer pw.Close()
		defer writer.Close()
		fw, err := writer.CreateFormFile(field, filename)
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		bw := bufio.NewWriter(fw)
		_, err = bw.WriteString(content)
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		_ = bw.Flush()
	}()
	return writer.FormDataContentType(), pr
}

func Test_Upload_HTTP_SuccessAndPreview(t *testing.T) {
	baseURL, cleanup := testutil.NewTestServer(t)
	defer cleanup()

	client := testutil.NewHTTPClient(t)

	// Build multipart payload
	ctype, body := buildMultipartNBTPayload(t, "nbt", "example.nbt", "nbtdata")

	req, err := http.NewRequest("POST", baseURL+"/upload/nbt", body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req = testutil.WithHTMX(req)
	req.Header.Set("Content-Type", ctype)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("upload request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	b, _ := io.ReadAll(resp.Body)
	bodyStr := string(b)

	// Response should be JSON with token and url fields
	var result map[string]interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("expected JSON response, got: %q", bodyStr)
	}
	token, _ := result["token"].(string)
	url, _ := result["url"].(string)
	if token == "" || url == "" {
		t.Fatalf("expected token and url in JSON response, got: %q", bodyStr)
	}
	if !strings.HasPrefix(url, "/u/") {
		t.Fatalf("expected url to start with /u/, got: %q", url)
	}

	// GET the preview page
	req2, _ := http.NewRequest("GET", baseURL+url, nil)
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("preview request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK on preview, got %d", resp2.StatusCode)
	}
	b2, _ := io.ReadAll(resp2.Body)
	s2 := string(b2)
	if !strings.Contains(s2, "Private Preview") || !strings.Contains(s2, "example.nbt") || !strings.Contains(s2, "SHA-256:") || !strings.Contains(s2, "Parsed:") || !strings.Contains(s2, "Block Count:") || !strings.Contains(s2, "Materials:") {
		t.Fatalf("preview page missing expected content: %q", s2)
	}
}

func Test_Upload_HTTP_Duplicate(t *testing.T) {
	baseURL, cleanup := testutil.NewTestServer(t)
	defer cleanup()

	client := testutil.NewHTTPClient(t)

	// First upload
	ctype1, body1 := buildMultipartNBTPayload(t, "nbt", "dupe.nbt", "samecontent")
	req1, _ := http.NewRequest("POST", baseURL+"/upload/nbt", body1)
	req1.Header.Set("Content-Type", ctype1)
	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatalf("first upload failed: %v", err)
	}
	_ = resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first upload expected 200, got %d", resp1.StatusCode)
	}

	// Second upload with same content -> expect 409
	ctype2, body2 := buildMultipartNBTPayload(t, "nbt", "dupe.nbt", "samecontent")
	req2, _ := http.NewRequest("POST", baseURL+"/upload/nbt", body2)
	req2.Header.Set("Content-Type", ctype2)
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("second upload failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		b2, _ := io.ReadAll(resp2.Body)
		t.Fatalf("expected 409 Conflict on duplicate, got %d: %s", resp2.StatusCode, string(b2))
	}
}
