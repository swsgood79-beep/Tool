package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const shippingExportTTL = 2 * time.Hour

type shippingExportJob struct {
	ID, Owner, Key, State, Message, Dir, SourceDir, FileName, RecordID string
	CreatedAt                                                          time.Time
	FinishedAt                                                         *time.Time
	Total, Processed, Workbooks, Warnings, Readers, MissingMetadata    int
	Bytes                                                              int64
	Photos                                                             []Photo
}

func (a *App) cleanupShippingExportsLocked(now time.Time) {
	for id, j := range a.shippingExports {
		if j.FinishedAt != nil && j.Readers == 0 && now.Sub(*j.FinishedAt) > shippingExportTTL {
			os.RemoveAll(j.Dir)
			delete(a.shippingExports, id)
		}
	}
}

// Caller holds App.mu. Snapshot metadata and pin files before starting a worker.
func (a *App) newShippingExportLocked(owner, key, recordID string) (*shippingExportJob, []ShippingRecord, bool, error) {
	if !validPhotoID.MatchString(key) {
		return nil, nil, false, errors.New("내보내기 요청을 다시 확인해 주세요.")
	}
	a.cleanupShippingExportsLocked(time.Now())
	if a.shippingExports == nil {
		a.shippingExports = map[string]*shippingExportJob{}
	}
	for _, j := range a.shippingExports {
		if j.Owner == owner && equal(j.Key, key) {
			if j.RecordID != recordID {
				return nil, nil, false, errors.New("내보내기 대상이 변경되었습니다. 다시 시도해 주세요.")
			}
			return j, nil, false, nil
		}
		if j.State == "building" {
			if j.Owner == owner && j.RecordID == recordID {
				return j, nil, false, nil
			}
			return nil, nil, false, errors.New("다른 관리자가 출하 자료를 준비 중입니다. 잠시 후 다시 시도해 주세요.")
		}
	}
	// Keep at most three completed files. An active download is never removed.
	for len(a.shippingExports) >= 3 {
		var oldest *shippingExportJob
		for _, j := range a.shippingExports {
			if j.FinishedAt != nil && j.Readers == 0 && (oldest == nil || j.CreatedAt.Before(oldest.CreatedAt)) {
				oldest = j
			}
		}
		if oldest == nil {
			return nil, nil, false, errors.New("다운로드가 진행 중입니다. 완료 후 다시 시도해 주세요.")
		}
		os.RemoveAll(oldest.Dir)
		delete(a.shippingExports, oldest.ID)
	}
	list := []ShippingRecord{}
	for _, r := range a.state.ShippingRecords {
		if r.ArchivedAt != nil && (recordID == "" || r.ID == recordID) {
			list = append(list, a.shippingCurrentNotes(&a.state, r))
		}
	}
	if len(list) == 0 {
		return nil, nil, false, errors.New("보관 완료된 출하 기록이 없습니다.")
	}
	snapshot, err := clone(State{ShippingRecords: list})
	if err != nil {
		return nil, nil, false, err
	}
	list = snapshot.ShippingRecords
	for i := range list {
		// Apply the user-defined order-name convention only to the export copy.
		// Dates and completion metadata remain archived; notes follow the linked source.
		list[i].ClientName = shippingExportClientName(list[i])
		list[i] = shippingExportMetadata(&a.state, list[i])
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].ArchivedAt.Equal(*list[j].ArchivedAt) {
			return list[i].ID < list[j].ID
		}
		return list[i].ArchivedAt.After(*list[j].ArchivedAt)
	})
	dir, err := os.MkdirTemp("", "alpha-shipping-export-")
	if err != nil {
		return nil, nil, false, err
	}
	j := &shippingExportJob{ID: token(), Owner: owner, Key: key, State: "building", Dir: dir, SourceDir: a.dir, CreatedAt: time.Now(), Total: len(list), RecordID: recordID}
	for _, record := range list {
		if strings.TrimSpace(record.ClientName) == "" || strings.TrimSpace(record.Project.ShippingDueDate) == "" {
			j.MissingMetadata++
		}
	}
	j.FileName = "출하기록_" + j.CreatedAt.In(kst).Format("20060102_1504") + ".zip"
	if recordID != "" {
		j.FileName = cleanAttachmentName(list[0].Project.Number + "_출하기록.xlsx")
	}
	if a.shippingExportPins == nil {
		a.shippingExportPins = map[string]int{}
	}
	seen := map[string]bool{}
	for _, r := range list {
		for _, p := range append(append([]Photo{}, r.Project.Photos...), r.Photos...) {
			if !seen[p.ID] {
				seen[p.ID] = true
				a.shippingExportPins[p.ID]++
				j.Photos = append(j.Photos, p)
			}
		}
	}
	a.shippingExports[j.ID] = j
	return j, list, true, nil
}
func (a *App) runShippingExport(j *shippingExportJob, records []ShippingRecord) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	var size int64
	var books, warnings int
	err := func() (err error) {
		defer func() {
			if v := recover(); v != nil {
				err = fmt.Errorf("export panic: %v", v)
			}
		}()
		size, books, warnings, err = buildShippingExport(ctx, j.Dir, records, j.CreatedAt, func(p Photo) ([]byte, error) { return shippingExportPhoto(j.SourceDir, p) }, func(done int) { a.mu.Lock(); j.Processed = done; a.mu.Unlock() }, shippingExportLimits{PhotoBytes: 24 << 20, Records: 40, Single: j.RecordID != ""})
		return err
	}()
	a.mu.Lock()
	for _, p := range j.Photos {
		a.shippingExportPins[p.ID]--
		if a.shippingExportPins[p.ID] <= 0 {
			delete(a.shippingExportPins, p.ID)
		}
	}
	a.pruneCommunityPhotos(j.Photos)
	j.Photos = nil
	now := time.Now()
	j.FinishedAt = &now
	if err != nil {
		log.Printf("shipping export failed: %v", err)
		j.State = "failed"
		j.Message = "출하 자료를 만들지 못했습니다. 저장 공간과 원본 자료를 확인한 뒤 다시 시도해 주세요."
		if errors.Is(err, context.DeadlineExceeded) {
			j.Message = "자료 준비 시간이 초과되었습니다. 잠시 후 다시 시도해 주세요."
		}
		os.RemoveAll(j.Dir)
	} else {
		j.State = "ready"
		j.Bytes = size
		j.Workbooks = books
		j.Warnings = warnings
		j.Processed = j.Total
	}
	a.mu.Unlock()
	time.AfterFunc(shippingExportTTL+time.Second, func() { a.mu.Lock(); a.cleanupShippingExportsLocked(time.Now()); a.mu.Unlock() })
}
func shippingExportStatus(j *shippingExportJob) map[string]any {
	return map[string]any{"id": j.ID, "state": j.State, "total": j.Total, "processed": j.Processed, "bytes": j.Bytes, "workbooks": j.Workbooks, "warnings": j.Warnings, "missing_metadata": j.MissingMetadata, "message": j.Message, "filename": j.FileName}
}

