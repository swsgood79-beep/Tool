package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var originalFiles = []string{"connection-v1.dat", "phone-push-v1.dat", "designers-v1.dat", "client-notes-v1.dat", "backup-v1.dat", "projects-v2.dat"}

const extensionKey = "_RecoveredV1"

// Original readers ignore this namespaced field. All original fields are independently materialized.
// The extension contains only recovery-specific state; original fields remain authoritative if another
// application writes them and removes the extension.
type directExtension struct {
	Version   int
	CoreHash  string
	AuxHashes map[string]string
	State     State
}

func jsonRaw(v any) json.RawMessage {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}
func rawObject(b []byte) (map[string]json.RawMessage, error) {
	out := map[string]json.RawMessage{}
	if len(b) == 0 || string(b) == "null" {
		return out, nil
	}
	e := json.Unmarshal(b, &out)
	if out == nil {
		out = map[string]json.RawMessage{}
	}
	return out, e
}
func objectFor(s State, name string) (map[string]json.RawMessage, error) {
	return rawObject(s.LegacySnapshots[name])
}
func patch(dst map[string]json.RawMessage, v any) {
	var src map[string]json.RawMessage
	json.Unmarshal(jsonRaw(v), &src)
	for k, b := range src {
		dst[k] = b
	}
}
func timeString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
func originalStage(st Stage, batch int64) map[string]any {
	return map[string]any{"Worker": st.Worker, "Requested": timeString(st.Requested), "Completed": timeString(st.Completed), "EventID": st.EventID, "BatchID": batch}
}
func originalPhoto(p Photo) map[string]any {
	return map[string]any{"ID": p.ID, "Name": p.Name, "Width": p.Width, "Height": p.Height, "Size": p.Size}
}
func photosProjection(ps []Photo) []map[string]any {
	out := []map[string]any{}
	for _, p := range ps {
		out = append(out, originalPhoto(p))
	}
	return out
}
func objectsByID(raw json.RawMessage, key string) map[string]map[string]json.RawMessage {
	out := map[string]map[string]json.RawMessage{}
	var rows []map[string]json.RawMessage
	json.Unmarshal(raw, &rows)
	for _, row := range rows {
		id := strings.TrimSpace(string(row[key]))
		if strings.HasPrefix(id, `"`) {
			var value string
			if json.Unmarshal(row[key], &value) != nil {
				continue
			}
			id = value
		}
		out[id] = row
	}
	return out
}

