package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"unicode/utf8"
)

type manualTransfer struct {
	ID, Owner, Board string
	Epoch            int64
	File             ShippingAttachment
	Updated          time.Time
}

func (a *App) manualStagePath(id string) string {
	return filepath.Join(a.dir, "manual-stages", id+".dat")
}
func (a *App) storeManualStage(s *manualTransfer) error {
	b, e := protect(jsonRaw(s))
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Join(a.dir, "manual-stages"), 0700); e != nil {
		return e
	}
	return atomicFile(a.manualStagePath(s.ID), b)
}

// Interrupted uploads have no committed references. Keep committed/manual and
// backup-pinned files; remove only this staging journal's unreferenced chunks.
func (a *App) cleanupManualStages() {
	entries, _ := os.ReadDir(filepath.Join(a.dir, "manual-stages"))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(a.dir, "manual-stages", entry.Name())
		b, e := os.ReadFile(path)
		if e != nil || len(b) > 128<<10 {
			continue
		}
		b, e = unprotect(b)
		if e != nil {
			continue
		}
		var s manualTransfer
		if json.Unmarshal(b, &s) != nil || !validPhotoID.MatchString(s.ID) || entry.Name() != s.ID+".dat" {
			continue
		}
		a.pruneCommunityPhotos(s.File.Parts)
		os.Remove(path)
	}
}
func (a *App) releaseManualStageLocked(s *manualTransfer) {
	for _, ph := range s.File.Parts {
		a.backupPins[ph.ID]--
		if a.backupPins[ph.ID] <= 0 {
			delete(a.backupPins, ph.ID)
		}
	}
	a.pruneCommunityPhotos(s.File.Parts)
	os.Remove(a.manualStagePath(s.ID))
	delete(a.manualTransfers, s.ID)
}
func (a *App) discardManualStage(s *manualTransfer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.releaseManualStageLocked(s)
}

