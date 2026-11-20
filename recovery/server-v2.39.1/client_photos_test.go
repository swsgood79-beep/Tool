package main

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

func clientPhotoFixture(t *testing.T) *App {
	t.Helper()
	a := fixture(t)
	c := ClientNote{ID: token(), Name: "MOT", Text: "업체 공통 주의사항", Revision: 1}
	for i := 0; i < 5; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 4, 4))
		img.Set(0, 0, color.RGBA{R: uint8(i * 40), G: 200, A: 255})
		var b bytes.Buffer
		if err := jpeg.Encode(&b, img, nil); err != nil {
			t.Fatal(err)
		}
		ph := Photo{ID: token(), Name: fmtID(int64(i)) + ".jpg", MIME: "image/jpeg", Legacy: true, Size: b.Len(), Width: 4, Height: 4}
		if err := a.createCanonicalPhoto(ph.ID, b.Bytes()); err != nil {
			t.Fatal(err)
		}
		c.Photos = append(c.Photos, ph)
	}
	if err := a.transaction(func(s *State) error {
		s.ClientNotes = append(s.ClientNotes, c)
		p := &s.Projects[0]
		p.ClientID = c.ID
		p.ClientPhotosSelected = true
		for _, ph := range c.Photos[:3] {
			p.ClientPhotoIDs = append(p.ClientPhotoIDs, ph.ID)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return a
}
func clientOrderUpload(t *testing.T, a *App, count int) *httptest.ResponseRecorder {
	t.Helper()
	p := a.state.Projects[0]
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range map[string]string{"csrf": "admin-csrf", "revision": fmtID(p.InfoRevision), "number": p.Number, "name": p.Name, "keep_client_link": "1", "sync_client": "1", "additional_instructions": "MOT 수주에만 적용할 메모"} {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < count; i++ {
		f, err := mw.CreateFormFile("photos", "order-only.png")
		if err != nil {
			t.Fatal(err)
		}
		img := image.NewRGBA(image.Rect(0, 0, 8, 6))
		img.Set(1, 1, color.RGBA{B: 255, A: 255})
		if err = png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
	}
	mw.Close()
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791/projects/"+fmtID(p.ID)+"/edit", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.AddCookie(adminCookie(a))
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func TestClientSelectedPhotosAppendAndReload(t *testing.T) {
	a := clientPhotoFixture(t)
	source := string(jsonRaw(a.state.ClientNotes))
	if got := len(a.projectView(a.state.Projects[0]).Photos); got != 3 {
		t.Fatal("expected three selected source photos", got)
	}
	if w := clientOrderUpload(t, a, 1); w.Code != 303 {
		t.Fatal("three selected from five plus one should save", w.Code, w.Body.String())
	}
	p := a.state.Projects[0]
	if len(p.Photos) != 1 || len(p.ClientPhotoIDs) != 3 || p.AdditionalInstructions != "MOT 수주에만 적용할 메모" {
		t.Fatal("order-only addition changed source selection", p)
	}
	if len(a.projectView(p).Photos) != 4 || string(jsonRaw(a.state.ClientNotes)) != source {
		t.Fatal("wrong displayed count or changed company source")
	}
	old := readOriginal(t, a.dir)
	if len(old.Projects[0].Photos) != 4 {
		t.Fatal("original projection lost source/added photos")
	}
	reopened, err := newApp(a.dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reopened.projectView(reopened.state.Projects[0]).Photos) != 4 || len(reopened.state.Projects[0].Photos) != 1 {
		t.Fatal("restart duplicated or lost photos")
	}
	if got := call(reopened, "GET", "/projects/1/edit", nil, adminCookie(reopened)); got.Code != 200 || !bytes.Contains(got.Body.Bytes(), []byte("이 수주에만 사진 추가")) {
		t.Fatal("edit guidance", got.Code)
	}
	before := string(jsonRaw(reopened.state.Projects))
	if w := clientOrderUpload(t, reopened, 2); w.Code != 409 {
		t.Fatal("actual combined count over five should reject", w.Code)
	}
	if string(jsonRaw(reopened.state.Projects)) != before {
		t.Fatal("failed save changed order")
	}
	if w := clientOrderUpload(t, reopened, 1); w.Code != 303 {
		t.Fatal("exactly five should save", w.Code, w.Body.String())
	}
	if len(reopened.projectView(reopened.state.Projects[0]).Photos) != 5 || string(jsonRaw(reopened.state.ClientNotes)) != source {
		t.Fatal("five photo boundary or company source")
	}
}
