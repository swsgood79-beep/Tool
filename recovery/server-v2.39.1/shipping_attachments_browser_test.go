package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestShippingAttachmentBrowser(t *testing.T) {
	if os.Getenv("SHIPPING_ATTACHMENT_BROWSER") != "1" {
		t.Skip("optional attachment/video compression browser")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is required only to generate the optional browser fixture")
	}
	videoFixture := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "testsrc2=size=1080x1920:rate=30", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100", "-t", "5", "-c:v", "libx264", "-preset", "ultrafast", "-crf", "4", "-c:a", "aac", "-shortest", "/tmp/alpha-shipping-video.mp4")
	if data, err := videoFixture.CombinedOutput(); err != nil {
		t.Fatal(err, string(data))
	}
	a := fixture(t)
	rec := approveShipment(t, a)
	ids := []string{rec.ID}
	if err := a.commitAnnouncement(func(s *State) error {
		for i := 0; i < 3; i++ {
			r := rec
			r.ID = token()
			r.Project.Number = "MEDIA-" + fmtID(int64(i))
			s.ShippingRecords = append(s.ShippingRecords, r)
			ids = append(ids, r.ID)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, i := range []int{1, 3} {
		if w := shippingUpload(t, a, ids[i], 1, false); w.Code != 303 {
			t.Fatal(w.Code)
		}
	}
	before := []*time.Time{}
	for _, r := range a.state.ShippingRecords {
		before = append(before, r.ArchivedAt)
	}
	owner := adminCookie(a)
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg := map[string]any{"cookie": map[string]string{"name": owner.Name, "value": owner.Value}, "ids": ids, "board": a.state.Board}
	cmd := exec.Command("node", "audit/browser-shipping-attachments-v1.4.82.cjs", srv.URL, string(jsonRaw(cfg)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	media := exec.Command("python3", "audit/check-shipping-video-v1.4.82.py")
	media.Stdout, media.Stderr = os.Stdout, os.Stderr
	if err := media.Run(); err != nil {
		t.Fatal(err)
	}
	for i, r := range a.state.ShippingRecords {
		if r.ArchivedAt == nil || len(r.Photos) != 1 {
			t.Fatal("photos/archival changed")
		}
		if before[i] != nil && !before[i].Equal(*r.ArchivedAt) {
			t.Fatal("original archive date changed")
		}
		if len(r.Attachments) != 2-i%2 {
			t.Fatal("attachment count", i, len(r.Attachments))
		}
	}
}
