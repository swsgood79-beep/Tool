package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type erpObservationKey struct{}
type erpObservationOptions struct {
	Directory, Session string
	Recovery           bool
}
type ERPObservationEvent struct {
	At       string `json:"at"`
	Name     string `json:"name"`
	Session  int    `json:"session"`
	Text     string `json:"text"`
	Duration int64  `json:"duration"`
}
type ERPObservationReport struct {
	Events    []ERPObservationEvent `json:"events"`
	Truncated bool                  `json:"truncated"`
	Dropped   int64                 `json:"dropped"`
	Cleanup   bool                  `json:"cleanup"`
	Ready     bool                  `json:"ready"`
	Reason    string                `json:"reason"`
}
type erpObservationJob struct {
	Directory, Session, Phase, Message, Backup string
	Result                                     *ERPDBReport
	stop                                       bool
	Recovery                                   bool
}

func (a *App) observationView() map[string]any {
	j := a.erpObservation
	if j == nil {
		return map[string]any{"phase": "idle", "message": "관찰 전 ERP를 실행하고 로그인해 주세요. 실제 입고는 관찰 중 표시 후 진행합니다."}
	}
	return map[string]any{"phase": j.Phase, "message": j.Message, "download": j.Result != nil && j.Backup != "", "backup": filepath.Base(j.Backup)}
}
func (a *App) erpObservationHTTP(w http.ResponseWriter, r *http.Request, s Session) bool {
	if !strings.HasPrefix(r.URL.Path, "/machining-admin/receipt-observer") {
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	if s.AccountID != "" {
		http.Error(w, "대표 관리자만 이용할 수 있습니다", 403)
		return true
	}
	switch {
	case r.Method == "GET" && r.URL.Path == "/machining-admin/receipt-observer":
		jsonReply(w, a.observationView())
	case r.Method == "GET" && r.URL.Path == "/machining-admin/receipt-observer/download":
		j := a.erpObservation
		if j == nil || j.Result == nil || j.Backup == "" {
			http.Error(w, "결과 저장 완료 후 받을 수 있습니다", 409)
			break
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="ERP-receipt-observation.json"`)
		json.NewEncoder(w).Encode(j.Result)
	case r.Method == "POST" && r.URL.Path == "/machining-admin/receipt-observer":
		action := r.FormValue("action")
		if action == "recover" {
			if a.erpObservation != nil && a.erpObservation.Result == nil {
				http.Error(w, "진행 중에는 복구할 수 없습니다", 409)
				break
			}
			data, e := os.ReadFile(filepath.Join(a.dir, "erp-observer-pending.dat"))
			if e != nil {
				http.Error(w, "복구할 관찰이 없습니다", 409)
				break
			}
			raw, e := unprotect(data)
			var v map[string]string
			if e != nil || json.Unmarshal(raw, &v) != nil {
				http.Error(w, "복구 정보를 읽지 못했습니다", 500)
				break
			}
			if !strings.HasPrefix(v["session"], "AlphaReceipt_") || !validERPFolder(v["folder"]) {
				http.Error(w, "복구 정보가 올바르지 않습니다", 500)
				break
			}
			directory, e := os.MkdirTemp(a.dir, ".erp-observer-")
			if e != nil {
				http.Error(w, "복구 준비 실패", 500)
				break
			}
			j := &erpObservationJob{Directory: directory, Session: v["session"], Phase: "recovering", Message: "이전 서버 관찰 세션을 정리합니다", Recovery: true}
			a.erpObservation = j
			go a.runObservation(j, v["folder"])
			w.WriteHeader(202)
			jsonReply(w, a.observationView())
			break
		}
		if action == "stop" {
			j := a.erpObservation
			if j == nil || j.Result != nil {
				http.Error(w, "진행 중인 관찰이 없습니다", 409)
				break
			}
			if e := os.WriteFile(filepath.Join(j.Directory, "stop"), []byte("stop"), 0600); e != nil {
				http.Error(w, "중지 요청 저장 실패", 500)
				break
			}
			j.stop = true
			j.Phase = "stopping"
			j.Message = "기록 수집을 종료하고 관찰 세션을 정리합니다"
			jsonReply(w, a.observationView())
			break
		}
		if action != "start" {
			http.Error(w, "잘못된 동작입니다", 400)
			break
		}
		if a.erpObservation != nil && a.erpObservation.Result == nil {
			http.Error(w, "이미 관찰 중입니다", 409)
			break
		}
		if a.testMode && a.erpObservationProbe == nil {
			http.Error(w, "격리 시험에서 실제 ERP를 관찰하지 않습니다", 409)
			break
		}
		folder := strings.TrimSpace(r.FormValue("folder"))
		if !validERPFolder(folder) {
			http.Error(w, "ERP 설치 폴더를 확인해 주세요", 400)
			break
		}
		// Refuse another observer after an unclean exit until its server session is recovered.
		if _, e := os.Stat(filepath.Join(a.dir, "erp-observer-pending.dat")); !os.IsNotExist(e) {
			http.Error(w, "이전 관찰의 정리 확인이 필요합니다. 관찰 결과와 복구 메모를 확인해 주세요", 409)
			break
		}
		b := make([]byte, 12)
		if _, e := rand.Read(b); e != nil {
			http.Error(w, "관찰 식별자 생성 실패", 500)
			break
		}
		directory, e := os.MkdirTemp(a.dir, ".erp-observer-")
		if e != nil {
			http.Error(w, "관찰 준비 실패", 500)
			break
		}
		j := &erpObservationJob{Directory: directory, Session: "AlphaReceipt_" + hex.EncodeToString(b), Phase: "preparing", Message: "추적 권한·ERP 실행 상태 확인 중입니다. 아직 입고하지 마세요"}
		journal, e := protect(jsonRaw(map[string]string{"session": j.Session, "folder": folder, "directory": directory}))
		if e == nil {
			check, err := unprotect(journal)
			if err != nil || !json.Valid(check) {
				e = fmt.Errorf("recovery verification failed")
			}
		}
		if e == nil {
			e = atomicFile(filepath.Join(a.dir, "erp-observer-pending.dat"), journal)
		}
		if e != nil {
			os.RemoveAll(directory)
			http.Error(w, "관찰 복구 정보 저장 실패", 500)
			break
		}
		a.erpObservation = j
		go a.runObservation(j, folder)
		w.WriteHeader(202)
		jsonReply(w, a.observationView())
	default:
		http.NotFound(w, r)
	}
	return true
}
func (a *App) runObservation(j *erpObservationJob, folder string) {
	ctx, cancel := context.WithTimeout(context.Background(), 160*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, erpObservationKey{}, erpObservationOptions{Directory: j.Directory, Session: j.Session, Recovery: j.Recovery})
	probe := a.erpObservationProbe
	if probe == nil {
		probe = platformERPDBProbe
	}
	done := make(chan ERPDBReport, 1)
	go func() { done <- probe(ctx, folder) }()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// Graceful application shutdown requests stop; allow the worker to clean up.
			if a.erpDBContext != nil && a.erpDBContext.Err() != nil {
				_ = os.WriteFile(filepath.Join(j.Directory, "stop"), []byte("stop"), 0600)
			}
			if b, e := os.ReadFile(filepath.Join(j.Directory, "phase")); e == nil && string(b) == "ready" {
				a.mu.Lock()
				if !j.stop {
					j.Phase = "recording"
					j.Message = "관찰 중 · 이제 ERP에서 실제 입고 1건을 진행한 뒤 관찰 종료를 눌러 주세요. 최대 90초 자동 종료"
				}
				a.mu.Unlock()
			}
		case result := <-done:
			result = decodeDBProbe(jsonRaw(result))
			result.Version = releaseVersion
			result.At = time.Now()
			if result.Observation == nil {
				result.Observation = &ERPObservationReport{Reason: "preparation_failed"}
			}
			raw := jsonRaw(result)
			sealed, e := protect(raw)
			backup := ""
			if e == nil {
				var check []byte
				check, e = unprotect(sealed)
				if e == nil && string(check) != string(raw) {
					e = fmt.Errorf("backup mismatch")
				}
			}
			if e == nil {
				backup = filepath.Join(a.dir, "ERP-receipt-observation-"+j.Session+".dat")
				e = atomicFile(backup, sealed)
			}
			// Clear the recovery journal only after explicit cleanup confirmation.
			if result.Observation.Cleanup {
				_ = os.Remove(filepath.Join(a.dir, "erp-observer-pending.dat"))
			}
			a.mu.Lock()
			j.Result = &result
			j.Phase = "finished"
			j.Message = "관찰 종료 · 결과를 확인해 주세요"
			if result.Observation.Reason == "recovered" {
				j.Message = "이전 관찰 세션 정리 완료 · 새 관찰을 시작할 수 있습니다"
			} else if !result.Observation.Ready {
				j.Message = "관찰 시작 실패 · ERP 실행/추적 권한을 결과에서 확인해 주세요"
			}
			if !result.Observation.Cleanup {
				j.Message += " · 서버 관찰 세션 정리 확인 필요"
			}
			if result.Observation.Truncated || result.Observation.Dropped > 0 {
				j.Message += " · 일부 기록 누락/잘림: 전체 과정 분석 불가"
			}
			if result.Observation.Reason != "captured" && result.Observation.Ready {
				j.Message += " · 수집 실패: 기록을 완전한 결과로 사용할 수 없습니다"
			}
			if result.Observation.Reason == "trace_permission_denied" {
				j.Message += " · SQL Server 관찰 권한 부족 (자동 권한 변경 없음)"
			}
			if result.Observation.Ready && len(result.Observation.Events) == 0 {
				j.Message += " · 호출 0건: 입고 성공을 의미하지 않습니다"
			}
			if e != nil {
				j.Message += " · 보호된 결과 저장 실패"
			} else {
				j.Backup = backup
			}
			a.mu.Unlock()
			if result.Observation.Cleanup {
				os.RemoveAll(j.Directory)
			}
			return
		}
	}
}
