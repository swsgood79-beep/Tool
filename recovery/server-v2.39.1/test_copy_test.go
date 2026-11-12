package main

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestExistingDataCopyIsIndependentAndRestoresPhotos(t *testing.T) {
	source := fixture(t)
	p := source.state.Projects[0]
	if e := source.requestInspection(p.ID, "combined", "기존 작업자", p.SettingForm, p.CombinedForm); e != nil {
		t.Fatal(e)
	}
	var img bytes.Buffer
	jpeg.Encode(&img, image.NewRGBA(image.Rect(0, 0, 2, 3)), nil)
	id := token()
	if e := source.createCanonicalPhoto(id, img.Bytes()); e != nil {
		t.Fatal(e)
	}
	if e := source.transaction(func(s *State) error {
		s.Projects[0].Photos = []Photo{{ID: id, Name: "existing.jpg", MIME: "image/jpeg", Legacy: true, Width: 2, Height: 3, Size: img.Len()}}
		s.Designers = []Designer{{ID: token(), Name: "기존 담당자", Phone: "01000000000", Revision: 1}}
		s.Integration.Connection.GitHubToken = "do-not-send"
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	before, e := testCopyInventory(source.dir)
	if e != nil {
		t.Fatal(e)
	}
	base := t.TempDir()
	target, e := copyForTest(source.dir, base, nil)
	if e != nil {
		t.Fatal(e)
	}
	app, e := newTestApp(target)
	if e != nil {
		t.Fatal(e)
	}
	if len(app.state.Projects) != 1 || len(app.state.Designers) != 1 || len(app.state.Events) != 2 || app.state.Integration.Connection.GitHubToken != "" {
		t.Fatal("copy did not load or credentials retained")
	}
	data, e := app.legacyPhoto(app.state.Projects[0].Photos[0])
	if e != nil || !bytes.Equal(data, img.Bytes()) {
		t.Fatal("copied photo mismatch", e)
	}
	if e = app.approve(p.ID, "combined", app.state.Projects[0].Setting.EventID); e != nil {
		t.Fatal(e)
	}
	if source.state.Projects[0].Shipping.Completed != nil {
		t.Fatal("source state changed")
	}
	after, e := testCopyInventory(source.dir)
	if e != nil {
		t.Fatal(e)
	}
	for n, h := range before {
		if after[n] != h {
			t.Fatal("original source changed", n)
		}
	}
	if e = selectTestCopy(base, target); e != nil {
		t.Fatal(e)
	}
	selected, e := selectedTestDir(base)
	if e != nil || selected != target {
		t.Fatal("copy selection missing", e)
	}
	again, e := newTestApp(selected)
	if e != nil || again.state.Projects[0].Shipping.Completed == nil {
		t.Fatal("copy changes lost on restart", e)
	}
}
func TestExistingCopyDetectsChangingSourceAndKeepsSelection(t *testing.T) {
	source := fixture(t)
	base := t.TempDir()
	first, e := copyForTest(source.dir, base, nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = selectTestCopy(base, first); e != nil {
		t.Fatal(e)
	}
	_, e = copyForTest(source.dir, base, func() error {
		return os.WriteFile(filepath.Join(source.dir, "projects-v2.dat"), []byte("changed externally"), 0600)
	})
	if e == nil {
		t.Fatal("changing source accepted")
	}
	selected, e := selectedTestDir(base)
	if e != nil || selected != first {
		t.Fatal("previous selection changed", e)
	}
}
func TestExistingCopyMissingSourceAndPathEscape(t *testing.T) {
	base := t.TempDir()
	if _, e := copyForTest(t.TempDir(), base, nil); e == nil {
		t.Fatal("missing primary accepted")
	}
	if _, e := os.Stat(filepath.Join(base, testMarker)); !os.IsNotExist(e) {
		t.Fatal("created test target without valid source")
	}
	if _, e := prepareTestDir(base); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(base, activeCopyFile), []byte(`{"Version":1,"Folder":"../../WanryoTalk"}`), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := selectedTestDir(base); e == nil {
		t.Fatal("escaped copy folder accepted")
	}
}
func TestExistingCopyPreservesOldSnapshotsAndFailedImport(t *testing.T) {
	source := fixture(t)
	base := t.TempDir()
	one, e := copyForTest(source.dir, base, nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = selectTestCopy(base, one); e != nil {
		t.Fatal(e)
	}
	two, e := copyForTest(source.dir, base, nil)
	if e != nil || one == two {
		t.Fatal("copy replaced previous folder", e)
	}
	if _, e = os.Stat(filepath.Join(one, "projects-v2.dat")); e != nil {
		t.Fatal(e)
	}
	if _, e = copyForTest(source.dir, base, func() error { return errors.New("simulated copy failure") }); e == nil {
		t.Fatal("failure ignored")
	}
	if selected, e := selectedTestDir(base); e != nil || selected != one {
		t.Fatal("failure changed active copy", e)
	}
}
