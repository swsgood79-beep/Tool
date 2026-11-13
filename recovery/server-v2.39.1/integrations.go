package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/SherClockHolmes/webpush-go"
	"github.com/go-webauthn/webauthn/webauthn"
	"io"
	"net/http"
	"time"
)

const pagesOrigin = "https://swsgood79-beep.github.io"
const repository = "swsgood79-beep/Tool"

type IntegrationConfig struct {
	Connection         ConnectionConfig
	Phone              PhoneData
	Market             MarketConfig
	Schedule           ScheduleConfig
	Backup             BackupConfig
	Passkeys           PasskeyData
	MobileEnabled      bool
	DesktopSpeechMuted bool
	Revision           int64
}
type ConnectionConfig struct{ Provider, Domain, AuthToken, GitHubToken, PublicURL string }
type PhoneData struct {
	Version               int
	PublicKey, PrivateKey string
	Subscription          *webpush.Subscription
	Delivered, Generation int64
}
type MarketConfig struct {
	Enabled                bool
	ClientID, ClientSecret string
	Revision               int64
}
type ScheduleConfig struct {
	Enabled         bool
	Mode, URL, Path string
	Revision        int64
}
type RecoveryKey struct {
	Version                     int
	Repository, BoxID, Identity string
}
type BackupConfig struct {
	Key                                            RecoveryKey
	Enabled                                        bool
	LastSuccess, LastSnapshot, LastHash, LastAsset string
	LastSize                                       int64
}
type PasskeyData struct {
	UserID      string
	Credentials []StoredCredential
}
type StoredCredential struct {
	Credential webauthn.Credential
	Created    string
}
type PasskeyFlow struct {
	Mode, Owner, Next string
	Revision          int64
	Session           *webauthn.SessionData
	Expires           time.Time
}
type Handoff struct {
	Owner, Next string
	Expires     time.Time
}
type IntegrationRuntime struct {
	Minutes          map[string]MinuteQuote
	Flows            map[string]PasskeyFlow
	Starts           map[string]PasskeyFlow
	Handoffs         map[string]Handoff
	Pair             string
	PairExpiry       time.Time
	PhoneStatus      string
	MarketResult     map[string]any
	MarketStatus     string
	MarketChecked    time.Time
	AccessToken      string
	TokenUntil       time.Time
	Bases            map[string]float64
	BaseDate         string
	ScheduleStatus   map[string]any
	BackupStatus     string
	BackupBusy       bool
	CancelTunnel     context.CancelFunc
	TunnelStatus     string
	TunnelSession    string
	NativeSent       map[int64]bool
	MarketNext       time.Time
	FXCache          map[string]any
	FXUntil          time.Time
	ScheduleSequence int64
}

func (a *App) initIntegration() {
	a.epoch++
	a.integration = IntegrationRuntime{Minutes: map[string]MinuteQuote{}, NativeSent: map[int64]bool{}, Flows: map[string]PasskeyFlow{}, Starts: map[string]PasskeyFlow{}, Handoffs: map[string]Handoff{}, Bases: map[string]float64{}, ScheduleStatus: map[string]any{}, MarketResult: map[string]any{"enabled": false, "quotes": []any{}}}
	if a.client == nil {
		a.client = &http.Client{Timeout: 20 * time.Second}
	}
	a.integration.PhoneStatus = "휴대폰 미연결"
	if a.state.Integration.Phone.Subscription != nil {
		a.integration.PhoneStatus = "저장된 휴대폰 연결"
	}
}
func (a *App) httpClient() *http.Client {
	if a.client != nil {
		return a.client
	}
	return &http.Client{Timeout: 20 * time.Second}
}
func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	d := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	if err := d.Decode(v); err != nil {
		return err
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		return errors.New("여러 JSON 값은 허용되지 않습니다")
	}
	return nil
}
func (a *App) startPumps(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.pumpNotifications(ctx)
			}
		}
	}()
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		var sched, backup time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.refreshMarkets(ctx)
				if time.Since(sched) >= 5*time.Minute {
					a.syncSchedule(ctx, false)
					sched = time.Now()
				}
				if time.Since(backup) >= time.Minute {
					a.maybeBackup(ctx)
					backup = time.Now()
				}
			}
		}
	}()
}
