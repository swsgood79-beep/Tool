package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestCalendarNavigationBrowser(t *testing.T) {
	if os.Getenv("CALENDAR_NAV_BROWSER") != "1" {
		t.Skip("optional real browser navigation")
	}
	a := fixture(t)
	srv := httptest.NewServer(a)
	defer srv.Close()
	cmd := exec.Command("node", "audit/browser-calendar-navigation-v1.4.105.cjs", srv.URL, a.state.Board)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}
