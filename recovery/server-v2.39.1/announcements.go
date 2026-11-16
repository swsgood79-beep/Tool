package main

import (
	"bytes"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type Announcement struct {
	ID, Title, Text           string
	Urgent, Pinned, Published bool
	Revision                  int64
	CreatedAt, UpdatedAt      time.Time
	Photos                    []Photo
}

func findAnnouncement(s *State, id string) *Announcement {
	for i := range s.Announcements {
		if s.Announcements[i].ID == id {
			return &s.Announcements[i]
		}
	}
	return nil
}
func announcements(s State, publishedOnly, pinnedOnly bool) []Announcement {
	out := []Announcement{}
	for _, n := range s.Announcements {
		if publishedOnly && !n.Published || pinnedOnly && !n.Pinned {
			continue
		}
		out = append(out, n)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Urgent != out[j].Urgent {
			return out[i].Urgent
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}
func (a *App) announcementView(worker bool) map[string]any {
	pinned := announcements(a.state, true, true)
	if len(pinned) > 3 {
		pinned = pinned[:3]
	}
	return map[string]any{"NoticeWorker": worker, "Board": a.state.Board, "WorkerURL": "/work/" + a.state.Board, "Announcements": announcements(a.state, worker, false), "PinnedAnnouncements": pinned, "PublishedCount": len(announcements(a.state, true, false))}
}
func (a *App) announcementPanel() string {
	var b bytes.Buffer
	if err := templates.ExecuteTemplate(&b, "announcement-panel", a.announcementView(true)); err != nil {
		return ""
	}
	return b.String()
}

// Public routes use the same unguessable board capability as the worker screen.
// Ended notices and their attachments are never served to workers.
func (a *App) announcementWorker(w http.ResponseWriter, r *http.Request) bool {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 0 || (parts[0] != "notices" && parts[0] != "notice-panel" && parts[0] != "notice-photo") {
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	if len(parts) < 2 || !equal(parts[1], a.state.Board) {
		http.NotFound(w, r)
		return true
	}
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		http.Error(w, "공지사항은 조회만 가능합니다", 405)
		return true
	}
	v := a.announcementView(true)
	switch {
	case parts[0] == "notice-panel" && len(parts) == 2:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(a.announcementPanel()))
	case parts[0] == "notices" && len(parts) == 2:
		a.render(w, "announcements.html", v)
	case parts[0] == "notices" && len(parts) == 3:
		n := findAnnouncement(&a.state, parts[2])
		if n == nil || !n.Published {
			http.NotFound(w, r)
			return true
		}
		v["Notice"] = *n
		a.render(w, "announcement-detail.html", v)
	case parts[0] == "notice-photo" && len(parts) == 4:
		n := findAnnouncement(&a.state, parts[2])
		if n == nil || !n.Published {
			http.NotFound(w, r)
			return true
		}
		a.photo(w, r, n.Photos, parts[3])
	default:
		http.NotFound(w, r)
	}
	return true
}
func (a *App) announcementAdminGET(w http.ResponseWriter, r *http.Request, v map[string]any) bool {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 0 || parts[0] != "announcements" {
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	for k, value := range a.announcementView(false) {
		v[k] = value
	}
	switch {
	case len(parts) == 1:
		a.render(w, "announcements.html", v)
	case len(parts) == 2 && parts[1] == "new":
		v["Notice"] = Announcement{Urgent: true, Pinned: true, Published: true}
		a.render(w, "announcement-edit.html", v)
	case len(parts) == 2 || len(parts) == 3 && parts[2] == "edit":
		n := findAnnouncement(&a.state, parts[1])
		if n == nil {
			http.NotFound(w, r)
			return true
		}
		v["Notice"] = *n
		if len(parts) == 3 {
			a.render(w, "announcement-edit.html", v)
		} else {
			a.render(w, "announcement-detail.html", v)
		}
	case len(parts) == 4 && parts[2] == "photos":
		n := findAnnouncement(&a.state, parts[1])
		if n == nil {
			http.NotFound(w, r)
			return true
		}
		a.photo(w, r, n.Photos, parts[3])
	default:
		http.NotFound(w, r)
	}
	return true
}

// Notice-only saves reuse the journaled state store without rewriting the order
// workbook or generating an inspection event, popup or TTS message.
func (a *App) commitAnnouncement(fn func(*State) error) error {
	if err := a.checkLegacySource(); err != nil {
		return err
	}
	s, err := clone(a.state)
	if err != nil {
		return err
	}
	if err = fn(&s); err != nil {
		return err
	}
	if err = a.persist(&s); err != nil {
		return err
	}
	a.state = s
	return nil
}
func (a *App) saveAnnouncement(r *http.Request) error {
	id := r.FormValue("id")
	title := strings.TrimSpace(r.FormValue("title"))
	text := strings.TrimSpace(r.FormValue("text"))
	if title == "" || text == "" || utf8.RuneCountInString(title) > 120 || utf8.RuneCountInString(text) > 10000 {
		return errors.New("제목(120자 이내)과 공지 내용(10,000자 이내)을 입력해 주세요.")
	}
	n := Announcement{ID: token(), CreatedAt: time.Now(), Published: true}
	if id != "" {
		old := findAnnouncement(&a.state, id)
		if old == nil {
			return errors.New("공지사항을 찾을 수 없습니다.")
		}
		n = *old
		if n.Revision != number(r.FormValue("revision")) {
			return errors.New("다른 화면에서 변경되었습니다. 새로고침 후 다시 수정해 주세요.")
		}
	}
	photos := removePhotos(n.Photos, r.Form["remove_photo"])
	uploads := 0
	if r.MultipartForm != nil {
		uploads = len(r.MultipartForm.File["photos"])
	}
	if len(photos)+uploads > 5 {
		return errors.New("첨부 사진은 기존 사진을 포함해 최대 5장입니다.")
	}
	added, err := a.uploadPhotos(r)
	if err != nil {
		return err
	}
	n.Title, n.Text = title, text
	n.Urgent = r.FormValue("urgent") == "1"
	n.Pinned = r.FormValue("pinned") == "1"
	n.Published = r.FormValue("published") == "1"
	n.Photos = append(photos, added...)
	n.UpdatedAt = time.Now()
	n.Revision++
	return a.commitAnnouncement(func(s *State) error {
		if id == "" {
			s.Announcements = append(s.Announcements, n)
		} else {
			*findAnnouncement(s, id) = n
		}
		return nil
	})
}
func (a *App) announcementAdminPOST(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/announcements/save" && r.URL.Path != "/announcements/end" {
		return false
	}
	var err error
	if r.URL.Path == "/announcements/save" {
		err = a.saveAnnouncement(r)
	} else {
		err = a.commitAnnouncement(func(s *State) error {
			n := findAnnouncement(s, r.FormValue("id"))
			if n == nil {
				return errors.New("공지사항을 찾을 수 없습니다.")
			}
			if n.Revision != number(r.FormValue("revision")) {
				return errors.New("다른 화면에서 변경되었습니다. 새로고침해 주세요.")
			}
			n.Published = false
			n.UpdatedAt = time.Now()
			n.Revision++
			return nil
		})
	}
	if err != nil {
		http.Error(w, err.Error(), 400)
	} else {
		http.Redirect(w, r, "/announcements", http.StatusSeeOther)
	}
	return true
}
