package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const cloudflaredSHA = "2493dbc168923bb3e59e5e7716f71deced92992eea9acd6e9d74f84ac7751343"

var cloudURL = regexp.MustCompile(`https://[a-z0-9]+(?:-[a-z0-9]+)*\.trycloudflare\.com`)

func (a *App) allowedRequest(r *http.Request) bool {
	if localRequest(r) {
		return true
	}
	a.mu.Lock()
	raw := a.state.Integration.Connection.PublicURL
	a.mu.Unlock()
	u, e := url.Parse(raw)
	if e != nil || raw == "" || r.Host != u.Host {
		return false
	}
	origin := r.Header.Get("Origin")
	return origin == "" || origin == raw
}
func (a *App) cloudTool(ctx context.Context) (string, error) {
	path := filepath.Join(a.dir, "cloudflared-2026.10.7.exe")
	if b, e := os.ReadFile(path); e == nil {
		if digest(b) != cloudflaredSHA {
			return "", errors.New("기존 인터넷 도구 체크섬이 다릅니다. 파일을 유지했습니다")
		}
		return path, nil
	}
	r, e := http.NewRequestWithContext(ctx, "GET", "https://github.com/cloudflare/cloudflared/releases/download/2026.10.7/cloudflared-windows-amd64.exe", nil)
	if e != nil {
		return "", e
	}
	res, e := a.httpClient().Do(r)
	if e != nil {
		return "", e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", fmt.Errorf("인터넷 도구 다운로드 HTTP %d", res.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, (128<<20)+1))
	if e != nil {
		return "", e
	}
	if digest(b) != cloudflaredSHA {
		return "", errors.New("인터넷 도구 체크섬 불일치")
	}
	return path, atomicFile(path, b)
}
func (a *App) startTunnel() {
	a.mu.Lock()
	if a.integration.CancelTunnel != nil {
		a.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.integration.CancelTunnel = cancel
	a.integration.TunnelStatus = "인터넷 연결 준비 중"
	a.integration.TunnelSession = token()
	base := a.localURL
	if base == "" {
		base = "http://127.0.0.1:18791"
	}
	a.mu.Unlock()
	go func() {
		defer func() { a.mu.Lock(); a.integration.CancelTunnel = nil; a.mu.Unlock() }()
		tool, e := a.cloudTool(ctx)
		if e != nil {
			a.mu.Lock()
			a.integration.TunnelStatus = e.Error()
			a.mu.Unlock()
			return
		}
		cmd := exec.CommandContext(ctx, tool, "tunnel", "--url", base, "--no-autoupdate", "--protocol", "http2")
		pipe, e := cmd.StderrPipe()
		if e != nil {
			return
		}
		cmd.Stdout = cmd.Stderr
		if e = cmd.Start(); e != nil {
			a.mu.Lock()
			a.integration.TunnelStatus = "연결 프로세스 시작 실패"
			a.mu.Unlock()
			return
		}
		release, err := manageChild(cmd)
		if err != nil {
			cancel()
			cmd.Wait()
			a.mu.Lock()
			a.integration.TunnelStatus = "인터넷 프로세스 관리 실패"
			a.mu.Unlock()
			return
		}
		defer release()
		scan := bufio.NewScanner(pipe)
		scan.Buffer(make([]byte, 4096), 128<<10)
		published := ""
		for scan.Scan() {
			endpoint := cloudURL.FindString(scan.Text())
			if endpoint == "" || endpoint == published || !strings.Contains(strings.TrimSuffix(strings.TrimPrefix(endpoint, "https://"), ".trycloudflare.com"), "-") {
				continue
			}
			a.mu.Lock()
			e = a.transaction(func(s *State) error { s.Integration.Connection.PublicURL = endpoint; return nil })
			a.integration.TunnelStatus = "외부 주소 확인 중"
			provider := a.state.Integration.Connection.Provider
			a.mu.Unlock()
			if e != nil {
				cancel()
				break
			}
			if e = a.verifyEndpoint(ctx, endpoint); e == nil && provider == "pages" {
				e = a.publishPointer(ctx, true)
			}
			a.mu.Lock()
			if e == nil {
				a.integration.TunnelStatus = "외부 접속 연결됨"
			} else {
				a.integration.TunnelStatus = "주소 게시 실패: " + e.Error()
			}
			a.mu.Unlock()
			published = endpoint
		}
		cmd.Wait()
		a.mu.Lock()
		a.state.Integration.Connection.PublicURL = ""
		a.integration.TunnelStatus = "인터넷 연결 종료"
		a.mu.Unlock()
		if published != "" {
			c, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			a.publishPointer(c, false)
			cancel()
		}
	}()
}
func (a *App) publishPointer(ctx context.Context, online bool) error {
	a.mu.Lock()
	c := a.state.Integration.Connection
	session := a.integration.TunnelSession
	if session == "" {
		session = token()
		a.integration.TunnelSession = session
	}
	a.mu.Unlock()
	if c.GitHubToken == "" {
		return errors.New("GitHub 연결키가 없습니다")
	}
	path := "/repos/" + repository + "/contents/qr-endpoint.json"
	b, status, e := a.github(ctx, "GET", path+"?ref=main", c.GitHubToken, nil, "")
	if e != nil {
		return e
	}
	var old struct{ SHA, Content string }
	if status == 200 {
		if e = json.Unmarshal(b, &old); e != nil {
			return e
		}
		if !online {
			decoded, e := base64.StdEncoding.DecodeString(strings.ReplaceAll(old.Content, "\n", ""))
			if e != nil {
				return e
			}
			var pointer struct{ Session string }
			if e = json.Unmarshal(decoded, &pointer); e != nil {
				return e
			}
			if pointer.Session != session {
				return nil
			}
		}
	} else if status != 404 {
		return fmt.Errorf("GitHub 주소 확인 HTTP %d", status)
	}
	endpoint := c.PublicURL
	if !online {
		endpoint = ""
	}
	pointer, _ := json.Marshal(map[string]any{"application": "wanryotalk", "version": 1, "online": online, "endpoint": endpoint, "session": session, "updated_at": time.Now().UTC().Format(time.RFC3339)})
	body, _ := json.Marshal(map[string]any{"message": "Update current PC connection", "branch": "main", "sha": old.SHA, "content": base64.StdEncoding.EncodeToString(pointer)})
	_, status, e = a.github(ctx, "PUT", path, c.GitHubToken, body, "application/json")
	if e != nil {
		return e
	}
	if status != 200 && status != 201 {
		return fmt.Errorf("GitHub 주소 게시 HTTP %d", status)
	}
	return nil
}

func (a *App) verifyEndpoint(ctx context.Context, endpoint string) error {
	a.mu.Lock()
	board, session := a.state.Board, a.integration.TunnelSession
	a.mu.Unlock()
	probe, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		r, e := http.NewRequestWithContext(probe, "GET", endpoint+"/ready/"+board, nil)
		if e != nil {
			return e
		}
		res, e := a.httpClient().Do(r)
		if e == nil {
			var v struct{ Application, Session string }
			data, err := io.ReadAll(io.LimitReader(res.Body, 2048))
			res.Body.Close()
			if err == nil && res.StatusCode == 200 && json.Unmarshal(data, &v) == nil && v.Application == "wanryotalk" && equal(v.Session, session) {
				return nil
			}
		}
		select {
		case <-probe.Done():
			return errors.New("외부 접속 검증 실패 · 주소를 게시하지 않았습니다")
		case <-time.After(time.Second):
		}
	}
}
