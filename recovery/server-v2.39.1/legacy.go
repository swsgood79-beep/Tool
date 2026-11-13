package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Original field names and JSON tags are copied from recovered RTTI.
type LegacyStage struct {
	BatchID                      int64
	Requested, Worker, Completed string
	EventID                      int64
}
type LegacyOrder struct {
	RejectedInspection                                                                                                                                    string
	SuppliedMaterialsColor                                                                                                                                string
	SuppliedMaterials                                                                                                                                     string
	SuppliedMaterialsManaged                                                                                                                              bool
	ClientPhotoIDs                                                                                                                                        []string
	ClientSourceRevision                                                                                                                                  int64
	ID, Generation, Revision, InfoRevision                                                                                                                int64
	Number, Name, Created, Completed, ShippingDueDate, DesignerID, DesignerName, DesignerPhone, ClientNoteID, SpecialInstructions, AdditionalInstructions string
	Active, ShippingDueManaged                                                                                                                            bool
	Setting, Shipping                                                                                                                                     LegacyStage
	Photos                                                                                                                                                []Photo
}
type LegacyEvent struct {
	Rejected    bool   `json:"rejected,omitempty"`
	RejectedAt  string `json:"rejected_at,omitempty"`
	BatchID     int64  `json:"batch_id,omitempty"`
	Pending     bool   `json:"pending_approval,omitempty"`
	RequestedAt string `json:"requested_at,omitempty"`
	ReadAt      string `json:"read_at,omitempty"`
	Note        string `json:"note,omitempty"`
	Revision    int64  `json:"revision,omitempty"`
	Inspection  string `json:"inspection,omitempty"`
	ProjectID   int64  `json:"project_id,omitempty"`
	ID          int64  `json:"id"`
	Worker      string `json:"worker_name"`
	Project     string `json:"name"`
	Order       string `json:"order_number"`
	Completed   string `json:"completed_at"`
	Submission  string
}
type LegacyHost struct {
	Version                int
	BoardToken, SigningKey string
	NextProject, NextEvent int64
	Projects               []LegacyOrder
	Events                 []LegacyEvent
	Market                 MarketConfig
	ScheduleSource         ScheduleConfig
	MobileAdmin            struct {
		Enabled      bool
		PasswordHash string
		Revision     int64
	}
	Passkeys PasskeyData
}

