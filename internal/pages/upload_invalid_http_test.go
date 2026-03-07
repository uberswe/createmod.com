package pages

import (
	"createmod/internal/testutil"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
)

// helper: raw multipart with bytes
func buildMultipartRaw(t *testing.T, field, filename string, data []byte) (string, io.Reader) {
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
		if _, err = fw.Write(data); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
	}()
	return w.FormDataContentType(), pr
}

func Test_Upload_HTTP_InvalidGzipRejected(t *testing.T) {
	baseURL, cleanup := testutil.NewTestServer(t)
	defer cleanup()

	client := testutil.NewHTTPClient(t)

	// Construct bytes that start with gzip magic but are not a valid gzip stream
	invalid := []byte{0x1f, 0x8b, 0x08, 0x00, 0x01, 0x02, 0x03}

	ctype, body := buildMultipartRaw(t, "nbt", "broken.nbt", invalid)
	req, _ := http.NewRequest("POST", baseURL+"/upload/nbt", body)
	req.Header.Set("Content-Type", ctype)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid gzip, got %d", resp.StatusCode)
	}
}
