package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Stage struct {
	Worker               string
	Pending              bool
	Requested, Completed *time.Time
	EventID              int64
}
type Photo struct {
	ID, Name, MIME string
	Width, Height  int
	Legacy         bool
	Size           int
}
type Project struct {
	RejectedInspection                                    string
	ID, Revision, InfoRevision, Generation                int64
	Number, Name, ShippingDueDate                         string
	SuppliedMaterials                                     string
	SuppliedMaterialsColor                                string
	SuppliedMaterialsManaged                              bool
	Hidden, CombinedPending                               bool
	LegacyCompleted, ShippingDueManaged                   bool
	Setting, Shipping                                     Stage
	DesignerID, DesignerName, DesignerPhone               string
	ClientID, SpecialInstructions, AdditionalInstructions string
	ClientPhotoIDs                                        []string
	ClientPhotosSelected                                  bool
	Photos                                                []Photo
	SettingForm, ShippingForm, CombinedForm, NoteForm     string
}
type Designer struct {
	ID, Name, Phone string
	Revision        int64
}
type ClientNote struct {
	ID, Name, Text string
	Revision       int64
	Photos         []Photo
}
type Event struct {
	Rejected                                                 bool
	RejectedAt                                               *time.Time
	ID, BatchID, ProjectID, Revision                         int64
	Order, Project, Worker, Note, Inspection, InspectionName string
	IsNote, Pending, Unread, Active                          bool
	RequestedAt, Completed                                   *time.Time
}
type State struct {
	WorkbookBase              []byte
	WorkbookBaseName          string
	WorkbookBindings          map[int64]WorkbookBinding
	Version                   int
	MaterializedPhotos        map[string]string
	Integration               IntegrationConfig
	LegacySourceHash          string
	LegacySnapshots           map[string]json.RawMessage
	LegacyHashes              map[string]string
	ScheduleApplied           string
	Board, Salt, PasswordHash string
	NextID                    int64
	Projects                  []Project
	Designers                 []Designer
	ClientNotes               []ClientNote
	Events                    []Event
}
type Session struct {
	CSRF    string
	Expires time.Time
}
type Ticket struct {
	Expires time.Time
	Next    string
}
type App struct {
	speechSynthesize func(context.Context, string) ([]byte, error)
	imports          map[string]*ImportDraft
	testMode         bool
	mu               sync.Mutex
	state            State
	dir              string
	sessions         map[string]Session
	challenges       map[string]Ticket
	failures         map[string][]time.Time
	reportError      string
	integration      IntegrationRuntime
	client           *http.Client
	localURL         string
	epoch            int64
	directHook       func(string, string) error
	storageNotice    string
}

var kst = time.FixedZone("Asia/Seoul", 9*3600)

func day(t time.Time) string { return t.In(kst).Format("2006-01-02") }
func token() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}
func (s *State) next() int64 { s.NextID++; return s.NextID }
func (s *State) project(id int64) *Project {
	for i := range s.Projects {
		if s.Projects[i].ID == id {
			return &s.Projects[i]
		}
	}
	return nil
}
func (p *Project) rotate() {
	p.SettingForm = token()
	p.ShippingForm = token()
	p.CombinedForm = token()
	p.NoteForm = token()
	p.Generation++
}
func (p *Project) touch() { p.Revision++; p.InfoRevision++ }
func clone(s State) (State, error) {
	b, e := json.Marshal(s)
	if e != nil {
		return State{}, e
	}
	var next State
	e = json.Unmarshal(b, &next)
	return next, e
}
func newApp(dir string) (*App, error) {
	if e := os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	if e := recoverDirect(dir); e != nil {
		return nil, fmt.Errorf("원본 저장 복구가 필요합니다: %w", e)
	}
	a := &App{dir: dir, sessions: map[string]Session{}, challenges: map[string]Ticket{}, failures: map[string][]time.Time{}}
	var have bool
	if _, e := os.Stat(filepath.Join(dir, "projects-v2.dat")); e == nil {
		state, ok, err := loadDirect(dir)
		if err != nil {
			return nil, err
		}
		if ok {
			a.state = state
			have = true
		} else {
			state, err = importLegacy(dir)
			if err != nil {
				return nil, err
			}
			a.state = state
			have = true
		}
	}
	// Upgrade v0.1/v0.2 once, only if the original source has not changed independently.
	if raw, e := os.ReadFile(filepath.Join(dir, "reconstructed-state.bin")); e == nil {
		original := a.state
		_, direct, _ := loadDirect(dir)
		if !direct {
			plain, err := unprotect(raw)
			if err != nil {
				return nil, err
			}
			var old State
			if err = json.Unmarshal(plain, &old); err != nil || old.Version != 1 {
				return nil, errors.New("이전 검증판 자료 오류")
			}
			if have && (old.Board != original.Board || old.LegacySourceHash == "" || old.LegacySourceHash != original.LegacySourceHash) {
				return nil, errors.New("원본과 이전 검증판 자료가 다릅니다. 자동 덮어쓰기를 중지했습니다")
			}
			a.state = old
			if err = a.checkLegacySource(); err != nil {
				return nil, err
			}
			have = true
		}
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	if !have {
		a.state = State{Version: 1, Board: token(), Projects: []Project{}, Events: []Event{}, Designers: []Designer{}, ClientNotes: []ClientNote{}, LegacyHashes: map[string]string{}}
	}
	if e := backupOriginals(dir); e != nil {
		return nil, e
	}
	a.state.Integration.Connection.PublicURL = ""
	n, e := clone(a.state)
	if e != nil {
		return nil, e
	}
	if e = a.persist(&n); e != nil {
		return nil, e
	}
	a.state = n
	a.initIntegration()
	return a, nil
}

func atomicFile(path string, b []byte) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".save-*")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(name, path)
}
func (a *App) persist(s *State) error { return a.writeOriginalGroup(s) }