func legacyDir() (string, error) {
	d, e := os.UserConfigDir()
	if e != nil {
		return "", e
	}
	return filepath.Join(d, "WanryoTalk"), nil
}
func legacyHash(path string) (string, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return "", e
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
func legacyJSON(path string, v any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	b, e = unprotect(b)
	if e != nil {
		return e
	}
	if len(b) > 128<<20 {
		return errors.New("원본 자료가 너무 큽니다")
	}
	return json.Unmarshal(b, v)
}
func legacyTime(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
		if t, e := time.Parse(layout, s); e == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("원본 기록의 날짜 형식을 확인할 수 없습니다: %q", s)
}
func importLegacy(dir string) (State, error) {
	var h LegacyHost
	path := filepath.Join(dir, "projects-v2.dat")
	if e := legacyJSON(path, &h); e != nil {
		return State{}, e
	}
	if h.Version < 5 || h.Version > 14 || !validPhotoID.MatchString(h.BoardToken) {
		return State{}, errors.New("확인되지 않은 원본 버전 또는 작업자 QR 키")
	}
	hash, e := legacyHash(path)
	if e != nil {
		return State{}, e
	}
	s := State{Version: 1, Board: h.BoardToken, NextID: max(h.NextProject, h.NextEvent), Projects: []Project{}, Events: []Event{}, Designers: []Designer{}, ClientNotes: []ClientNote{}, LegacySourceHash: hash}
	s.PasswordHash = h.MobileAdmin.PasswordHash
	s.Integration = IntegrationConfig{Market: h.Market, Schedule: h.ScheduleSource, Passkeys: h.Passkeys, MobileEnabled: h.MobileAdmin.Enabled, Revision: h.MobileAdmin.Revision}
	seen := map[int64]bool{}
	for _, old := range h.Projects {
		if old.ID <= 0 || seen[old.ID] || !validText(old.Number, 80) || !validText(old.Name, 120) {
			return State{}, errors.New("원본 수주 식별자 또는 필드 오류")
		}
		seen[old.ID] = true
		p := Project{ID: old.ID, Revision: max(old.Revision, 1), InfoRevision: max(old.InfoRevision, 1), Number: old.Number, Name: old.Name, RejectedInspection: old.RejectedInspection, SuppliedMaterials: old.SuppliedMaterials, SuppliedMaterialsColor: old.SuppliedMaterialsColor, SuppliedMaterialsManaged: old.SuppliedMaterialsManaged, ShippingDueDate: old.ShippingDueDate, DesignerID: old.DesignerID, DesignerName: old.DesignerName, DesignerPhone: old.DesignerPhone, ClientID: old.ClientNoteID, ClientPhotoIDs: old.ClientPhotoIDs, ClientPhotosSelected: old.ClientPhotoIDs != nil, SpecialInstructions: old.SpecialInstructions, AdditionalInstructions: old.AdditionalInstructions, Photos: old.Photos, LegacyCompleted: old.Completed != "" && old.Shipping.Completed == "", Hidden: !old.Active && old.Completed == "", ShippingDueManaged: old.ShippingDueManaged}
		for i, st := range []LegacyStage{old.Setting, old.Shipping} {
			completed, e := legacyTime(st.Completed)
			if e != nil {
				return State{}, e
			}
			requested, e := legacyTime(st.Requested)
			if e != nil {
				return State{}, e
			}
			next := Stage{Worker: st.Worker, Pending: requested != nil && completed == nil, Requested: requested, Completed: completed, EventID: st.EventID}
			if i == 0 {
				p.Setting = next
			} else {
				p.Shipping = next
			}
		}
		p.CombinedPending = p.Setting.Pending && p.Shipping.Pending && old.Setting.BatchID != 0 && old.Setting.BatchID == old.Shipping.BatchID
		p.rotate()
		p.Generation = max(old.Generation, p.Generation)
		for i := range p.Photos {
			if !validPhotoID.MatchString(p.Photos[i].ID) {
				return State{}, errors.New("원본 사진 ID 오류")
			}
			p.Photos[i].Legacy = true
			p.Photos[i].MIME = "image/jpeg"
		}
		s.NextID = max(s.NextID, p.ID)
		s.Projects = append(s.Projects, p)
	}
	seen = map[int64]bool{}
	for _, old := range h.Events {
		if old.ID <= 0 || seen[old.ID] {
			return State{}, errors.New("원본 이벤트 ID 중복")
		}
		seen[old.ID] = true
		completed, e := legacyTime(old.Completed)
		if e != nil {
			return State{}, e
		}
		requested, e := legacyTime(old.RequestedAt)
		if e != nil {
			return State{}, e
		}
		rejectedAt, e := legacyTime(old.RejectedAt)
		if e != nil {
			return State{}, e
		}
		s.Events = append(s.Events, Event{ID: old.ID, Rejected: old.Rejected, RejectedAt: rejectedAt, BatchID: old.BatchID, ProjectID: old.ProjectID, Revision: max(old.Revision, 1), Order: old.Order, Project: old.Project, Worker: old.Worker, Note: old.Note, Inspection: old.Inspection, InspectionName: stageLabel(old.Inspection), IsNote: old.Inspection == "note" || old.Note != "" && old.Inspection == "", Pending: old.Pending, Unread: old.ReadAt == "", RequestedAt: requested, Completed: completed})
		s.NextID = max(s.NextID, old.ID)
	}
	aux := []struct {
		name   string
		target any
	}{{"connection-v1.dat", &s.Integration.Connection}, {"phone-push-v1.dat", &s.Integration.Phone}, {"backup-v1.dat", &s.Integration.Backup}}
	for _, x := range aux {
		if _, err := os.Stat(filepath.Join(dir, x.name)); err == nil {
			if err = legacyJSON(filepath.Join(dir, x.name), x.target); err != nil {
				return State{}, fmt.Errorf("%s: %w", x.name, err)
			}
		} else if !os.IsNotExist(err) {
			return State{}, err
		}
	}
	var designers struct {
		Version int
		People  []Designer
	}
	if _, e := os.Stat(filepath.Join(dir, "designers-v1.dat")); e == nil {
		if e = legacyJSON(filepath.Join(dir, "designers-v1.dat"), &designers); e != nil {
			return State{}, e
		}
		s.Designers = designers.People
	}
	var clients struct {
		Version int
		Notes   []ClientNote
	}
	if _, e := os.Stat(filepath.Join(dir, "client-notes-v1.dat")); e == nil {
		if e = legacyJSON(filepath.Join(dir, "client-notes-v1.dat"), &clients); e != nil {
			return State{}, e
		}
		s.ClientNotes = clients.Notes
		for i := range s.ClientNotes {
			for j := range s.ClientNotes[i].Photos {
				s.ClientNotes[i].Photos[j].Legacy = true
				s.ClientNotes[i].Photos[j].MIME = "image/jpeg"
			}
		}
	}
	// Expired endpoints must be re-established by the tunnel, never reused blindly.
	// Original linked photos were materialized as independent IDs. Match bytes,
	// rather than names, so source photos are not shown twice after linking live data.
	photoHashes := map[string]string{}
	photoHash := func(id string) string {
		if v, ok := photoHashes[id]; ok {
			return v
		}
		if !validPhotoID.MatchString(id) {
			return ""
		}
		b, e := os.ReadFile(filepath.Join(dir, "project-photos", id+".dat"))
		if e != nil {
			return ""
		}
		b, e = unprotect(b)
		if e != nil {
			return ""
		}
		v := digest(b)
		photoHashes[id] = v
		return v
	}
	for i := range s.Projects {
		p := &s.Projects[i]
		if p.ClientID == "" {
			continue
		}
		for _, c := range s.ClientNotes {
			if c.ID != p.ClientID {
				continue
			}
			for _, src := range c.Photos {
				selected := !p.ClientPhotosSelected
				for _, id := range p.ClientPhotoIDs {
					if src.ID == id {
						selected = true
					}
				}
				if !selected {
					continue
				}
				hash := photoHash(src.ID)
				if hash == "" {
					continue
				}
				for j, ph := range p.Photos {
					if photoHash(ph.ID) == hash {
						p.Photos = append(p.Photos[:j], p.Photos[j+1:]...)
						break
					}
				}
			}
		}
	}
	s.LegacySnapshots = map[string]json.RawMessage{}
	s.LegacyHashes = map[string]string{}
	for _, name := range []string{"projects-v2.dat", "connection-v1.dat", "phone-push-v1.dat", "backup-v1.dat", "designers-v1.dat", "client-notes-v1.dat"} {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return State{}, err
		}
		plain, err := unprotect(raw)
		if err != nil {
			return State{}, err
		}
		if !json.Valid(plain) {
			return State{}, errors.New("원본 JSON 오류")
		}
		s.LegacyHashes[name] = digest(raw)
		s.LegacySnapshots[name] = plain
	}
	if s.LegacyHashes["projects-v2.dat"] != hash {
		return State{}, errors.New("원본 자료가 읽는 동안 변경되었습니다")
	}
	s.Integration.Connection.PublicURL = ""
	return s, nil
}
func (a *App) checkLegacySource() error {
	for _, name := range originalFiles {
		if _, tracked := a.state.LegacyHashes[name]; !tracked {
			if _, e := os.Stat(filepath.Join(a.dir, name)); e == nil {
				return errors.New("다른 프로세스가 원본 저장 파일을 만들었습니다")
			} else if !os.IsNotExist(e) {
				return e
			}
		}
	}
	for name, expected := range a.state.LegacyHashes {
		hash, e := legacyHash(filepath.Join(a.dir, name))
		if e != nil || hash != expected {
			return errors.New("원본 설정·명단이 변경되었습니다. 원본 프로그램을 종료하고 자료 대조가 필요합니다")
		}
	}
	if a.state.LegacySourceHash == "" {
		return nil
	}
	h, e := legacyHash(filepath.Join(a.dir, "projects-v2.dat"))
	if e != nil {
		return e
	}
	if h != a.state.LegacySourceHash {
		return errors.New("원본 프로그램이 회사 자료를 변경했습니다. 두 프로그램을 함께 사용하지 말고 자료 대조가 필요합니다.")
	}
	return nil
}
func (a *App) legacyPhoto(p Photo) ([]byte, error) {
	if !validPhotoID.MatchString(p.ID) {
		return nil, errors.New("사진 ID 오류")
	}
	b, e := os.ReadFile(filepath.Join(a.dir, "project-photos", p.ID+".dat"))
	if e != nil {
		return nil, e
	}
	return unprotect(b)
}
func secureRequest(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
