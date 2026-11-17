package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type CommunityComment struct {
	ID, Author, Text, Owner string
	CreatedAt               time.Time
}
type CommunityPost struct {
	ID, Author, Category, Title, Text, Owner string
	Revision                                 int64
	CreatedAt, UpdatedAt                     time.Time
	Photos                                   []Photo
	Likes                                    []string
	Comments                                 []CommunityComment
}
type CommunityView struct {
	CommunityPost
	CanEdit, Liked bool
	CategoryLabel  string
}

func communityCategory(s string) string {
	return map[string]string{"work": "작업안내", "suggestion": "건의사항", "daily": "일상생활"}[s]
}
func findCommunity(s *State, id string) *CommunityPost {
	for i := range s.Community {
		if s.Community[i].ID == id {
			return &s.Community[i]
		}
	}
	return nil
}
func communityIdentity(w http.ResponseWriter, r *http.Request, create bool) string {
	c, e := r.Cookie("alpha-community-owner-v1")
	v := ""
	if e == nil && validPhotoID.MatchString(c.Value) {
		v = c.Value
	}
	if v == "" && create {
		v = token()
		http.SetCookie(w, &http.Cookie{Name: "alpha-community-owner-v1", Value: v, Path: "/", HttpOnly: true, Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https", SameSite: http.SameSiteStrictMode, MaxAge: 31536000})
	}
	if v == "" {
		return ""
	}
	return hashToken(v)
}
func (a *App) communityHTTP(w http.ResponseWriter, r *http.Request) bool {
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	admin := len(p) > 0 && p[0] == "community-admin"
	if len(p) == 0 || p[0] != "community" && !admin {
		return false
	}
	var session Session
	if admin {
		var ok bool
		session, ok = a.admin(w, r)
		if !ok {
			return true
		}
	} else if len(p) < 2 || !equal(p[1], a.state.Board) {
		http.NotFound(w, r)
		return true
	}
	w.Header().Set("Cache-Control", "no-store")
	owner := communityIdentity(w, r, r.Method == "GET")
	if r.Method == "POST" {
		if !admin && (owner == "" || !equal(r.FormValue("csrf"), a.state.Board)) {
			http.Error(w, "작업자 요청을 확인할 수 없습니다. 화면을 새로고침해 주세요.", 403)
			return true
		}
		if !admin && len(p) != 3 || admin && len(p) != 2 {
			http.NotFound(w, r)
			return true
		}
		action := p[len(p)-1]
		var removed []Photo
		if action == "delete" {
			if old := findCommunity(&a.state, r.FormValue("id")); old != nil {
				removed = append(removed, old.Photos...)
			}
		}
		var err error
		switch action {
		case "save":
			if admin {
				http.NotFound(w, r)
				return true
			}
			err = a.saveCommunity(r, owner)
		case "like", "comment", "delete", "delete-comment":
			err = a.commitAnnouncement(func(s *State) error {
				n := findCommunity(s, r.FormValue("id"))
				if n == nil {
					return errors.New("글을 찾을 수 없습니다.")
				}
				switch action {
				case "delete":
					if !admin && n.Owner != owner {
						return errors.New("이 브라우저에서 작성한 글만 삭제할 수 있습니다.")
					}
					for i := range s.Community {
						if s.Community[i].ID == n.ID {
							s.Community = append(s.Community[:i], s.Community[i+1:]...)
							break
						}
					}
				case "like":
					if admin {
						return errors.New("작업자 화면에서 이용해 주세요.")
					}
					found := false
					for i, v := range n.Likes {
						if v == owner {
							n.Likes = append(n.Likes[:i], n.Likes[i+1:]...)
							found = true
							break
						}
					}
					if !found {
						n.Likes = append(n.Likes, owner)
					}
					n.Revision++
				case "comment":
					if admin {
						return errors.New("작업자 화면에서 이용해 주세요.")
					}
					author, text := strings.TrimSpace(r.FormValue("author")), strings.TrimSpace(r.FormValue("text"))
					if author == "" || text == "" || utf8.RuneCountInString(author) > 30 || utf8.RuneCountInString(text) > 1000 {
						return errors.New("작성자(30자 이내)와 댓글(1,000자 이내)을 입력해 주세요.")
					}
					if len(n.Comments) >= 1000 {
						return errors.New("댓글은 글당 최대 1,000개입니다.")
					}
					n.Comments = append(n.Comments, CommunityComment{token(), author, text, owner, time.Now()})
					n.Revision++
				case "delete-comment":
					found := false
					for i, c := range n.Comments {
						if c.ID == r.FormValue("comment") {
							if !admin && c.Owner != owner {
								return errors.New("이 브라우저에서 작성한 댓글만 삭제할 수 있습니다.")
							}
							n.Comments = append(n.Comments[:i], n.Comments[i+1:]...)
							found = true
							break
						}
					}
					if !found {
						return errors.New("댓글을 찾을 수 없습니다.")
					}
					n.Revision++
				}
				return nil
			})
		default:
			http.NotFound(w, r)
			return true
		}
		if err != nil {
			http.Error(w, err.Error(), 400)
			return true
		}
		a.pruneCommunityPhotos(removed)
		next := "/community/" + a.state.Board
		if admin {
			next = "/community-admin"
		} else if action == "comment" || action == "like" || action == "delete-comment" {
			next += "/" + r.FormValue("id")
		}
		http.Redirect(w, r, next, 303)
		return true
	}
	if r.Method != "GET" {
		http.Error(w, "지원하지 않는 요청입니다.", 405)
		return true
	}
	v := map[string]any{"Board": a.state.Board, "WorkerURL": "/work/" + a.state.Board, "CommunityAdmin": admin, "CSRF": a.state.Board}
	if admin {
		v["CSRF"] = session.CSRF
	}
	v["Base"] = "/community/" + a.state.Board
	if admin {
		v["Base"] = "/community-admin"
	}
	offset := 2
	if admin {
		offset = 1
	}
	tail := p[offset:]
	if !admin && len(tail) == 1 && tail[0] == "status" {
		posts := []map[string]any{}
		for _, post := range a.state.Community {
			posts = append(posts, map[string]any{"id": post.ID, "mine": post.Owner == owner})
		}
		jsonReply(w, map[string]any{"posts": posts})
		return true
	}
	var n *CommunityPost
	if len(tail) > 0 && tail[0] != "new" {
		n = findCommunity(&a.state, tail[0])
		if n == nil {
			http.NotFound(w, r)
			return true
		}
	}
	if len(tail) == 3 && tail[1] == "photos" {
		a.photo(w, r, n.Photos, tail[2])
		return true
	}
	if len(tail) == 1 && tail[0] == "new" || len(tail) == 2 && tail[1] == "edit" {
		if admin {
			http.NotFound(w, r)
			return true
		}
		q := CommunityPost{Category: "daily"}
		if n != nil {
			if n.Owner != owner {
				http.Error(w, "이 브라우저에서 작성한 글만 수정할 수 있습니다.", 403)
				return true
			}
			q = *n
		}
		v["Post"] = q
		a.render(w, "community-edit.html", v)
		return true
	}
	if len(tail) > 1 {
		http.NotFound(w, r)
		return true
	}
	if n != nil {
		q := CommunityView{CommunityPost: *n, CanEdit: admin || n.Owner == owner, CategoryLabel: communityCategory(n.Category)}
		for _, like := range n.Likes {
			if like == owner {
				q.Liked = true
			}
		}
		v["Post"] = q
		v["Owner"] = owner
		category := r.URL.Query().Get("category")
		if category == "" {
			category = n.Category
		}
		if category != "all" && communityCategory(category) == "" {
			category = n.Category
		}
		v["Category"] = category
		listCategory := category
		if listCategory == "all" {
			listCategory = ""
		}
		v["ListCategory"] = listCategory
		v["Query"] = r.URL.Query().Get("q")
		listPage := number(r.URL.Query().Get("page"))
		if listPage < 1 {
			listPage = 1
		}
		v["ListPage"] = listPage
		query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
		navigation := []CommunityPost{}
		for _, post := range a.state.Community {
			if (category == "all" || post.Category == category) && (query == "" || strings.Contains(strings.ToLower(post.Author+" "+post.Title+" "+post.Text), query)) {
				navigation = append(navigation, post)
			}
		}
		sort.SliceStable(navigation, func(i, j int) bool { return navigation[i].CreatedAt.After(navigation[j].CreatedAt) })
		for i, post := range navigation {
			if post.ID == n.ID {
				if i > 0 {
					v["PreviousPost"] = navigation[i-1].ID
				}
				if i+1 < len(navigation) {
					v["NextPost"] = navigation[i+1].ID
				}
				break
			}
		}
		a.render(w, "community-detail.html", v)
		return true
	}
	posts := []CommunityView{}
	category := r.URL.Query().Get("category")
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	for _, n := range a.state.Community {
		if category != "" && n.Category != category {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(n.Author+" "+n.Title+" "+n.Text), query) {
			continue
		}
		q := CommunityView{CommunityPost: n, CanEdit: admin || n.Owner == owner, CategoryLabel: communityCategory(n.Category)}
		for _, like := range n.Likes {
			if like == owner {
				q.Liked = true
			}
		}
		posts = append(posts, q)
	}
	sort.SliceStable(posts, func(i, j int) bool { return posts[i].CreatedAt.After(posts[j].CreatedAt) })
	page := number(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pages := (len(posts) + 19) / 20
	if pages < 1 {
		pages = 1
	}
	if page > int64(pages) {
		page = int64(pages)
	}
	start := int(page-1) * 20
	end := start + 20
	if end > len(posts) {
		end = len(posts)
	}
	v["Posts"] = posts[start:end]
	v["Category"] = category
	v["Query"] = r.URL.Query().Get("q")
	v["Page"], v["Pages"], v["Previous"], v["Next"] = page, pages, page-1, page+1
	a.render(w, "community.html", v)
	return true
}
func (a *App) saveCommunity(r *http.Request, owner string) error {
	author, title, text, category := strings.TrimSpace(r.FormValue("author")), strings.TrimSpace(r.FormValue("title")), strings.TrimSpace(r.FormValue("text")), r.FormValue("category")
	if author == "" || title == "" || text == "" || utf8.RuneCountInString(author) > 30 || utf8.RuneCountInString(title) > 120 || utf8.RuneCountInString(text) > 10000 || communityCategory(category) == "" {
		return errors.New("작성자·분류·제목·내용을 확인해 주세요. 작성자 30자, 제목 120자, 내용 10,000자 이내입니다.")
	}
	n := CommunityPost{ID: token(), Owner: owner, CreatedAt: time.Now()}
	id := r.FormValue("id")
	if id != "" {
		old := findCommunity(&a.state, id)
		if old == nil || old.Owner != owner {
			return errors.New("이 브라우저에서 작성한 글만 수정할 수 있습니다.")
		}
		n = *old
		if n.Revision != number(r.FormValue("revision")) {
			return errors.New("글이 변경되었습니다. 새로고침 후 수정해 주세요.")
		}
	}
	removedPhotos := append([]Photo{}, n.Photos...)
	photos := removePhotos(n.Photos, r.Form["remove_photo"])
	count := 0
	if r.MultipartForm != nil {
		count = len(r.MultipartForm.File["photos"])
	}
	if len(photos)+count > 5 {
		return errors.New("사진은 최대 5장입니다.")
	}
	added, err := a.uploadCommunityPhotos(r)
	if err != nil {
		return err
	}
	n.Author, n.Title, n.Text, n.Category = author, title, text, category
	n.Photos = append(photos, added...)
	n.Revision++
	n.UpdatedAt = time.Now()
	err = a.commitAnnouncement(func(s *State) error {
		if id == "" {
			s.Community = append(s.Community, n)
		} else {
			*findCommunity(s, id) = n
		}
		return nil
	})
	if err == nil {
		a.pruneCommunityPhotos(removedPhotos)
	}
	return err
}

// Delete only unreferenced community attachment files after the state is durable.
func (a *App) pruneCommunityPhotos(candidates []Photo) {
	used := map[string]bool{}
	for _, n := range a.state.Community {
		for _, p := range n.Photos {
			used[p.ID] = true
		}
	}
	for _, n := range a.state.Announcements {
		for _, p := range n.Photos {
			used[p.ID] = true
		}
	}
	for _, n := range a.state.Projects {
		for _, p := range n.Photos {
			used[p.ID] = true
		}
	}
	for _, n := range a.state.ClientNotes {
		for _, p := range n.Photos {
			used[p.ID] = true
		}
	}
	for _, record := range a.state.ShippingRecords {
		for _, p := range append(append([]Photo{}, record.Project.Photos...), record.Photos...) {
			used[p.ID] = true
		}
	}
	for _, p := range candidates {
		if !used[p.ID] && validPhotoID.MatchString(p.ID) {
			_ = os.Remove(filepath.Join(a.dir, "photos", p.ID))
		}
	}
}
