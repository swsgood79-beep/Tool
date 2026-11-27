package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func machiningAllowedPath(path string) bool {
	switch path {
	case "/machining", "/machining/status", "/machining/action", "/my-password", "/login", "/mobile-admin/login", "/logout", "/mobile-admin/logout":
		return true
	}
	for _, p := range []string{"/native-admin/bootstrap", "/native-admin/register", "/native-admin/test", "/native-admin/device-disable", "/native-admin/token-refresh", "/native-admin/session/enroll", "/native-admin/session/restore", "/native-admin/session/revoke"} {
		if p == path {
			return true
		}
	}
	return false
}
func (a *App) machiningGate(w http.ResponseWriter, r *http.Request) bool {
	s, ok := a.session(r)
	if !ok || !machiningSession(&a.state, s) || machiningAllowedPath(r.URL.Path) {
		return false
	}
	if r.Method == "GET" && (r.URL.Path == "/" || r.URL.Path == "/mobile-admin" || r.URL.Path == "/native-admin/view") {
		http.Redirect(w, r, "/machining", 303)
	} else {
		http.Error(w, "가공팀 계정은 가공 업무만 이용할 수 있습니다", 403)
	}
	return true
}
func (a *App) machiningHTTP(w http.ResponseWriter, r *http.Request, s Session) bool {
	path := r.URL.Path
	if path != "/machining" && !strings.HasPrefix(path, "/machining/") && path != "/machining-admin" && !strings.HasPrefix(path, "/machining-admin/") {
		return false
	}
	if a.erpDBSyncHTTP(w, r, s) {
		return true
	}
	if a.erpDBHTTP(w, r, s) {
		return true
	}
	machine := machiningSession(&a.state, s)
	if purchasingSession(&a.state, s) || (machine && strings.HasPrefix(path, "/machining-admin")) {
		http.Error(w, "관리자 권한이 필요합니다", 403)
		return true
	}
	if path == "/machining/status" && r.Method == "GET" {
		jsonReply(w, map[string]any{"jobs": machiningSorted(&a.state), "unreceived": machiningUnreceived(&a.state), "last_import": machiningLastImport(&a.state), "bridge_status": func() string {
			if machine {
				return ""
			}
			return a.machiningBridgeStatus()
		}()})
		return true
	}
	if r.Method == "POST" {
		var err error
		switch path {
		case "/machining-admin/bridge":
			if s.AccountID != "" {
				http.Error(w, "대표 관리자만 설정할 수 있습니다", 403)
				return true
			}
			if a.erpDBSyncConfig().Enabled {
				http.Error(w, "DB 자동 연결을 끈 뒤 화면 수집을 설정해 주세요", 409)
				return true
			}
			marker := filepath.Join(a.dir, "erp-bridge-enabled")
			if r.FormValue("action") == "setup" {
				if a.testMode {
					http.Error(w, "격리 테스트에서는 실제 ERP 설정을 실행하지 않습니다", 409)
					return true
				}
				if a.machiningBridgeSetup || a.machiningBridgeSetting {
					http.Error(w, "ERP 최초 설정이 이미 진행 중입니다", 409)
					return true
				}
				if err = atomicFile(marker, []byte("enabled")); err != nil {
					http.Error(w, "설정 저장 실패", 500)
					return true
				}
				a.machiningBridgeSetup = true
				http.Redirect(w, r, "/machining-admin", 303)
				return true
			}

			if r.FormValue("enabled") == "1" {
				err = atomicFile(marker, []byte("enabled"))
			} else {
				err = os.Remove(marker)
				if os.IsNotExist(err) {
					err = nil
				}
			}
			if err != nil {
				http.Error(w, "자동 연결 설정 저장 실패", 500)
			} else {
				a.machiningBridgeMessage = ""
				http.Redirect(w, r, "/machining-admin", 303)
			}
			return true
		case "/machining/action":
			err = a.actMachining(s, r.FormValue("id"), r.FormValue("phase"), r.FormValue("action"), r.FormValue("run"), number(r.FormValue("revision")), int(number(r.FormValue("quantity"))))
			if err != nil {
				w.WriteHeader(409)
				jsonReply(w, map[string]any{"ok": false, "error": err.Error()})
			} else {
				jsonReply(w, map[string]any{"ok": true})
			}
			return true
		case "/machining-admin/action":
			err = a.machiningAdminAction(s, r.FormValue("id"), r.FormValue("action"), r.FormValue("run"), number(r.FormValue("revision")), int(number(r.FormValue("quantity"))))
			if err != nil {
				w.WriteHeader(409)
				jsonReply(w, map[string]any{"ok": false, "error": err.Error()})
			} else {
				jsonReply(w, map[string]any{"ok": true})
			}
			return true
		case "/machining-admin/preview":
			file, h, e := r.FormFile("workbook")
			if e != nil {
				http.Error(w, "ERP XLSX 파일을 선택해 주세요", 400)
				return true
			}
			defer file.Close()
			data, e := io.ReadAll(io.LimitReader(file, (16<<20)+1))
			if e != nil {
				http.Error(w, "파일을 읽을 수 없습니다", 400)
				return true
			}
			rows, e := parseMachiningWorkbook(data)
			if e != nil {
				http.Error(w, e.Error(), 400)
				return true
			}
			if a.machiningDrafts == nil {
				a.machiningDrafts = map[string]MachiningDraft{}
			}
			for k, d := range a.machiningDrafts {
				if time.Since(d.Created) > 20*time.Minute {
					delete(a.machiningDrafts, k)
				}
			}
			if len(a.machiningDrafts) >= 20 {
				http.Error(w, "열려 있는 미리보기가 많습니다. 잠시 후 다시 시도해 주세요", 429)
				return true
			}
			key := token()
			d := MachiningDraft{Owner: hashToken(s.CSRF), Created: time.Now(), Filename: filepath.Base(h.Filename), Hash: digest(data), Rows: rows}
			a.machiningDrafts[key] = d
			groups := machiningGroups(rows)
			units := 0
			for _, g := range groups {
				units += g.Required
			}
			a.renderTemplate(w, templates, "machining-import.html", map[string]any{"CSRF": s.CSRF, "Draft": d, "Token": key, "Groups": groups, "Rows": len(rows), "Units": units})
			return true
		case "/machining-admin/import":
			key := r.FormValue("token")
			d, ok := a.machiningDrafts[key]
			if !ok || d.Owner != hashToken(s.CSRF) || time.Since(d.Created) > 20*time.Minute {
				http.Error(w, "미리보기가 만료됐습니다. 파일을 다시 선택해 주세요", 409)
				return true
			}
			err = a.importMachining(d, machiningActor(&a.state, s))
			if err == nil {
				delete(a.machiningDrafts, key)
				http.Redirect(w, r, "/machining-admin?saved=1", 303)
				return true
			}
		default:
			http.NotFound(w, r)
			return true
		}
		http.Error(w, err.Error(), 409)
		return true
	}
	if r.Method != "GET" {
		http.Error(w, "지원하지 않는 요청입니다", 405)
		return true
	}
	if path == "/machining-admin/history.csv" {
		a.machiningCSV(w)
		return true
	}
	if path != "/machining" && path != "/machining-admin" {
		http.NotFound(w, r)
		return true
	}
	ownerPreview := path == "/machining" && s.AccountID == "" && r.URL.Query().Get("preview") == "1"
	if path == "/machining" && !machine && !ownerPreview {
		http.Redirect(w, r, "/machining-admin", 303)
		return true
	}
	actor := machiningActor(&a.state, s)
	a.renderTemplate(w, templates, "machining.html", map[string]any{"CSRF": s.CSRF, "Name": actor.Name, "Account": actor.ID, "Admin": !machine && !ownerPreview, "Owner": s.AccountID == "", "ReadOnly": ownerPreview, "Saved": r.URL.Query().Get("saved") == "1", "BridgeEnabled": a.machiningBridgeEnabled(), "BridgeStatus": a.machiningBridgeStatus(), "NativeApp": nativeAdminAgent(r)})
	return true
}
func machiningLastImport(s *State) *MachiningImport {
	if len(s.MachiningImports) == 0 {
		return nil
	}
	return &s.MachiningImports[len(s.MachiningImports)-1]
}
func (a *App) machiningCSV(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="machining-history.csv"`)
	w.Write([]byte{239, 187, 191})
	c := csv.NewWriter(w)
	c.Write([]string{"수주번호", "프로젝트", "발주번호", "로봇품번", "가공품번", "공정", "수량", "시작 작업자", "시작 일시", "완료 작업자", "완료 일시", "ERP 입고 확인자", "ERP 입고 확인 일시"})
	stamp := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.In(kst).Format("2006-01-02 15:04:05")
	}
	for _, j := range machiningSorted(&a.state) {
		robots := []string{}
		seen := map[string]bool{}
		for _, s := range j.Sources {
			if !seen[s.Robot] {
				robots = append(robots, s.Robot)
				seen[s.Robot] = true
			}
		}
		for i, runs := range [][]MachiningRun{j.Cutting, j.MillingFirst, j.Milling} {
			for _, r := range runs {
				complete, receipt := "", ""
				if r.CompletedBy != nil {
					complete = r.CompletedBy.Name
				}
				if r.ReceiptBy != nil {
					receipt = r.ReceiptBy.Name
				}
				row := []string{j.Order, j.Project, j.Purchase, strings.Join(robots, " / "), j.Part, func() string {
					if i == 2 && machiningLegacy(&j, r.ID) {
						return "기존 가공"
					}
					return []string{"절단", "가공 1차", "가공 2차"}[i]
				}(), fmt.Sprint(r.Quantity), r.StartedBy.Name, stamp(&r.Started), complete, stamp(r.Completed), receipt, stamp(r.ReceiptAt)}
				for k, v := range row {
					if strings.HasPrefix(v, "=") || strings.HasPrefix(v, "+") || strings.HasPrefix(v, "-") || strings.HasPrefix(v, "@") || strings.HasPrefix(v, "\t") || strings.HasPrefix(v, "\r") {
						row[k] = "'" + v
					}
				}
				c.Write(row)
			}
		}
	}
	c.Flush()
}
