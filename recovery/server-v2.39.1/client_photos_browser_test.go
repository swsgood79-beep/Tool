package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestClientSelectedPhotosBrowser(t *testing.T) {
	if os.Getenv("CLIENT_PHOTOS_BROWSER") != "1" {
		t.Skip("optional order-specific client photo browser")
	}
	a := clientPhotoFixture(t)
	ids := []int64{a.state.Projects[0].ID}
	if err := a.transaction(func(s *State) error {
		for i := 0; i < 3; i++ {
			p := s.Projects[0]
			p.ID = s.next()
			p.Number = "MOT-" + fmtID(int64(i))
			p.rotate()
			s.Projects = append(s.Projects, p)
			ids = append(ids, p.ID)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cookie := adminCookie(a)
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg := map[string]any{"cookie": map[string]string{"name": cookie.Name, "value": cookie.Value}, "ids": ids, "board": a.state.Board}
	cmd := exec.Command("node", "audit/browser-client-photos-v1.4.81.cjs", srv.URL, string(jsonRaw(cfg)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	for _, p := range a.state.Projects {
		if len(a.projectView(p).Photos) != 4 || len(p.Photos) != 1 || len(p.ClientPhotoIDs) != 3 {
			t.Fatal("browser did not preserve source selection and add exactly one photo")
		}
	}
	if len(a.state.ClientNotes[0].Photos) != 5 {
		t.Fatal("browser changed original source")
	}
}
