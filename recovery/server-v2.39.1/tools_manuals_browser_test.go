package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestToolsManualsBrowser(t *testing.T) {
	if os.Getenv("TOOLS_MANUALS_BROWSER") != "1" {
		t.Skip("optional browser")
	}
	a := fixture(t)
	owner := adminCookie(a)
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg, _ := json.Marshal(map[string]any{"cookie": map[string]string{"name": owner.Name, "value": owner.Value}, "board": a.state.Board})
	cmd := exec.Command("node", "audit/browser-manual-board-v1.4.100.cjs", srv.URL, string(cfg))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
