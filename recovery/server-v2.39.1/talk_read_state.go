package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
)

type talkReadUpdate struct {
	CSRF   string   `json:"csrf"`
	Reader string   `json:"reader"`
	Source string   `json:"source_reader"`
	IDs    []string `json:"read_ids"`
}

// The credential identifies reading state only. It never grants posting/editing ownership.
func (a *App) talkReadHTTP(w http.ResponseWriter, r *http.Request) bool {
	const prefix = "/talk-read-state/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		return false
	}
	if !equal(strings.TrimPrefix(r.URL.Path, prefix), a.state.Board) {
		http.NotFound(w, r)
		return true
	}
	if r.Method != "POST" {
		w.WriteHeader(405)
		return true
	}
	r.Body = http.MaxBytesReader(w, r.Body, 512<<10)
	var in talkReadUpdate
	if json.NewDecoder(r.Body).Decode(&in) != nil || !equal(in.CSRF, a.state.Board) || !validPhotoID.MatchString(in.Reader) || (in.Source != "" && !validPhotoID.MatchString(in.Source)) || len(in.IDs) > 5000 {
		http.Error(w, "읽음 기록 요청 오류", 400)
		return true
	}
	key := hashToken(a.state.Board + ":" + in.Reader)
	source := ""
	if in.Source != "" {
		source = hashToken(a.state.Board + ":" + in.Source)
	}
	live := map[string]bool{}
	for _, p := range a.state.Community {
		live[p.ID] = true
	}
	set := map[string]bool{}
	for _, ids := range [][]string{a.state.TalkReaders[key], a.state.TalkReaders[source], in.IDs} {
		for _, id := range ids {
			if live[id] {
				set[id] = true
			}
		}
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) > 5000 {
		http.Error(w, "읽음 기록 한도를 확인해 주세요.", 429)
		return true
	}
	old := a.state.TalkReaders[key]
	changed := len(old) != len(ids)
	if !changed {
		for i := range ids {
			if old[i] != ids[i] {
				changed = true
				break
			}
		}
	}
	if changed {
		if _, found := a.state.TalkReaders[key]; !found && len(a.state.TalkReaders) >= 1024 {
			http.Error(w, "읽음 연결 수를 확인해 주세요.", 429)
			return true
		}
		if err := a.commitAnnouncement(func(s *State) error {
			if s.TalkReaders == nil {
				s.TalkReaders = map[string][]string{}
			}
			if len(ids) == 0 {
				delete(s.TalkReaders, key)
			} else {
				s.TalkReaders[key] = ids
			}
			return nil
		}); err != nil {
			http.Error(w, "읽음 기록 저장 실패", 500)
			return true
		}
	}
	jsonReply(w, map[string]any{"read_ids": ids})
	return true
}
