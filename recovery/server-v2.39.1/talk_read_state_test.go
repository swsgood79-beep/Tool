package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func readFixture(t *testing.T) *App {
	a := fixture(t)
	for i := 0; i < 3; i++ {
		a.state.Community = append(a.state.Community, CommunityPost{ID: token(), Title: "읽음 시험", Text: "본문", Author: "다른 작업자", Owner: hashToken(token()), Category: "work", Revision: 1, CreatedAt: time.Now()})
	}
	return a
}
func readCall(a *App, reader, source string, ids []string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(talkReadUpdate{a.state.Board, reader, source, ids})
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791/talk-read-state/"+a.state.Board, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func readIDs(t *testing.T, w *httptest.ResponseRecorder) []string {
	t.Helper()
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var out struct {
		IDs []string `json:"read_ids"`
	}
	if json.Unmarshal(w.Body.Bytes(), &out) != nil {
		t.Fatal("bad response")
	}
	return out.IDs
}
func TestTalkReadStateMergeIsolationPersistenceAndNoWorkbookRewrite(t *testing.T) {
	a := readFixture(t)
	reader, other, legacy := token(), token(), token()
	id1, id2 := a.state.Community[0].ID, a.state.Community[1].ID
	report := filepath.Join(a.dir, "검사완료기록-복구서버.xlsx")
	before, _ := os.ReadFile(report)
	if ids := readIDs(t, readCall(a, reader, "", []string{id1, token()})); len(ids) != 1 || ids[0] != id1 {
		t.Fatal(ids)
	}
	if ids := readIDs(t, readCall(a, other, "", nil)); len(ids) != 0 {
		t.Fatal("other device saw read marks")
	}
	readIDs(t, readCall(a, legacy, "", []string{id2}))
	if ids := readIDs(t, readCall(a, reader, legacy, []string{id1})); len(ids) != 2 {
		t.Fatal("migration replaced reads", ids)
	}
	again, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	if ids := readIDs(t, readCall(again, reader, "", nil)); len(ids) != 2 {
		t.Fatal("restart lost reads", ids)
	}
	after, _ := os.ReadFile(report)
	if !bytes.Equal(before, after) {
		t.Fatal("read marks rewrote inspection workbook")
	}
	rawBefore, _ := os.ReadFile(filepath.Join(a.dir, "projects-v2.dat"))
	readIDs(t, readCall(a, reader, "", nil))
	rawAfter, _ := os.ReadFile(filepath.Join(a.dir, "projects-v2.dat"))
	if !bytes.Equal(rawBefore, rawAfter) {
		t.Fatal("unchanged polling rewrote data")
	}
	a.state.Community = a.state.Community[1:]
	if ids := readIDs(t, readCall(a, reader, "", nil)); len(ids) != 1 || ids[0] != id2 {
		t.Fatal("deleted post mark not removed", ids)
	}
	if w := readCall(a, "invalid", "", []string{id1}); w.Code != 400 {
		t.Fatal("invalid reader accepted")
	}
}
