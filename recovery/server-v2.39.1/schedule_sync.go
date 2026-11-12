package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/xuri/excelize/v2"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func scheduleURLAllowed(raw string) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}
	h := strings.ToLower(u.Hostname())
	for _, host := range []string{"onedrive.live.com", "1drv.ms", "api.onedrive.com", "onedrive.com", "sharepoint.com", "1drv.com", "files.1drv.com", "livefilestore.com"} {
		if h == host || strings.HasSuffix(h, "."+host) {
			return true
		}
	}
	return false
}
func (a *App) readSchedule(ctx context.Context, c ScheduleConfig) ([]byte, error) {
	if c.Mode == "file" {
		f, e := os.Open(c.Path)
		if e != nil {
			return nil, e
		}
		defer f.Close()
		b, e := io.ReadAll(io.LimitReader(f, (32<<20)+1))
		if len(b) > 32<<20 {
			return nil, errors.New("일정표가 32MB를 넘습니다")
		}
		return b, e
	}
	if !scheduleURLAllowed(c.URL) {
		return nil, errors.New("허용된 OneDrive 공유 주소가 아닙니다")
	}
	endpoint := "https://api.onedrive.com/v1.0/shares/u!" + base64.RawURLEncoding.EncodeToString([]byte(c.URL)) + "/root/content"
	client := *a.httpClient()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 8 || !scheduleURLAllowed(req.URL.String()) {
			return errors.New("허용하지 않는 다운로드 주소")
		}
		return nil
	}
	r, e := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if e != nil {
		return nil, e
	}
	res, e := client.Do(r)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("일정표 HTTP %d", res.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, (32<<20)+1))
	if len(b) > 32<<20 {
		return nil, errors.New("일정표 용량 초과")
	}
	return b, e
}
func scheduleDate(s string) (date string, unknown bool, err error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "ASAP") || strings.Contains(s, "미정") || s == "-" || strings.Contains(s, "협의") {
		return "", true, nil
	}
	for _, layout := range []string{"2006-01-02", "2006/01/02", "2006.01.02", "2006.1.2", "2006-1-2", "2006/1/2", "2006-01-02 15:04:05"} {
		if t, e := time.ParseInLocation(layout, s, kst); e == nil {
			return day(t), false, nil
		}
	}
	if n, ok := numeric(s); ok && n > 30000 && n < 100000 {
		if t, e := excelize.ExcelDateToTime(n, false); e == nil {
			return t.Format("2006-01-02"), false, nil
		}
	}
	return "", false, errors.New("날짜 확인 필요")
}
func parseSchedule(data []byte) (map[string]string, map[string]bool, error) {
	book, e := excelize.OpenReader(bytes.NewReader(data))
	if e != nil {
		return nil, nil, e
	}
	defer book.Close()
	rows, e := book.GetRows("조립일정", excelize.Options{RawCellValue: true})
	if e != nil {
		return nil, nil, e
	}
	if len(rows) > 20000 {
		return nil, nil, errors.New("일정표 20,000행 제한")
	}
	head, numberCol, dateCol := -1, -1, -1
	for i, row := range rows {
		numberCol, dateCol = -1, -1
		if i > 30 {
			break
		}
		for j, s := range row {
			v := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s, " ", ""), "\n", ""))
			if v == "수주no" || v == "erp수주no" || v == "수주번호" {
				numberCol = j
			}
			if v == "발송예정일" {
				dateCol = j
			}
		}
		if numberCol >= 0 && dateCol >= 0 {
			head = i
			break
		}
	}
	if head < 0 {
		return nil, nil, errors.New("수주NO·발송예정일 열을 찾을 수 없습니다")
	}
	// Excel merged date cells apply to the covered order rows; empty cells outside
	// a merge remain explicitly unset. Never forward-fill arbitrary blank rows.
	merged, err := book.GetMergeCells("조립일정")
	if err != nil {
		return nil, nil, err
	}
	for _, merge := range merged {
		sc, sr, e1 := excelize.CellNameToCoordinates(merge.GetStartAxis())
		ec, er, e2 := excelize.CellNameToCoordinates(merge.GetEndAxis())
		if e1 != nil || e2 != nil || sr <= head+1 || er > 20000 {
			continue
		}
		value, e := book.GetCellValue("조립일정", merge.GetStartAxis(), excelize.Options{RawCellValue: true})
		if e != nil {
			return nil, nil, e
		}
		for _, col := range []int{numberCol, dateCol} {
			if col+1 < sc || col+1 > ec {
				continue
			}
			for row := sr - 1; row < er; row++ {
				for len(rows) <= row {
					rows = append(rows, []string{})
				}
				for len(rows[row]) <= col {
					rows[row] = append(rows[row], "")
				}
				if rows[row][col] == "" {
					rows[row][col] = value
				}
			}
		}
	}
	dates := map[string]string{}
	blankOrders := map[string]bool{}
	conflicts := map[string]bool{}
	for _, row := range rows[head+1:] {
		if len(row) <= numberCol {
			continue
		}
		n := strings.TrimSpace(row[numberCol])
		if n == "" {
			continue
		}
		value := ""
		if len(row) > dateCol {
			value = row[dateCol]
		}
		if strings.TrimSpace(value) == "" {
			blankOrders[n] = true
			continue
		}
		d, _, e := scheduleDate(value)
		if e != nil {
			conflicts[n] = true
			continue
		}
		if old, ok := dates[n]; ok && old != d {
			conflicts[n] = true
		} else {
			dates[n] = d
		}
	}
	for n := range blankOrders {
		if _, ok := dates[n]; !ok && !conflicts[n] {
			dates[n] = ""
		}
	}
	return dates, conflicts, nil
}
func (a *App) syncSchedule(ctx context.Context, force bool) {
	a.mu.Lock()
	c := a.state.Integration.Schedule
	epoch := a.epoch
	a.integration.ScheduleSequence++
	sequence := a.integration.ScheduleSequence
	a.mu.Unlock()
	if !c.Enabled && !force {
		return
	}
	data, e := a.readSchedule(ctx, c)
	var dates map[string]string
	var conflicts map[string]bool
	if e == nil {
		dates, conflicts, e = parseSchedule(data)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if epoch != a.epoch || sequence != a.integration.ScheduleSequence || c.Revision != a.state.Integration.Schedule.Revision {
		return
	}
	status := map[string]any{"label": "납기 확인 완료", "auto": c.Enabled, "next": time.Now().Add(5 * time.Minute).Format(time.RFC3339), "source_orders": len(dates), "conflicts": len(conflicts)}
	if e != nil {
		status["label"] = "갱신 실패 · 기존 납기 유지"
		status["error"] = e.Error()
		a.integration.ScheduleStatus = status
		return
	}
	matched, changed := 0, 0
	e = a.transaction(func(s *State) error {
		for i := range s.Projects {
			p := &s.Projects[i]
			if conflicts[p.Number] {
				continue
			}
			d, ok := dates[p.Number]
			if ok {
				matched++
			}
			if !ok && !p.ShippingDueManaged {
				continue
			}
			if p.ShippingDueDate != d {
				p.ShippingDueDate = d
				p.touch()
				changed++
			}
			p.ShippingDueManaged = ok
		}
		if changed > 0 {
			s.ScheduleApplied = time.Now().Format(time.RFC3339)
		}
		return nil
	})
	if e != nil {
		status["error"] = e.Error()
	} else {
		status["success"] = time.Now().Format(time.RFC3339)
	}
	dated, unset := 0, 0
	for _, d := range dates {
		if d == "" {
			unset++
		} else {
			dated++
		}
	}
	status["dated"] = dated
	status["unset"] = unset
	status["matched"] = matched
	status["changed"] = changed
	status["unmatched"] = len(dates) - matched
	a.integration.ScheduleStatus = status
}

func (a *App) scheduleJSON() map[string]any {
	r := a.integration.ScheduleStatus
	s := map[string]any{"Error": text(r["error"]), "Success": r["success"], "Next": r["next"]}
	for _, key := range []string{"SourceOrders", "Dated", "Unset", "Matched", "Changed", "Unmatched", "Conflicts"} {
		s[key] = 0
	}
	for from, to := range map[string]string{"source_orders": "SourceOrders", "matched": "Matched", "changed": "Changed", "unmatched": "Unmatched", "conflicts": "Conflicts", "dated": "Dated", "unset": "Unset"} {
		if v, ok := r[from]; ok {
			s[to] = v
		}
	}
	label := text(r["label"])
	if label == "" {
		label = "납기표 미연결"
	}
	return map[string]any{"status": s, "enabled": a.state.Integration.Schedule.Enabled, "applied": a.state.ScheduleApplied, "label": label}
}