// Runs before the main router's long-held lock. Neither ZIP building nor a slow
// client downloading the finished file blocks inspections or notification pumps.
func (a *App) shippingExportHTTP(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/shipping-archive-admin/export" {
		return false
	}
	if r.Method == "POST" {
		r.Body = http.MaxBytesReader(w, r.Body, 8192)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "내보내기 요청을 확인해 주세요.", 400)
			return true
		}
	}
	a.mu.Lock()
	session, ok := a.admin(w, r)
	if !ok {
		a.mu.Unlock()
		return true
	}
	if r.Method == "POST" {
		job, records, start, err := a.newShippingExportLocked(session.AccountID, r.FormValue("export_key"), r.FormValue("record"))
		if err != nil {
			a.mu.Unlock()
			http.Error(w, err.Error(), 400)
			return true
		}
		result := shippingExportStatus(job)
		a.mu.Unlock()
		if start {
			go a.runShippingExport(job, records)
		}
		jsonReply(w, result)
		return true
	}
	if r.Method != "GET" {
		a.mu.Unlock()
		http.Error(w, "지원하지 않는 요청입니다.", 405)
		return true
	}
	a.cleanupShippingExportsLocked(time.Now())
	job := a.shippingExports[r.URL.Query().Get("job")]
	if job == nil || job.Owner != session.AccountID {
		a.mu.Unlock()
		http.NotFound(w, r)
		return true
	}
	if r.URL.Query().Get("download") != "1" {
		result := shippingExportStatus(job)
		a.mu.Unlock()
		jsonReply(w, result)
		return true
	}
	if job.State != "ready" {
		a.mu.Unlock()
		http.Error(w, "출하 자료가 아직 준비되지 않았습니다.", 409)
		return true
	}
	target := "export.zip"
	contentType := "application/zip"
	if job.RecordID != "" {
		target = "export.xlsx"
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	f, err := os.Open(filepath.Join(job.Dir, target))
	if err != nil {
		a.mu.Unlock()
		http.Error(w, "준비된 파일이 만료되었습니다. 다시 내보내기 해 주세요.", 410)
		return true
	}
	job.Readers++
	name, created := job.FileName, job.CreatedAt
	a.mu.Unlock()
	defer func() {
		f.Close()
		a.mu.Lock()
		job.Readers--
		a.cleanupShippingExportsLocked(time.Now())
		a.mu.Unlock()
	}()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+target+`"; filename*=UTF-8''`+url.PathEscape(name))
	w.Header().Set("Cache-Control", "no-store")
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(30 * time.Minute))
	http.ServeContent(w, r, name, created, f)
	return true
}
func shippingExportPhoto(dir string, p Photo) ([]byte, error) {
	if !validPhotoID.MatchString(p.ID) {
		return nil, errors.New("invalid photo")
	}
	path := filepath.Join(dir, "photos", p.ID)
	if p.Legacy {
		path = filepath.Join(dir, "project-photos", p.ID+".dat")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (9<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 9<<20 {
		return nil, errors.New("photo too large")
	}
	return unprotect(data)
}