// Caller holds a.mu. Changes become visible only after durable state succeeds.
func (a *App) transaction(fn func(*State) error) error {
	if err := a.checkLegacySource(); err != nil {
		return err
	}
	n, e := clone(a.state)
	if e != nil {
		return e
	}
	if e = fn(&n); e != nil {
		return e
	}
	if e = a.persist(&n); e != nil {
		return e
	}
	a.state = n
	b, e := workbook(n)
	if e == nil {
		e = atomicFile(filepath.Join(a.dir, "검사완료기록-복구서버.xlsx"), b)
	}
	a.reportError = ""
	if e != nil {
		a.reportError = e.Error()
	}
	return nil
}
func stageLabel(kind string) string {
	if kind == "combined" {
		return "세팅·출하 통합검사"
	}
	if kind == "setting" {
		return "세팅검사"
	}
	if kind == "shipping" {
		return "출하검사"
	}
	return "비고 메시지"
}
func equal(a, b string) bool { return hmac.Equal([]byte(a), []byte(b)) }
func hashToken(s string) string {
	x := sha256.Sum256([]byte(s))
	return base64.RawURLEncoding.EncodeToString(x[:])
}
func number(s string) int64 { n, _ := strconv.ParseInt(s, 10, 64); return n }

type ShipmentSummary struct {
	ProjectID                     int64 `json:"project_id"`
	Number, Name                  string
	InfoURL                       string
	PhotoCount                    int
	SettingWorker, ShippingWorker string
	SuppliedMaterials             string
	SuppliedMaterialsColor        string
}

type InspectionView struct {
	Kind, Label, Form string
	Step              int
	State             Stage
	Locked            bool
}
type ProjectView struct {
	RejectionMessage string
	Project
	Board, DesignerCallURL                                                    string
	Completed, Active, CompletedToday, HasSpecialInstructions, LegacyComplete bool
	Inspections                                                               []InspectionView
}

