package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFirstStartChoiceDoesNotGuessOrOverwrite(t *testing.T) {
	home := t.TempDir()
	original := filepath.Join(home, "WanryoTalk")
	if e := os.MkdirAll(original, 0700); e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(original, "projects-v2.dat")
	if e := os.WriteFile(file, []byte("unchanged"), 0600); e != nil {
		t.Fatal(e)
	}
	for _, answer := range []int{6, 7, 2} {
		p, e, err := firstProductionChoice(home, func(string) int { return answer })
		if answer == 6 {
			if p || e || err != nil {
				t.Fatal(err)
			}
		} else if err == nil {
			t.Fatal("missing test selected")
		}
	}
	testDir := filepath.Join(home, testFolderName)
	if e := os.MkdirAll(testDir, 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(testDir, "projects-v2.dat"), []byte("test"), 0600); e != nil {
		t.Fatal(e)
	}
	p, e, err := firstProductionChoice(home, func(string) int { return 7 })
	if !p || e || err != nil {
		t.Fatal("test choice failed", err)
	}
	b, err := os.ReadFile(file)
	if err != nil || string(b) != "unchanged" {
		t.Fatal("original changed")
	}
}
func TestFirstStartEmptyNeedsExplicitChoice(t *testing.T) {
	home := t.TempDir()
	p, e, err := firstProductionChoice(home, func(string) int { return 6 })
	if p || !e || err != nil {
		t.Fatal(err)
	}
	_, _, err = firstProductionChoice(home, func(string) int { return 2 })
	if err == nil {
		t.Fatal("cancel accepted")
	}
}

func TestDesktopConnectionOnlyGreenAfterVerifiedTunnel(t *testing.T) {
	a := fixture(t)
	a.state.Integration.Connection.PublicURL = "https://fixture.trycloudflare.com"
	if a.desktopSnapshot().Connected {
		t.Fatal("persisted URL marked connected")
	}
	a.integration.CancelTunnel = func() {}
	a.integration.TunnelStatus = "외부 주소 확인 중"
	if a.desktopSnapshot().Connected {
		t.Fatal("unverified tunnel marked connected")
	}
	a.integration.TunnelStatus = "외부 접속 연결됨"
	if !a.desktopSnapshot().Connected {
		t.Fatal("verified tunnel not connected")
	}
	a.integration.TunnelStatus = "주소 게시 실패"
	if a.desktopSnapshot().Connected {
		t.Fatal("failed pointer marked connected")
	}
	a.testMode = true
	a.integration.TunnelStatus = "외부 접속 연결됨"
	if a.desktopSnapshot().Connected {
		t.Fatal("test marked connected")
	}
}
