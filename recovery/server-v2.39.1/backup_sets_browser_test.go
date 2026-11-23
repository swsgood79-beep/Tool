package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestBackupSetsBrowser(t *testing.T) {
	if os.Getenv("BACKUP_BROWSER") != "1" {
		t.Skip("optional real browser backup screen")
	}
	a, remote := setupSegmentedBackup(t)
	owner := adminCookie(a)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/audit/backup-failure" {
			remote.mu.Lock()
			remote.failPatch = r.URL.Query().Get("enable") == "1"
			remote.mu.Unlock()
			w.WriteHeader(204)
			return
		}
		a.ServeHTTP(w, r)
	}))
	defer srv.Close()
	cfg, _ := json.Marshal(map[string]any{"cookie": map[string]string{"name": owner.Name, "value": owner.Value}})
	cmd := exec.Command("node", "audit/browser-backup-sets-v1.4.88.cjs", srv.URL, string(cfg))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