func (a *App) manualTransferHTTP(w http.ResponseWriter, r *http.Request, action, owner, board string, epoch int64, valid func() bool) {
	if r.Method != "POST" {
		http.Error(w, "POST 요청이 필요합니다.", 405)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, shippingAttachmentChunk+1)
	if action != "upload-part" {
		if e := r.ParseForm(); e != nil {
			http.Error(w, "요청을 확인해 주세요.", 400)
			return
		}
	}
	a.manualTransferMu.Lock()
	defer a.manualTransferMu.Unlock()
	a.mu.Lock()
	ok := valid()
	a.mu.Unlock()
	if !ok {
		http.Error(w, "요청 인증을 확인해 주세요.", 403)
		return
	}
	if a.manualTransfers == nil {
		a.manualTransfers = map[string]*manualTransfer{}
	}
	for _, s := range a.manualTransfers {
		if s.Epoch != epoch || s.Board != board || time.Since(s.Updated) > 2*time.Hour {
			a.discardManualStage(s)
		}
	}
	if action == "upload-start" {
		size, e := strconv.ParseInt(r.FormValue("size"), 10, 64)
		name := cleanAttachmentName(r.FormValue("name"))
		if e != nil || size < 1 || size > manualFileLimit || name == "" || utf8.RuneCountInString(name) > 180 {
			http.Error(w, "파일은 한 개당 300MB 이하로 등록해 주세요.", 400)
			return
		}
		count := 0
		for _, s := range a.manualTransfers {
			if s.Owner == owner {
				count++
			}
		}
		if count >= 10 || len(a.manualTransfers) >= 50 {
			http.Error(w, "진행 중인 첨부를 완료하거나 잠시 후 다시 시도해 주세요.", 429)
			return
		}
		id := token()
		s := &manualTransfer{ID: id, Owner: owner, Board: board, Epoch: epoch, File: ShippingAttachment{ID: token(), Name: name, Size: size, Kind: "file", MIME: "application/octet-stream", CreatedAt: time.Now()}, Updated: time.Now()}
		if e := a.storeManualStage(s); e != nil {
			http.Error(w, "첨부 준비 실패", 500)
			return
		}
		a.manualTransfers[id] = s
		jsonReply(w, map[string]any{"id": id, "chunk": shippingAttachmentChunk})
		return
	}
	s := a.manualTransfers[r.FormValue("id")]
	if s == nil || s.Owner != owner || s.Board != board || s.Epoch != epoch {
		http.Error(w, "첨부 연결을 확인해 주세요.", 404)
		return
	}
	if action == "upload-cancel" {
		a.discardManualStage(s)
		jsonReply(w, map[string]bool{"ok": true})
		return
	}
	if action != "upload-part" {
		http.NotFound(w, r)
		return
	}
	i, e := strconv.Atoi(r.FormValue("index"))
	if e != nil || i < 0 || i > len(s.File.Parts) {
		http.Error(w, "첨부 순서 오류", 409)
		return
	}
	b, e := io.ReadAll(r.Body)
	want := min(int64(shippingAttachmentChunk), s.File.Size-int64(i)*shippingAttachmentChunk)
	if e != nil || want < 1 || int64(len(b)) != want {
		http.Error(w, "첨부 조각 용량 오류", 400)
		return
	}
	if i < len(s.File.Parts) {
		if digest(b) != s.File.Hashes[i] {
			http.Error(w, "다른 첨부 조각입니다.", 409)
			return
		}
		s.Updated = time.Now()
		jsonReply(w, map[string]bool{"ok": true})
		return
	}
	a.mu.Lock()
	ok = valid()
	a.mu.Unlock()
	if !ok {
		http.Error(w, "연결이 변경되었습니다.", 403)
		return
	}
	ph := Photo{ID: token(), Size: len(b)}
	encrypted, e := protect(b)
	if e == nil {
		e = os.MkdirAll(filepath.Join(a.dir, "photos"), 0700)
	}
	if e == nil {
		e = atomicFile(filepath.Join(a.dir, "photos", ph.ID), encrypted)
	}
	if e != nil {
		http.Error(w, "첨부 저장 실패", 500)
		return
	}
	s.File.Parts = append(s.File.Parts, ph)
	s.File.Hashes = append(s.File.Hashes, digest(b))
	s.Updated = time.Now()
	if e = a.storeManualStage(s); e != nil {
		s.File.Parts = s.File.Parts[:i]
		s.File.Hashes = s.File.Hashes[:i]
		a.mu.Lock()
		a.pruneCommunityPhotos([]Photo{ph})
		a.mu.Unlock()
		http.Error(w, "첨부 기록 저장 실패", 500)
		return
	}
	a.mu.Lock()
	if a.backupPins == nil {
		a.backupPins = map[string]int{}
	}
	a.backupPins[ph.ID]++
	a.mu.Unlock()
	jsonReply(w, map[string]bool{"ok": true})
}
func (a *App) finishManualTransfer(w http.ResponseWriter, r *http.Request, base string, post ManualPost, valid func() bool) {
	a.manualTransferMu.Lock()
	defer a.manualTransferMu.Unlock()
	a.mu.Lock()
	defer a.mu.Unlock()
	if !valid() {
		http.Error(w, "연결을 확인해 주세요.", 403)
		return
	}
	for _, old := range a.state.Manuals {
		if old.ID == post.ID {
			if old.Owner != post.Owner {
				http.Error(w, "등록 식별자가 다릅니다.", 409)
				return
			}
			for _, id := range r.Form["file_id"] {
				if staged := a.manualTransfers[id]; staged != nil && staged.Owner == post.Owner {
					a.releaseManualStageLocked(staged)
				}
			}
			http.Redirect(w, r, base+"/"+old.ID+"?saved=1", 303)
			return
		}
	}
	ids := r.Form["file_id"]
	if len(ids) < 1 || len(ids) > 5 {
		http.Error(w, "파일은 1~5개 등록해 주세요.", 400)
		return
	}
	seen := map[string]bool{}
	stages := []*manualTransfer{}
	for _, id := range ids {
		s := a.manualTransfers[id]
		if seen[id] || s == nil || s.Owner != post.Owner || s.Board != a.state.Board || s.Epoch != a.epoch || len(s.File.Parts) != int((s.File.Size+shippingAttachmentChunk-1)/shippingAttachmentChunk) {
			http.Error(w, "첨부 업로드를 완료한 뒤 다시 저장해 주세요.", 409)
			return
		}
		seen[id] = true
		post.Files = append(post.Files, s.File)
		stages = append(stages, s)
	}
	if e := a.commitAnnouncement(func(state *State) error { state.Manuals = append(state.Manuals, post); return nil }); e != nil {
		http.Error(w, fmt.Sprint(e), 500)
		return
	}
	for _, s := range stages {
		for _, ph := range s.File.Parts {
			a.backupPins[ph.ID]--
			if a.backupPins[ph.ID] <= 0 {
				delete(a.backupPins, ph.ID)
			}
		}
		delete(a.manualTransfers, s.ID)
		os.Remove(a.manualStagePath(s.ID))
	}
	http.Redirect(w, r, base+"/"+post.ID+"?saved=1", 303)
}
