package main

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTalkStatusBoardGuardAndForumNavigation(t *testing.T) {
	a := fixture(t)
	author := communityOwner(t, a)
	viewer := communityOwner(t, a)
	base := "/community/" + a.state.Board
	for i := 0; i < 3; i++ {
		category := "work"
		if i == 2 {
			category = "daily"
		}
		if w := communitySave(t, a, author, map[string]string{"csrf": a.state.Board, "author": "김동홍", "category": category, "title": "게시판 글 " + fmtID(int64(i+1)), "text": "작업 방법과 현장 이야기를 나눠요."}, i == 0); w.Code != 303 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	var status struct {
		Posts []struct {
			ID   string `json:"id"`
			Mine bool   `json:"mine"`
		} `json:"posts"`
	}
	w := call(a, "GET", base+"/status", nil, viewer)
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &status) != nil || len(status.Posts) != 3 {
		t.Fatal("status", w.Code, w.Body.String())
	}
	for _, p := range status.Posts {
		if p.Mine {
			t.Fatal("viewer owns other browser post")
		}
	}
	if strings.Contains(w.Body.String(), "Owner") || strings.Contains(w.Body.String(), "owner") {
		t.Fatal("owner secret exposed")
	}
	own := call(a, "GET", base+"/status", nil, author)
	if json.Unmarshal(own.Body.Bytes(), &status) != nil {
		t.Fatal("status JSON")
	}
	for _, p := range status.Posts {
		if !p.Mine {
			t.Fatal("own posts not identified")
		}
	}
	if w := call(a, "GET", "/community/"+token()+"/status", nil, viewer); w.Code != 404 {
		t.Fatal("wrong board status")
	}
	p1, p2 := a.state.Community[0], a.state.Community[1]
	detail := call(a, "GET", base+"/"+p1.ID+"?category=work&q=게시판&page=2", nil, viewer).Body.String()
	if !strings.Contains(detail, `data-talk-read="`+p1.ID+`"`) || !strings.Contains(detail, p2.ID+"?category=work") || !strings.Contains(detail, "목록으로") || !strings.Contains(detail, "page=2") {
		t.Fatal("detail/read navigation context")
	}
	if strings.Contains(detail, a.state.Community[2].ID+"?category=work") {
		t.Fatal("next post escaped category")
	}
	list := call(a, "GET", base, nil, viewer).Body.String()
	if !strings.Contains(list, "talk-row") || !strings.Contains(list, "talk-title") || strings.Contains(list, "community-excerpt") {
		t.Fatal("forum list not applied")
	}
	if !strings.Contains(list, "talk-unread-v1.4.39.js") {
		t.Fatal("unread script missing")
	}
	if w := call(a, "POST", base+"/delete", url.Values{"csrf": {a.state.Board}, "id": {p1.ID}}, author); w.Code != 303 {
		t.Fatal(w.Code)
	}
	w = call(a, "GET", base+"/status", nil, viewer)
	json.Unmarshal(w.Body.Bytes(), &status)
	if len(status.Posts) != 2 {
		t.Fatal("deleted post remains in unread status")
	}
}
func TestTalkForumAndUnreadFixture(t *testing.T) {
	a := fixture(t)
	author := communityOwner(t, a)
	viewer := communityOwner(t, a)
	admin := adminCookie(a)
	base := "/community/" + a.state.Board
	for i, cat := range []string{"work", "work", "suggestion", "daily"} {
		if w := communitySave(t, a, author, map[string]string{"csrf": a.state.Board, "author": "김동홍", "category": cat, "title": []string{"보호커버 체결 순서 공유합니다", "케이블 정리할 때 참고하세요", "공구 보관 위치를 정하면 어떨까요", "오늘 점심 맛있게 드셨나요"}[i], "text": "브라켓 위치를 먼저 맞춘 뒤 체결하면 수월해요.\n사진의 표시 위치를 확인해 주세요."}, i == 0); w.Code != 303 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	id := a.state.Community[0].ID
	out := os.Getenv("TALK_HTML_OUT")
	if out == "" {
		return
	}
	routes := map[string]string{"feed.html": base, "work-category.html": base + "?category=work", "suggestion-category.html": base + "?category=suggestion", "daily-category.html": base + "?category=daily", "new.html": base + "/new", "detail.html": base + "/" + id + "?category=work", "all-detail.html": base + "/" + id + "?category=all", "work.html": "/work/" + a.state.Board + "?entry=skip", "home.html": "/", "admin.html": "/community-admin", "status.json": base + "/status"}
	for name, route := range routes {
		w := call(a, "GET", route, nil, viewer, admin)
		if w.Code != 200 {
			t.Fatal(route, w.Code, w.Body.String())
		}
		os.WriteFile(filepath.Join(out, name), w.Body.Bytes(), 0600)
	}
	for _, post := range a.state.Community {
		for _, category := range []string{"all", post.Category} {
			w := call(a, "GET", base+"/"+post.ID+"?category="+category, nil, viewer, admin)
			if w.Code != 200 {
				t.Fatal(w.Code)
			}
			os.WriteFile(filepath.Join(out, "detail-"+category+"-"+post.ID+".html"), w.Body.Bytes(), 0600)
		}
	}
	photo := a.state.Community[0].Photos[0]
	raw, _ := os.ReadFile(filepath.Join(a.dir, "photos", photo.ID))
	raw, _ = unprotect(raw)
	os.WriteFile(filepath.Join(out, "photo.jpg"), raw, 0600)
	ids := []string{}
	for _, p := range a.state.Community {
		ids = append(ids, p.ID)
	}
	os.WriteFile(filepath.Join(out, "fixture.json"), jsonRaw(map[string]any{"board": a.state.Board, "id": id, "ids": ids}), 0600)
}
