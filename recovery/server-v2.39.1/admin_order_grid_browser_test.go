package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestAdminOrderGridBrowser(t *testing.T) {
	if os.Getenv("ADMIN_GRID_BROWSER") != "1" {
		t.Skip("optional browser layout check")
	}
	a := fixture(t)
	for i := 0; i < 3; i++ {
		p := Project{ID: a.state.next(), Number: "GRID-" + fmtID(int64(i)), Name: "두열 시험 " + fmtID(int64(i)), Revision: 1, InfoRevision: 1, ShippingDueDate: day(time.Now())}
		p.rotate()
		a.state.Projects = append(a.state.Projects, p)
	}
	cookie := adminCookie(a)
	srv := httptest.NewServer(a)
	defer srv.Close()
	cmd := exec.Command("node", "audit/browser-admin-grid-v1.4.59.cjs", srv.URL, string(jsonRaw(map[string]string{"name": cookie.Name, "value": cookie.Value})))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}
