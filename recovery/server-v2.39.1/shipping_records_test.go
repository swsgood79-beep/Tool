package main

import (
	"bytes"
	"errors"
	"filippo.io/age"
	"image"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func approveShipment(t *testing.T, a *App) ShippingRecord {
	t.Helper()
	before := len(a.state.ShippingRecords)
	p := a.state.Projects[0]
	if err := a.requestInspection(p.ID, "combined", "출하 작업자", p.SettingForm, p.CombinedForm); err != nil {
		t.Fatal(err)
	}
	p = a.state.Projects[0]
	if err := a.approve(p.ID, "combined", p.Setting.EventID); err != nil {
		t.Fatal(err)
	}
	if len(a.state.ShippingRecords) != before+1 {
		t.Fatal("missing approval snapshot")
	}
	return a.state.ShippingRecords[before]
}
func shippingUpload(t *testing.T, a *App, id string, count int, badLast bool) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	m := multipart.NewWriter(&b)
	m.WriteField("csrf", a.state.Board)
	m.WriteField("memo", "출하 포장 확인 완료")
	var im bytes.Buffer
	png.Encode(&im, image.NewRGBA(image.Rect(0, 0, 8, 6)))
	for i := 0; i < count; i++ {
		f, _ := m.CreateFormFile("photos", "출하.png")
		if badLast && i == count-1 {
			f.Write([]byte("not an image"))
		} else {
			f.Write(im.Bytes())
		}
	}
	m.Close()
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791/shipping-records/"+a.state.Board+"/"+id+"/save", &b)
	r.Header.Set("Content-Type", m.FormDataContentType())
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func TestShippingSnapshotSurvivesSourceRemovalAndRestart(t *testing.T) {
	a := fixture(t)
	a.state.Designers = []Designer{{ID: "d", Name: "김재환 계장", Phone: "01012345678"}}
	a.state.ClientNotes = []ClientNote{{ID: "c", Name: "고객사", Text: "고객사 특이사항"}}
	a.state.Projects[0].DesignerID = "d"
	a.state.Projects[0].ClientID = "c"
	a.state.Projects[0].SuppliedMaterials = "사급품 수령"
	a.state.Projects[0].SuppliedMaterialsColor = "#00B050"
	a.state.MaterialReceipts = map[string][]MaterialReceipt{"T-001": {{Kind: "모터", Manufacturer: "PANASONIC", PartNumber: "MSMF", Quantity: "2"}}}
	record := approveShipment(t, a)
	if record.ArchivedAt != nil || record.Project.DesignerName != "김재환 계장" || record.Project.SpecialInstructions != "고객사 특이사항" || len(record.Receipts) != 1 {
		t.Fatal("incomplete snapshot", record)
	}
	if err := a.transaction(func(s *State) error {
		s.Projects = nil
		s.Designers = nil
		s.ClientNotes = nil
		s.MaterialReceipts = nil
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if w := call(a, "GET", "/shipping-records/"+a.state.Board+"?pending=1", nil); w.Code != 200 || !strings.Contains(w.Body.String(), record.Project.Name) {
		t.Fatal("pending lost after source removal", w.Code)
	}
	if w := shippingUpload(t, a, record.ID, 1, false); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	copy, err := newApp(a.dir)
	if err != nil {
		t.Fatal(err)
	}
	saved := copy.state.ShippingRecords[0]
	if saved.ArchivedAt == nil || len(saved.Photos) != 1 || saved.Project.DesignerName != "김재환 계장" || len(saved.Receipts) != 1 {
		t.Fatal("snapshot lost on restart")
	}
	w := call(copy, "GET", "/shipping-records/"+copy.state.Board+"?q=T-001", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "사진 1장") {
		t.Fatal("archive search", w.Code)
	}
	if w := call(copy, "GET", "/shipping-records/"+copy.state.Board+"/"+saved.ID+"/photo/"+saved.Photos[0].ID, nil); w.Code != 200 {
		t.Fatal("archived photo missing")
	}
}
func TestShippingUploadLimitsFailureAndDuplicate(t *testing.T) {
	for _, count := range []int{0, 30, 31} {
		t.Run(fmtID(int64(count)), func(t *testing.T) {
			a := fixture(t)
			record := approveShipment(t, a)
			w := shippingUpload(t, a, record.ID, count, false)
			if count == 30 {
				if w.Code != 303 || len(a.state.ShippingRecords[0].Photos) != 30 {
					t.Fatal("30 must be accepted", w.Code, w.Body.String())
				}
				w = shippingUpload(t, a, record.ID, 1, false)
				if w.Code != 303 || len(a.state.ShippingRecords[0].Photos) != 30 {
					t.Fatal("duplicate replaced archive")
				}
			} else if w.Code != 400 || a.state.ShippingRecords[0].ArchivedAt != nil {
				t.Fatal("invalid count accepted", w.Code)
			}
		})
	}
	a := fixture(t)
	record := approveShipment(t, a)
	if w := shippingUpload(t, a, record.ID, 2, true); w.Code != 400 {
		t.Fatal("bad image accepted")
	}
	files, _ := os.ReadDir(filepath.Join(a.dir, "photos"))
	if len(files) != 0 {
		t.Fatal("failed batch left orphan files")
	}
}
func TestShippingPermissionsAndAdminDeletion(t *testing.T) {
	a := fixture(t)
	record := approveShipment(t, a)
	base := "/shipping-records/" + a.state.Board + "/" + record.ID
	for _, path := range []string{"/shipping-records/wrong", base + "/delete"} {
		w := call(a, "POST", path, url.Values{"csrf": {a.state.Board}, "confirm": {"1"}})
		if w.Code == 200 || w.Code == 303 {
			t.Fatal("unauthorized mutation", path)
		}
	}
	w := call(a, "POST", base+"/save", url.Values{"csrf": {"wrong"}})
	if w.Code != 403 {
		t.Fatal("missing csrf guard")
	}
	if w = shippingUpload(t, a, record.ID, 1, false); w.Code != 303 {
		t.Fatal(w.Code)
	}
	photo := a.state.ShippingRecords[0].Photos[0]
	cookie := adminCookie(a)
	csrf := a.sessions[hashToken(cookie.Value)].CSRF
	w = call(a, "POST", "/shipping-archive-admin/"+record.ID+"/delete", url.Values{"csrf": {csrf}, "confirm": {"1"}}, cookie)
	if w.Code != 303 || len(a.state.ShippingRecords) != 1 || a.state.ShippingRecords[0].ArchivedAt != nil || len(a.state.ShippingRecords[0].Photos) != 0 || len(a.state.Projects) != 1 {
		t.Fatal("delete scope", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filepath.Join(a.dir, "photos", photo.ID)); !os.IsNotExist(err) {
		t.Fatal("photo orphan after archive deletion")
	}
}
func TestShippingNoHistoricalOrRejectedMigration(t *testing.T) {
	a := fixture(t)
	now := time.Now()
	a.state.Projects[0].Shipping.Completed = &now
	if w := call(a, "GET", "/work/"+a.state.Board+"?entry=skip", nil); w.Code != 200 || !strings.Contains(w.Body.String(), "출하사진 필요") {
		t.Fatal(w.Code)
	}
	if len(a.state.ShippingRecords) != 0 {
		t.Fatal("historical completed order migrated unexpectedly")
	}
}

func TestShippingSpecialPhotosIndependentAndPortable(t *testing.T) {
	a := fixture(t)
	var raw bytes.Buffer
	png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 5, 7)))
	id := token()
	protected, _ := protect(raw.Bytes())
	os.MkdirAll(filepath.Join(a.dir, "photos"), 0700)
	if err := atomicFile(filepath.Join(a.dir, "photos", id), protected); err != nil {
		t.Fatal(err)
	}
	if err := a.transaction(func(s *State) error {
		s.Projects[0].Photos = []Photo{{ID: id, Name: "special.png", MIME: "image/png", Width: 5, Height: 7, Size: raw.Len()}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	record := approveShipment(t, a)
	if len(record.Project.Photos) != 1 || record.Project.Photos[0].ID == id || record.Project.Photos[0].Legacy {
		t.Fatal("special photos not copied independently")
	}
	if err := a.transaction(func(s *State) error { s.Projects = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(a.dir, "photos", id))
	os.Remove(filepath.Join(a.dir, "project-photos", id+".dat"))
	if w := shippingUpload(t, a, record.ID, 1, false); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	identity, _ := age.GenerateX25519Identity()
	key := RecoveryKey{1, repository, token(), identity.String()}
	a.state.Integration.Backup.Key = key
	encrypted, err := a.portable(a.state)
	if err != nil {
		t.Fatal("archive backup", err)
	}
	fresh, err := newApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = fresh.restorePortable(key, encrypted); err != nil {
		t.Fatal("archive restore", err)
	}
	restored := fresh.state.ShippingRecords[0]
	for _, ph := range append(restored.Project.Photos, restored.Photos...) {
		w := call(fresh, "GET", "/shipping-records/"+fresh.state.Board+"/"+record.ID+"/photo/"+ph.ID, nil)
		if w.Code != 200 {
			t.Fatal("restored snapshot photo unreadable", w.Code)
		}
	}
	// Existing Talk attachment pruning must respect archive-owned image references.
	fresh.pruneCommunityPhotos(restored.Photos)
	if _, err = os.Stat(filepath.Join(fresh.dir, "photos", restored.Photos[0].ID)); err != nil {
		t.Fatal("archive photo pruned by community", err)
	}
}

func TestShippingApprovalStagesAndSaveRollback(t *testing.T) {
	a := fixture(t)
	p := a.state.Projects[0]
	if err := a.requestInspection(p.ID, "setting", "세팅 작업자", p.SettingForm, ""); err != nil {
		t.Fatal(err)
	}
	if err := a.rejectInspection(p.ID, "setting", a.state.Projects[0].Setting.EventID); err != nil {
		t.Fatal(err)
	}
	if len(a.state.ShippingRecords) != 0 {
		t.Fatal("rejected inspection created photo task")
	}
	p = a.state.Projects[0]
	if err := a.requestInspection(p.ID, "setting", "세팅 작업자", p.SettingForm, ""); err != nil {
		t.Fatal(err)
	}
	if err := a.approve(p.ID, "setting", a.state.Projects[0].Setting.EventID); err != nil {
		t.Fatal(err)
	}
	if len(a.state.ShippingRecords) != 0 {
		t.Fatal("setting-only approval created photo task")
	}
	p = a.state.Projects[0]
	if err := a.requestInspection(p.ID, "shipping", "출하 작업자", p.ShippingForm, ""); err != nil {
		t.Fatal(err)
	}
	if err := a.approve(p.ID, "shipping", a.state.Projects[0].Shipping.EventID); err != nil {
		t.Fatal(err)
	}
	record := a.state.ShippingRecords[0]
	if record.Project.Setting.Worker != "세팅 작업자" || record.Project.Shipping.Worker != "출하 작업자" {
		t.Fatal("worker snapshot mismatch")
	}
	a.directHook = func(phase, name string) error {
		if phase == "written" {
			return errors.New("simulated storage failure")
		}
		return nil
	}
	if w := shippingUpload(t, a, record.ID, 1, false); w.Code != 400 || a.state.ShippingRecords[0].ArchivedAt != nil {
		t.Fatal("failed save completed record")
	}
	files, _ := os.ReadDir(filepath.Join(a.dir, "photos"))
	if len(files) != 0 {
		t.Fatal("failed commit orphaned upload")
	}
	a.directHook = nil
	if w := shippingUpload(t, a, record.ID, 1, false); w.Code != 303 {
		t.Fatal("retry did not recover", w.Code, w.Body.String())
	}
}
