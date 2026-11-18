package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestWorkerAppThemeBrowser(t *testing.T) {
	if os.Getenv("WORKER_THEME_BROWSER") != "1" {
		t.Skip("optional installed app theme flow")
	}
	a := fixture(t)
	a.state.Integration.Connection.Provider = "pages"
	a.state.Integration.Connection.PublicURL = "https://app-old.trycloudflare.com"
	srv := httptest.NewServer(a)
	defer srv.Close()
	cmd := exec.Command("node", "audit/browser-app-theme-v1.4.63.cjs", srv.URL, a.state.Board, "[]")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