func (a *App) projectView(p Project) ProjectView {
	p.SuppliedMaterials = firstMaterialLine(p.SuppliedMaterials)
	for _, d := range a.state.Designers {
		if p.DesignerID == d.ID {
			p.DesignerName = d.Name
			p.DesignerPhone = d.Phone
		}
	}
	for _, c := range a.state.ClientNotes {
		if p.ClientID == c.ID {
			p.SpecialInstructions = c.Text
			sourcePhotos := c.Photos
			if p.ClientPhotosSelected {
				sourcePhotos = nil
				for _, ph := range c.Photos {
					for _, id := range p.ClientPhotoIDs {
						if ph.ID == id {
							sourcePhotos = append(sourcePhotos, ph)
						}
					}
				}
			}
			p.Photos = append(append([]Photo{}, sourcePhotos...), p.Photos...)
		}
	}
	v := ProjectView{Project: p, Board: a.state.Board, Completed: p.Shipping.Completed != nil || p.LegacyCompleted, Active: !p.Hidden && p.Shipping.Completed == nil && !p.LegacyCompleted, LegacyComplete: p.LegacyCompleted, HasSpecialInstructions: p.SpecialInstructions != "" || p.AdditionalInstructions != "" || len(p.Photos) > 0}
	if p.Shipping.Completed != nil {
		v.CompletedToday = day(*p.Shipping.Completed) == day(time.Now())
	}
	if p.DesignerPhone != "" {
		v.DesignerCallURL = "tel:" + p.DesignerPhone
	}
	if p.RejectedInspection == "setting" || p.RejectedInspection == "shipping" || p.RejectedInspection == "combined" {
		v.RejectionMessage = stageLabel(p.RejectedInspection) + " 신청이 반려되었습니다. 확인 후 다시 신청해 주세요."
	}
	v.Inspections = []InspectionView{{"setting", "세팅검사", p.SettingForm, 1, p.Setting, false}, {"shipping", "출하검사", p.ShippingForm, 2, p.Shipping, p.Setting.Completed == nil}}
	return v
}
func (a *App) view(csrf string) map[string]any {
	projects := []ProjectView{}
	shipments := []ShipmentSummary{}
	requests := []Event{}
	events := []Event{}
	unread := 0
	var through int64
	for _, p := range a.state.Projects {
		v := a.projectView(p)
		projects = append(projects, v)
		if v.CompletedToday {
			infoURL := ""
			if v.HasSpecialInstructions {
				infoURL = "/project-info/" + a.state.Board + "/" + fmtID(v.ID)
			}
			shipments = append(shipments, ShipmentSummary{ProjectID: v.ID, Number: v.Number, Name: v.Name, InfoURL: infoURL, PhotoCount: len(v.Photos), SettingWorker: v.Setting.Worker, ShippingWorker: v.Shipping.Worker, SuppliedMaterials: v.SuppliedMaterials, SuppliedMaterialsColor: v.SuppliedMaterialsColor})
		}
	}
	sort.SliceStable(projects, func(i, j int) bool {
		a, b := projects[i], projects[j]
		if a.ShippingDueDate != b.ShippingDueDate {
			if a.ShippingDueDate == "" {
				return false
			}
			if b.ShippingDueDate == "" {
				return true
			}
			return a.ShippingDueDate < b.ShippingDueDate
		}
		return a.Number < b.Number
	})
	for i := len(a.state.Events) - 1; i >= 0; i-- {
		e := a.state.Events[i]
		if !notifyEvent(e) {
			e.Unread = false
		}
		if e.Rejected {
			e.InspectionName = stageLabel(e.Inspection) + " 반려"
		}
		if p := a.state.project(e.ProjectID); p != nil {
			e.Order = p.Number
			e.Project = p.Name
			e.Active = !p.Hidden && p.Shipping.Completed == nil
		}
		if e.Unread {
			unread++
			if e.ID > through {
				through = e.ID
			}
		}
		if len(events) < 100 {
			events = append(events, e)
		}
		if e.Pending && (e.BatchID == 0 || e.BatchID == e.ID) {
			requests = append(requests, e)
		}
	}
	sort.Slice(requests, func(i, j int) bool { return requests[i].ID < requests[j].ID })
	reportPath := filepath.Join(a.dir, "검사완료기록-복구서버.xlsx")
	lastSync := ""
	if st, err := os.Stat(reportPath); err == nil {
		lastSync = st.ModTime().In(kst).Format("2006-01-02 15:04:05")
	}
	c := a.state.Integration.Connection
	return map[string]any{"Report": map[string]any{"Path": reportPath, "LastSync": lastSync, "Error": a.reportError}, "CSRF": csrf, "BusinessDate": day(time.Now()), "ScheduleHint": "복구 서버 · 수동 납기일 기준", "Projects": projects, "TodayShipments": shipments, "Requests": requests, "Events": events, "UnreadCount": unread, "UnreadThrough": through, "Designers": a.state.Designers, "ClientNotes": a.state.ClientNotes, "WorkerURL": "/work/" + a.state.Board, "MobileAdminEnabled": a.state.Integration.MobileEnabled, "PublicURL": c.PublicURL, "ConnectionOnline": c.PublicURL != "", "PagesMode": c.Provider == "pages", "EntryGate": false, "SelectedDesigner": "", "SelectedClient": ""}
}
