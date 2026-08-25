package pages

import (
	"testing"

	"createmod/internal/store"
)

func TestApprovedTempImages(t *testing.T) {
	imgs := []store.TempUploadImage{
		{Filename: "a.webp", ModerationStatus: store.TempImageApproved},
		{Filename: "b.webp", ModerationStatus: store.TempImagePending},
		{Filename: "c.webp", ModerationStatus: store.TempImageRejected},
		{Filename: "d.webp", ModerationStatus: store.TempImageApproved},
	}
	got := approvedTempImages(imgs, nil)
	if len(got) != 2 {
		t.Fatalf("approved = %d, want 2", len(got))
	}
	for _, img := range got {
		if img.ModerationStatus != store.TempImageApproved {
			t.Errorf("included non-approved image %q (%s)", img.Filename, img.ModerationStatus)
		}
	}
	// Pending and rejected must never be shown.
	for _, img := range got {
		if img.Filename == "b.webp" || img.Filename == "c.webp" {
			t.Errorf("pending/rejected image %q leaked into display", img.Filename)
		}
	}
}