// Preserve unknown original keys rather than rebuilding a lossy subset of HostData.
func (a *App) projectOriginal(s *State) (map[string][]byte, error) {
	if s.LegacySnapshots == nil {
		s.LegacySnapshots = map[string]json.RawMessage{}
	}
	if s.MaterializedPhotos == nil {
		s.MaterializedPhotos = map[string]string{}
	}
	if err := a.normalizePhotos(s); err != nil {
		return nil, err
	}
	host, e := objectFor(*s, "projects-v2.dat")
	if e != nil {
		return nil, e
	}
	delete(host, extensionKey)
	if len(host["SigningKey"]) == 0 {
		host["SigningKey"] = jsonRaw(token())
	}
	oldProjects := objectsByID(host["Projects"], "ID")
	oldEvents := objectsByID(host["Events"], "id")
	orders := []map[string]json.RawMessage{}
	for i := range s.Projects {
		p := &s.Projects[i]
		old := oldProjects[fmtID(p.ID)]
		if old == nil {
			old = map[string]json.RawMessage{}
		}
		for _, d := range s.Designers {
			if d.ID == p.DesignerID {
				p.DesignerName = d.Name
				p.DesignerPhone = d.Phone
				break
			}
		}
		created := old["Created"]
		if len(created) == 0 {
			created = jsonRaw(time.Now().UTC().Format(time.RFC3339Nano))
		}
		batch := int64(0)
		if p.CombinedPending {
			batch = p.Setting.EventID
		}
		completed := timeString(p.Shipping.Completed)
		if p.LegacyCompleted && completed == "" {
			var previous string
			json.Unmarshal(old["Completed"], &previous)
			completed = previous
			if completed == "" {
				return nil, errors.New("검사 구분 없는 이전 완료 시각을 보존할 수 없습니다")
			}
		}
		photos := append([]Photo{}, p.Photos...)
		sourceIDs := []string{}
		sourceRevision := int64(0)
		if p.ClientID != "" {
			for _, c := range s.ClientNotes {
				if c.ID != p.ClientID {
					continue
				}
				sourceRevision = c.Revision
				p.SpecialInstructions = c.Text
				for _, source := range c.Photos {
					selected := !p.ClientPhotosSelected
					for _, id := range p.ClientPhotoIDs {
						if id == source.ID {
							selected = true
						}
					}
					if !selected {
						continue
					}
					sourceIDs = append(sourceIDs, source.ID)
					copy, e := a.materializeSourcePhoto(s, p.ID, source)
					if e != nil {
						return nil, e
					}
					photos = append(photos, copy)
				}
			}
		}
		if len(photos) > 5 {
			return nil, errors.New("원본 형식은 수주당 사진 5장까지 저장할 수 있습니다")
		}
		patch(old, map[string]any{"ID": p.ID, "Number": p.Number, "Name": p.Name, "RejectedInspection": p.RejectedInspection, "Created": json.RawMessage(created), "Active": !p.Hidden && completed == "", "Completed": completed, "Setting": originalStage(p.Setting, batch), "Shipping": originalStage(p.Shipping, batch), "Generation": p.Generation, "Revision": p.Revision, "InfoRevision": p.InfoRevision, "ShippingDueDate": p.ShippingDueDate, "ShippingDueManaged": p.ShippingDueManaged, "SuppliedMaterials": p.SuppliedMaterials, "SuppliedMaterialsColor": p.SuppliedMaterialsColor, "SuppliedMaterialsManaged": p.SuppliedMaterialsManaged, "DesignerID": p.DesignerID, "DesignerName": p.DesignerName, "DesignerPhone": p.DesignerPhone, "ClientNoteID": p.ClientID, "ClientSourceRevision": sourceRevision, "ClientPhotoIDs": sourceIDs, "SpecialInstructions": p.SpecialInstructions, "AdditionalInstructions": p.AdditionalInstructions, "Photos": photosProjection(photos)})
		orders = append(orders, old)
	}
	events := []map[string]json.RawMessage{}
	for _, ev := range s.Events {
		old := oldEvents[fmtID(ev.ID)]
		if old == nil {
			old = map[string]json.RawMessage{}
		}
		readAt := ""
		if !ev.Unread {
			json.Unmarshal(old["read_at"], &readAt)
			if readAt == "" {
				readAt = time.Now().UTC().Format(time.RFC3339Nano)
			}
		}
		kind := ev.Inspection
		if ev.IsNote {
			kind = "note"
		}
		completed := timeString(ev.Completed)
		if completed == "" && ev.Pending {
			completed = timeString(ev.RequestedAt)
		}
		submission := old["Submission"]
		if len(submission) == 0 {
			submission = jsonRaw(token())
		}
		patch(old, map[string]any{"id": ev.ID, "project_id": ev.ProjectID, "order_number": ev.Order, "name": ev.Project, "worker_name": ev.Worker, "inspection": kind, "pending_approval": ev.Pending, "rejected": ev.Rejected, "rejected_at": timeString(ev.RejectedAt), "requested_at": timeString(ev.RequestedAt), "completed_at": completed, "read_at": readAt, "batch_id": ev.BatchID, "note": ev.Note, "revision": ev.Revision, "Submission": json.RawMessage(submission)})
		events = append(events, old)
	}
	// bcrypt is the original password format. A v0.1/v0.2 PBKDF2 password stays only
	// in the extension until the next successful password login rehashes it.
	password := s.PasswordHash
	mobile := s.Integration.MobileEnabled
	if password != "" && !strings.HasPrefix(password, "$2") {
		password = ""
		mobile = false
	}
	patch(host, map[string]any{"Version": 14, "BoardToken": s.Board, "NextProject": s.NextID + 1, "NextEvent": s.NextID + 1, "Projects": orders, "Events": events, "Market": s.Integration.Market, "ScheduleSource": s.Integration.Schedule, "ScheduleApplied": s.ScheduleApplied, "SourcesLinked": true, "MobileAdmin": map[string]any{"Enabled": mobile, "PasswordHash": password, "Revision": s.Integration.Revision}, "Passkeys": s.Integration.Passkeys})
	result := map[string][]byte{}
	aux := map[string]any{
		"connection-v1.dat": map[string]any{"Provider": s.Integration.Connection.Provider, "Domain": s.Integration.Connection.Domain, "AuthToken": s.Integration.Connection.AuthToken, "GitHubToken": s.Integration.Connection.GitHubToken},
		"phone-push-v1.dat": s.Integration.Phone,
		"designers-v1.dat":  map[string]any{"Version": 1, "People": s.Designers},
		"backup-v1.dat":     s.Integration.Backup,
	}
	notes := []map[string]any{}
	for _, c := range s.ClientNotes {
		notes = append(notes, map[string]any{"ID": c.ID, "Name": c.Name, "Text": c.Text, "Revision": c.Revision, "Photos": photosProjection(c.Photos)})
	}
	aux["client-notes-v1.dat"] = map[string]any{"Version": 1, "Notes": notes}
	for name, v := range aux {
		obj, e := objectFor(*s, name)
		if e != nil {
			return nil, e
		}
		patch(obj, v)
		if name == "phone-push-v1.dat" || name == "backup-v1.dat" {
			obj["Version"] = jsonRaw(1)
		}
		result[name] = jsonRaw(obj)
	}
	core := jsonRaw(host)
	s.LegacySnapshots["projects-v2.dat"] = core
	for name, b := range result {
		s.LegacySnapshots[name] = b
	}
	extState, e := clone(*s)
	if e != nil {
		return nil, e
	}
	extState.LegacyHashes = nil
	extState.LegacySourceHash = ""
	extState.LegacySnapshots = nil
	auxHashes := map[string]string{}
	for name, b := range result {
		auxHashes[name] = digest(b)
	}
	host[extensionKey] = jsonRaw(directExtension{1, digest(core), auxHashes, extState})
	result["projects-v2.dat"] = jsonRaw(host)
	if err := validateOriginalProjection(result); err != nil {
		return nil, err
	}
	return result, nil
}
func validateOriginalProjection(files map[string][]byte) error {
	var h LegacyHost
	if e := json.Unmarshal(files["projects-v2.dat"], &h); e != nil {
		return e
	}
	if h.Version != 14 || !validPhotoID.MatchString(h.BoardToken) {
		return errors.New("원본 저장 형식 검증 실패")
	}
	seen := map[int64]bool{}
	for _, p := range h.Projects {
		if p.ID <= 0 || seen[p.ID] || !validText(p.Number, 80) || !validText(p.Name, 120) || len(p.Photos) > 5 {
			return errors.New("원본 수주 저장 형식 오류")
		}
		seen[p.ID] = true
		for _, ph := range p.Photos {
			if !validPhotoID.MatchString(ph.ID) || ph.Size <= 0 || ph.Size > 8<<20 {
				return errors.New("원본 사진 메타데이터 오류")
			}
		}
	}
	seen = map[int64]bool{}
	for _, e := range h.Events {
		if e.ID <= 0 || seen[e.ID] {
			return errors.New("원본 기록 번호 오류")
		}
		seen[e.ID] = true
	}
	if h.MobileAdmin.PasswordHash != "" {
		if _, e := bcrypt.Cost([]byte(h.MobileAdmin.PasswordHash)); e != nil {
			return errors.New("원본 비밀번호 형식 오류")
		}
	}
	for name, b := range files {
		if !json.Valid(b) {
			return fmt.Errorf("%s JSON 검증 실패", name)
		}
	}
	return nil
}
func loadDirect(dir string) (State, bool, error) {
	b, e := os.ReadFile(filepath.Join(dir, "projects-v2.dat"))
	if e != nil {
		return State{}, false, e
	}
	plain, e := unprotect(b)
	if e != nil {
		return State{}, false, e
	}
	obj, e := rawObject(plain)
	if e != nil {
		return State{}, false, e
	}
	raw, ok := obj[extensionKey]
	if !ok {
		return State{}, false, nil
	}
	var ext directExtension
	if e = json.Unmarshal(raw, &ext); e != nil || ext.Version != 1 {
		return State{}, false, errors.New("지원하지 않거나 손상된 직접 저장 확장 형식")
	}
	delete(obj, extensionKey)
	if digest(jsonRaw(obj)) != ext.CoreHash {
		return State{}, false, nil
	}
	for name, expected := range ext.AuxHashes {
		if !isOriginalFile(name) || name == "projects-v2.dat" {
			return State{}, false, errors.New("잘못된 확장 파일 이름")
		}
		var decoded json.RawMessage
		if e = legacyJSON(filepath.Join(dir, name), &decoded); e != nil {
			return State{}, false, e
		}
		if digest(decoded) != expected {
			return State{}, false, nil
		}
	}
	s := ext.State
	if s.Version != 1 || !validPhotoID.MatchString(s.Board) {
		return State{}, false, errors.New("직접 저장 상태 오류")
	}
	var h LegacyHost
	json.Unmarshal(plain, &h)
	if h.BoardToken != s.Board {
		return State{}, false, errors.New("원본/확장 작업자 키 불일치")
	}
	s.LegacySnapshots = map[string]json.RawMessage{"projects-v2.dat": jsonRaw(obj)}
	s.LegacyHashes = map[string]string{}
	for _, name := range originalFiles {
		cipher, e := os.ReadFile(filepath.Join(dir, name))
		if e != nil {
			return State{}, false, e
		}
		decoded, e := unprotect(cipher)
		if e != nil {
			return State{}, false, e
		}
		if name != "projects-v2.dat" {
			s.LegacySnapshots[name] = decoded
		}
		s.LegacyHashes[name] = digest(cipher)
	}
	s.LegacySourceHash = s.LegacyHashes["projects-v2.dat"]
	return s, true, nil
}
func isOriginalFile(name string) bool {
	for _, n := range originalFiles {
		if n == name {
			return true
		}
	}
	return false
}
func backupOnce(path string) error {
	src, e := os.ReadFile(path)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	dst := path + ".bak"
	f, e := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(e) {
		old, e := os.ReadFile(dst)
		if e != nil {
			return e
		}
		decoded, e := unprotect(old)
		if e != nil || !json.Valid(decoded) {
			return fmt.Errorf("기존 %s 백업을 검증할 수 없습니다. 덮어쓰지 않았습니다", filepath.Base(dst))
		}
		return nil
	}
	if e != nil {
		return e
	}
	ok := false
	defer func() {
		f.Close()
		if !ok {
			os.Remove(dst)
		}
	}()
	if _, e = f.Write(src); e != nil {
		return e
	}
	if e = f.Sync(); e != nil {
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	check, e := os.ReadFile(dst)
	if e != nil || !bytes.Equal(check, src) {
		return errors.New("초기 백업 검증 실패")
	}
	ok = true
	return nil
}
func backupOriginals(dir string) error {
	for _, name := range append(append([]string{}, originalFiles...), "reconstructed-state.bin") {
		if e := backupOnce(filepath.Join(dir, name)); e != nil {
			return e
		}
	}
	return nil
}
