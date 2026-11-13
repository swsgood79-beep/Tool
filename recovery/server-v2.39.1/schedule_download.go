package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type scheduleDownloadCandidate struct{ label, url string }

func scheduleDownloadCandidates(raw string) ([]scheduleDownloadCandidate, error) {
	raw = strings.TrimSpace(html.UnescapeString(raw))
	if len(raw) > 16384 || !scheduleURLAllowed(raw) {
		return nil, errors.New("허용된 OneDrive 공유 주소가 아닙니다")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errors.New("공유 주소의 형식을 확인하세요")
	}
	u.Fragment = ""
	urls := []string{u.String()}
	// Modern OneDrive viewer links can carry the original short sharing URL in redeem.
	if token := u.Query().Get("redeem"); token != "" {
		token = strings.TrimPrefix(token, "u!")
		for _, enc := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.StdEncoding} {
			b, e := enc.DecodeString(token)
			if e != nil {
				continue
			}
			share := strings.TrimSpace(string(b))
			v, e := url.Parse(share)
			if e == nil && scheduleURLAllowed(share) && (strings.EqualFold(v.Hostname(), "1drv.ms") || strings.EqualFold(v.Hostname(), "onedrive.live.com")) {
				v.Fragment = ""
				if v.String() != urls[0] {
					urls = append(urls, v.String())
				}
			}
			break
		}
	}
	out := []scheduleDownloadCandidate{}
	seen := map[string]bool{}
	add := func(label, endpoint string) {
		if !seen[endpoint] {
			seen[endpoint] = true
			out = append(out, scheduleDownloadCandidate{label, endpoint})
		}
	}
	for _, share := range urls {
		v, _ := url.Parse(share)
		q := v.Query()
		q.Set("download", "1")
		v.RawQuery = q.Encode()
		add("공유 주소 직접 다운로드", v.String())
	}
	// Prefer the embedded original short URL for the shares API, then retain the original fallback.
	for i := len(urls) - 1; i >= 0; i-- {
		add("OneDrive 파일 API", "https://api.onedrive.com/v1.0/shares/u!"+base64.RawURLEncoding.EncodeToString([]byte(urls[i]))+"/root/content")
	}
	return out, nil
}
func scheduleWorkbookBytes(b []byte) bool {
	z, e := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if e != nil {
		return false
	}
	for _, f := range z.File {
		if f.Name == "xl/workbook.xml" {
			return true
		}
	}
	return false
}
func (a *App) downloadSchedule(parent context.Context, raw string) ([]byte, error) {
	candidates, err := scheduleDownloadCandidates(raw)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	client := *a.httpClient()
	client.Jar, _ = cookiejar.New(nil)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 8 || !scheduleURLAllowed(req.URL.String()) {
			return errors.New("허용하지 않는 다운로드 주소")
		}
		return nil
	}
	failures := []string{}
	for _, candidate := range candidates {
		if ctx.Err() != nil {
			break
		}
		req, e := http.NewRequestWithContext(ctx, "GET", candidate.url, nil)
		if e != nil {
			return nil, errors.New("공유 주소의 형식을 확인하세요")
		}
		req.Header.Set("User-Agent", "AlphaWorkComplete/"+releaseVersion)
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Pragma", "no-cache")
		req.Header.Set("Accept", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/octet-stream;q=0.9,*/*;q=0.1")
		res, e := client.Do(req)
		if e != nil {
			// Do not echo the sharing URL/auth tokens in status messages.
			failures = append(failures, candidate.label+": 연결 또는 다운로드 이동 실패")
			continue
		}
		if res.StatusCode != http.StatusOK {
			res.Body.Close()
			message := fmt.Sprintf("%s: HTTP %d", candidate.label, res.StatusCode)
			failures = append(failures, message)
			if res.StatusCode == http.StatusTooManyRequests {
				return nil, errors.New("OneDrive 요청 제한(HTTP 429)입니다. 잠시 후 지금 갱신을 눌러 주세요")
			}
			continue
		}
		b, e := io.ReadAll(io.LimitReader(res.Body, (32<<20)+1))
		res.Body.Close()
		if len(b) > 32<<20 {
			return nil, errors.New("납기 일정표가 32MB를 넘습니다")
		}
		if e != nil {
			failures = append(failures, candidate.label+": 파일 읽기 중단")
			continue
		}
		if !scheduleWorkbookBytes(b) {
			failures = append(failures, candidate.label+": 엑셀 대신 안내·로그인 페이지 또는 다른 파일 응답")
			continue
		}
		return b, nil
	}
	if ctx.Err() != nil {
		return nil, errors.New("OneDrive 다운로드가 중단되었거나 제한 시간을 넘었습니다. 기존 납기는 유지합니다")
	}
	return nil, fmt.Errorf("OneDrive 파일을 받지 못했습니다. %s. 로그인 없는 다운로드 권한을 확인하거나 PC 동기화 파일 방식을 사용하세요", strings.Join(failures, " / "))
}
