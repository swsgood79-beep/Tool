package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMachiningBrowser(t *testing.T) {
	if os.Getenv("MACHINING_BROWSER") != "1" {
		t.Skip("optional real Chromium flow")
	}
	a := fixture(t)
	worker, _ := machiningWorker(t, a, "machinebrowser")
	owner := adminCookie(a)
	rows := machiningSampleRows()
	rows[0][5] = "TEST-ROBOT-PR2-400(530)-20N-N-EL-V3"
	rows[0][6] = "TEST-S/P-0001-1234L-V3"
	rows[1][6] = rows[0][6]
	d := machiningDraft(t, rows)
	if e := a.importMachining(d, MachiningActor{Name: "시험 관리자"}); e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(t.TempDir(), "erp-synthetic.xlsx")
	if e := os.WriteFile(file, machiningWorkbook(t, rows), 0600); e != nil {
		t.Fatal(e)
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg := map[string]any{"worker": map[string]string{"name": worker.Name, "value": worker.Value}, "admin": map[string]string{"name": owner.Name, "value": owner.Value}, "file": file}
	cmd := exec.Command("node", "audit/browser-machining-v1.4.103.cjs", srv.URL, string(jsonRaw(cfg)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
