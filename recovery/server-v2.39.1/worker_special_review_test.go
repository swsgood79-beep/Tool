package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkerSpecialReviewRendering(t *testing.T) {
	a := fixture(t)
	p := &a.state.Projects[0]
	p.SpecialInstructions = "주의 <script>alert(1)</script>\n치수 확인"
	p.AdditionalInstructions = "추가 메모 확인"
	// Synthetic metadata; the browser test serves a synthetic photo.
	p.Photos = []Photo{{ID: "review-photo", Name: "첨부사진", Width: 20, Height: 20}}
	paths := map[string]string{"work.html": "/work/" + a.state.Board + "?entry=skip", "card.html": "/card/" + a.state.Board + "/1"}
	for name, path := range paths {
		w := call(a, "GET", path, nil)
		body := w.Body.String()
		if w.Code != 200 || !strings.Contains(body, `class="worker-special-review"`) || !strings.Contains(body, "추가 메모 확인") || !strings.Contains(body, "review-photo") || strings.Contains(body, "<script>alert(1)</script>") {
			t.Fatalf("missing or unsafe review: %s %d", name, w.Code)
		}
		if out := os.Getenv("REVIEW_HTML_OUT"); out != "" {
			if err := os.WriteFile(filepath.Join(out, name), w.Body.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	p.SpecialInstructions = ""
	p.AdditionalInstructions = ""
	p.Photos = nil
	w := call(a, "GET", paths["card.html"], nil)
	if strings.Contains(w.Body.String(), `class="worker-special-review"`) {
		t.Fatal("empty project should not ask for acknowledgement")
	}
	if out := os.Getenv("REVIEW_HTML_OUT"); out != "" {
		if err := os.WriteFile(filepath.Join(out, "empty-card.html"), w.Body.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	p.AdditionalInstructions = "追加メモ only"
	if w := call(a, "GET", paths["card.html"], nil); !strings.Contains(w.Body.String(), `class="worker-special-review"`) {
		t.Fatal("additional-only review missing")
	}
	p.AdditionalInstructions = ""
	p.Photos = []Photo{{ID: "review-photo", Name: "写真"}}
	if w := call(a, "GET", paths["card.html"], nil); !strings.Contains(w.Body.String(), `class="worker-special-review"`) {
		t.Fatal("photo-only review missing")
	}
	p.SpecialInstructions = "주의 <script>alert(1)</script>\n치수 확인"
	p.AdditionalInstructions = "추가 메모 확인"
	p.Setting.Pending = true
	p.Setting.Worker = "신청 작업자"
	p.touch()
	if out := os.Getenv("REVIEW_HTML_OUT"); out != "" {
		w := call(a, "GET", paths["card.html"], nil)
		if err := os.WriteFile(filepath.Join(out, "pending-card.html"), w.Body.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}

}
