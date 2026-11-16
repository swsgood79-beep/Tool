package main

import (
	"bytes"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestDataLocationVerifiedCopyAndRestartSelection(t *testing.T) {
	a := fixture(t)
	if err := atomicFile(filepath.Join(a.dir, productionMarker), []byte(productionMarkerValue)); err != nil {
		t.Fatal(err)
	}
	owner := communityOwner(t, a)
	if w := communitySave(t, a, owner, map[string]string{"csrf": a.state.Board, "author": "시험", "category": "daily", "title": "D드라이브 보관 시험", "text": "사진 포함"}, true); w.Code != 303 {
		t.Fatal(w.Code)
	}
	home := t.TempDir()
	os.MkdirAll(locationBase(home), 0700)
	target := filepath.Join(t.TempDir(), "D-data")
	before, err := storageInventory(a.dir)
	if err != nil {
		t.Fatal(err)
	}
	atomicFile(filepath.Join(locationBase(home), pendingLocationFile), jsonRaw(DataLocation{1, target, a.dir}))
	selected, err := resolveDataLocation(home)
	if err != nil || selected != target {
		t.Fatal("migration", selected, err)
	}
	after, err := storageInventory(target)
	if err != nil || !bytes.Equal(jsonRaw(before), jsonRaw(after)) {
		t.Fatal("copy mismatch", err)
	}
	originalAfter, _ := storageInventory(a.dir)
	if !bytes.Equal(jsonRaw(before), jsonRaw(originalAfter)) {
		t.Fatal("source changed")
	}
	restart, err := newApp(target)
	if err != nil || restart.state.Board != a.state.Board || len(restart.state.Community) != 1 {
		t.Fatal("new path unreadable", err)
	}
	selected, err = resolveDataLocation(home)
	if err != nil || selected != target {
		t.Fatal("selection lost")
	}
	a.storageHome = home
	c := adminCookie(a)
	if w := call(a, "GET", "/data-storage", nil, c); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if out := os.Getenv("COMMUNITY_HTML_OUT"); out != "" {
		os.WriteFile(filepath.Join(out, "storage.html"), call(a, "GET", "/data-storage", nil, c).Body.Bytes(), 0600)
	}
	if w := call(a, "POST", "/data-storage", url.Values{"csrf": {"bad"}, "directory": {filepath.Join(t.TempDir(), "new")}}, c); w.Code != 403 {
		t.Fatal("storage CSRF")
	}
}
func TestDataLocationFailureKeepsSourceAndRejectsUnsafeTargets(t *testing.T) {
	a := fixture(t)
	atomicFile(filepath.Join(a.dir, productionMarker), []byte(productionMarkerValue))
	home := t.TempDir()
	os.MkdirAll(locationBase(home), 0700)
	target := t.TempDir()
	os.WriteFile(filepath.Join(target, "company.txt"), []byte("keep"), 0600)
	atomicFile(filepath.Join(locationBase(home), pendingLocationFile), jsonRaw(DataLocation{1, target, a.dir}))
	selected, err := resolveDataLocation(home)
	if err != nil || selected != a.dir {
		t.Fatal("did not keep old source", err)
	}
	b, _ := os.ReadFile(filepath.Join(target, "company.txt"))
	if string(b) != "keep" {
		t.Fatal("destination overwritten")
	}
	for _, path := range []string{a.dir, filepath.Join(a.dir, "nested"), filepath.Dir(a.dir), "relative", filepath.Join(t.TempDir(), "WanryoTalk")} {
		if err := checkLocationPath(a.dir, path); err == nil {
			t.Fatal("unsafe target accepted", path)
		}
	}
	os.Remove(filepath.Join(locationBase(home), pendingLocationFile))
	atomicFile(filepath.Join(locationBase(home), locationFile), jsonRaw(DataLocation{Version: 1, Directory: filepath.Join(t.TempDir(), "missing")}))
	if _, err = resolveDataLocation(home); err == nil {
		t.Fatal("missing drive silently started empty")
	}
}
