package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

var user32 = syscall.NewLazyDLL("user32.dll")
var gdi32 = syscall.NewLazyDLL("gdi32.dll")
var shell32 = syscall.NewLazyDLL("shell32.dll")
var kernel32 = syscall.NewLazyDLL("kernel32.dll")

//go:uintptrescapes
func win(name string, args ...uintptr) uintptr {
	r, _, _ := user32.NewProc(name).Call(args...)
	return r
}

//go:uintptrescapes
func gdi(name string, args ...uintptr) uintptr {
	r, _, _ := gdi32.NewProc(name).Call(args...)
	return r
}
func wide(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(strings.ReplaceAll(s, "\x00", ""))
	return p
}
func wp(p any) uintptr {
	switch x := p.(type) {
	case *uint16:
		return uintptr(unsafe.Pointer(x))
	case *winRect:
		return uintptr(unsafe.Pointer(x))
	}
	panic("unsupported native pointer")
}
func signed(v int32) uintptr { return uintptr(uint32(v)) }

const (
	wmPaint         = 0x000f
	wmClose         = 0x0010
	wmDestroy       = 2
	wmSize          = 5
	wmCommand       = 0x111
	wmTimer         = 0x113
	wmLButtonUp     = 0x202
	wmLButtonDouble = 0x203
	wmRButtonUp     = 0x205
	wmTray          = 0x8001
	wmNativeStop    = 0x8002
	wmNotice        = 0x8003
	wmDrawItem      = 0x2b
)

var (
	mainClass  = "AlphaWorkRecoveryTestMain"
	toastClass = "AlphaWorkRecoveryTestPopup"
	badgeClass = "AlphaWorkRecoveryTestUnread"
)

const (
	idConnect         = 101
	idDisconnect      = 102
	idSettings        = 103
	idAdmin           = 104
	idPreview         = 105
	idFolder          = 106
	idExit            = 107
	idUnread          = 108
	idHistory         = 109
	idURL             = 110
	idStatus          = 111
	idStore           = 112
	idConnectionBadge = 113
	idSpeechToggle    = 114
	idSpeechPreview   = 115
	idSpeechStatus    = 116
)

var desktopMu sync.RWMutex
var currentDesktop *winDesktop
var nativeProc = syscall.NewCallback(nativeWindowProc)

type nativePopup struct {
	Created time.Time
	Notice  desktopEvent
	Window  uintptr
	Clock   popupClock
}
type winDesktop struct {
	TaskbarMessage                                   uintptr
	App                                              *App
	Stop                                             func()
	Window, Badge, Instance, Icon, Font, Bold, Large uintptr
	Controls                                         map[int]uintptr
	Labels                                           map[int]string
	Popups                                           map[uintptr]*nativePopup
	Queue                                            []desktopEvent
	Seen                                             map[int64]bool
	Messages                                         chan desktopEvent
	Scale                                            float64
	TrayAdded                                        bool
	Done                                             chan struct{}
	Last                                             desktopView
	Closing                                          bool
	Speech                                           *inspectionAnnouncer
	SpeechReady                                      bool
}

func colorref(r, g, b byte) uintptr { return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16 }

var (
	cBlue   = colorref(23, 58, 105)
	cAccent = colorref(37, 99, 235)
	cBack   = colorref(242, 246, 250)
	cWhite  = colorref(255, 255, 255)
	cText   = colorref(22, 46, 75)
	cMuted  = colorref(100, 122, 145)
	cGold   = colorref(255, 222, 102)
)

func fill(dc uintptr, r winRect, color uintptr) {
	brush := gdi("CreateSolidBrush", color)
	win("FillRect", dc, uintptr(unsafe.Pointer(&r)), brush)
	gdi("DeleteObject", brush)
}
func drawText(dc uintptr, text string, r winRect, color, font uintptr, flags uintptr) {
	gdi("SetBkMode", dc, 1)
	gdi("SetTextColor", dc, color)
	old := gdi("SelectObject", dc, font)
	win("DrawTextW", dc, uintptr(unsafe.Pointer(wide(text))), ^uintptr(0), uintptr(unsafe.Pointer(&r)), flags)
	gdi("SelectObject", dc, old)
}
func copyWide(dst []uint16, s string) {
	clear(dst)
	v := utf16.Encode([]rune(s))
	if len(v) >= len(dst) {
		v = v[:len(dst)-1]
	}
	copy(dst, v)
}
func nativeError(message string) {
	win("MessageBoxW", 0, uintptr(unsafe.Pointer(wide(message))), uintptr(unsafe.Pointer(wide("알파 프로그램 시작 확인"))), 0x10)
}
func lockOriginalApp() (func(), error) {
	mutex, _ := desktopIdentity(desktopIsTest)
	h, _, e := kernel32.NewProc("CreateMutexW").Call(0, 0, uintptr(unsafe.Pointer(wide(mutex))))
	if h == 0 {
		return nil, fmt.Errorf("프로그램 단일 실행 확인 실패: %v", e)
	}
	if e == syscall.Errno(183) {
		syscall.CloseHandle(syscall.Handle(h))
		existing := win("FindWindowW", uintptr(unsafe.Pointer(wide(mainClass))), 0)
		if existing != 0 {
			win("ShowWindow", existing, 9)
			win("SetForegroundWindow", existing)
		}
		if desktopIsTest {
			return nil, errors.New("테스트판이 이미 실행 중입니다")
		}
		return nil, errors.New("기존 알파 프로그램 또는 운영판이 실행 중입니다. 트레이에서 완전히 종료한 뒤 운영판을 실행하세요")
	}
	return func() { syscall.CloseHandle(syscall.Handle(h)) }, nil
}
func makeNativeIcon(instance uintptr) uintptr {
	pixels := make([]byte, 32*32*4)
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			i := (y*32 + x) * 4
			pixels[i] = 235
			pixels[i+1] = 99
			pixels[i+2] = 37
			pixels[i+3] = 255
		}
	}
	glyph := []byte{0x00, 0x39, 0x45, 0x45, 0x45, 0x4d, 0x33, 0x00}
	for y, row := range glyph {
		for x := 0; x < 8; x++ {
			if row&(1<<uint(7-x)) == 0 {
				continue
			}
			for dy := 0; dy < 3; dy++ {
				for dx := 0; dx < 3; dx++ {
					i := ((y*3+4+dy)*32 + x*3 + 4 + dx) * 4
					pixels[i] = 255
					pixels[i+1] = 255
					pixels[i+2] = 255
				}
			}
		}
	}
	mask := make([]byte, 128)
	r := win("CreateIcon", instance, 32, 32, 1, 32, uintptr(unsafe.Pointer(&mask[0])), uintptr(unsafe.Pointer(&pixels[0])))
	return r
}
func registerNativeClass(name string, instance, icon uintptr) error {
	wc := winClass{Size: uint32(unsafe.Sizeof(winClass{})), Proc: nativeProc, Instance: instance, Icon: icon, SmallIcon: icon, Cursor: win("LoadCursorW", 0, 32512), Class: wide(name)}
	r, _, e := user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))
	if r == 0 && e != syscall.Errno(1410) {
		return fmt.Errorf("Windows 창 등록 실패: %v", e)
	}
	return nil
}
func startDesktop(a *App, stop func()) (func(), bool, error) {
	ready := make(chan error, 1)
	d := &winDesktop{App: a, Stop: stop, Controls: map[int]uintptr{}, Labels: map[int]string{}, Popups: map[uintptr]*nativePopup{}, Seen: map[int64]bool{}, Messages: make(chan desktopEvent, 256), Scale: 1, Done: make(chan struct{})}
	go d.run(ready)
	if e := <-ready; e != nil {
		return nil, false, e
	}
	return func() {
		win("PostMessageW", d.Window, wmNativeStop, 0, 0)
		select {
		case <-d.Done:
		case <-time.After(3 * time.Second):
		}
	}, true, nil
}
func (d *winDesktop) run(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(d.Done)
	defer d.Stop()
	if p := user32.NewProc("SetProcessDPIAware"); p.Find() == nil {
		p.Call()
	}
	instance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	d.Instance = instance
	d.Icon = makeNativeIcon(instance)
	if d.Icon == 0 {
		d.Icon = win("LoadIconW", 0, 32512)
	}
	for _, name := range []string{mainClass, toastClass, badgeClass} {
		if e := registerNativeClass(name, instance, d.Icon); e != nil {
			ready <- e
			return
		}
	}
	desktopMu.Lock()
	currentDesktop = d
	desktopMu.Unlock()
	defer func() {
		desktopMu.Lock()
		if currentDesktop == d {
			currentDesktop = nil
		}
		desktopMu.Unlock()
	}()
	d.Window = win("CreateWindowExW", 0, uintptr(unsafe.Pointer(wide(mainClass))), uintptr(unsafe.Pointer(wide(productTitle(d.App.testMode)))), 0x02cf0000, 0x80000000, 0x80000000, 900, 800, 0, 0, instance, 0)
	if d.Window == 0 {
		ready <- errors.New("Windows 관리 창 생성 실패")
		return
	}
	if p := user32.NewProc("GetDpiForWindow"); p.Find() == nil {
		dpi, _, _ := p.Call(d.Window)
		if dpi > 0 {
			d.Scale = float64(dpi) / 96
		}
	}
	d.TaskbarMessage = win("RegisterWindowMessageW", uintptr(unsafe.Pointer(wide("TaskbarCreated"))))
	area := d.workArea()
	if fit := float64(area.Bottom-area.Top-32) / 800; fit > 0 && d.Scale > fit {
		d.Scale = fit
	}
	win("SetWindowPos", d.Window, 0, 0, 0, uintptr(d.px(900)), uintptr(d.px(800)), 0x06)
	d.Font = d.makeFont(14, 400)
	d.Bold = d.makeFont(15, 700)
	d.Large = d.makeFont(23, 700)
	defer func() {
		for _, font := range []uintptr{d.Font, d.Bold, d.Large} {
			gdi("DeleteObject", font)
		}
	}()
	d.initControls()
	d.App.mu.Lock()
	muted := !d.App.ttsConfig.Enabled || d.App.ttsConfigError != ""
	d.App.mu.Unlock()
	d.Speech = newInspectionAnnouncer(!muted, d.App.playConfiguredSpeech)
	d.Speech.mu.Lock()
	d.Speech.notice = playImmediateInspectionChime
	d.Speech.stopNotice = stopImmediateInspectionChime
	d.Speech.runConfigured = func(ctx context.Context, text string, cfg TTSConfig) error {
		_, err := runTTSPython(ctx, text, "speak", cfg)
		return err
	}
	d.Speech.mu.Unlock()
	d.App.mu.Lock()
	d.App.nativeQueue = d.Speech
	if !muted {
		d.App.integration.NativeSpeechSelected = true
		delete(d.App.integration.SpeechLeases, "local")
	}
	d.App.mu.Unlock()
	defer func() { d.Speech.close(); d.App.mu.Lock(); d.App.nativeQueue = nil; d.App.mu.Unlock() }()
	d.addTray()
	win("SetTimer", d.Window, 1, 250, 0)
	d.refresh()
	win("ShowWindow", d.Window, 5)
	win("UpdateWindow", d.Window)
	ready <- nil
	if !d.App.testMode {
		go func() { _, _ = prepareTTSRuntime() }()
	}
	var m winMessage
	for {
		r := win("GetMessageW", uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		win("TranslateMessage", uintptr(unsafe.Pointer(&m)))
		win("DispatchMessageW", uintptr(unsafe.Pointer(&m)))
	}
	if d.Icon != 0 {
		win("DestroyIcon", d.Icon)
	}
}
func (d *winDesktop) px(v int) int32 { return int32(float64(v) * d.Scale) }
func (d *winDesktop) makeFont(size, weight int) uintptr {
	return gdi("CreateFontW", signed(-d.px(size)), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(wide("맑은 고딕"))))
}
func (d *winDesktop) control(id int, kind, text string, style uintptr) {
	h := win("CreateWindowExW", 0, uintptr(unsafe.Pointer(wide(kind))), uintptr(unsafe.Pointer(wide(text))), 0x50000000|style, 0, 0, 0, 0, d.Window, uintptr(id), d.Instance, 0)
	d.Controls[id] = h
	d.Labels[id] = text
	win("SendMessageW", h, 0x30, d.Font, 1)
}
func (d *winDesktop) initControls() {
	for _, c := range []struct {
		id    int
		label string
	}{{idConnect, "인터넷 연결"}, {idDisconnect, "인터넷 끊기"}, {idSettings, "고정 QR 설정"}, {idAdmin, "수주 · 메시지 관리"}, {idPreview, "알림 미리보기"}, {idFolder, "엑셀 폴더"}, {idExit, "종료"}, {idUnread, "안 읽은 알림"}, {idSpeechToggle, "음성 알림 켜짐"}, {idSpeechPreview, "음성 설정·테스트"}} {
		d.control(c.id, "BUTTON", c.label, 0x0b)
	}
	d.control(idURL, "EDIT", "", 0x00800880)
	d.control(idHistory, "EDIT", "", 0x00a008c4)
	d.control(idStatus, "STATIC", "", 0)
	d.control(idConnectionBadge, "STATIC", "● 연결 안 됨", 0)
	d.control(idStore, "STATIC", "", 0)
	d.control(idSpeechStatus, "STATIC", "", 0)
	d.layout()
}
func (d *winDesktop) layout() {
	if len(d.Controls) == 0 {
		return
	}
	var r winRect
	win("GetClientRect", d.Window, uintptr(unsafe.Pointer(&r)))
	w := int(float64(r.Right) / d.Scale)
	h := int(float64(r.Bottom) / d.Scale)
	if w < 600 {
		w = 600
	}
	if h < 550 {
		h = 550
	}
	place := func(id, x, y, cw, ch int) {
		win("MoveWindow", d.Controls[id], signed(d.px(x)), signed(d.px(y)), signed(d.px(cw)), signed(d.px(ch)), 1)
	}
	place(idConnectionBadge, w-180, 116, 148, 23)
	place(idURL, 32, 142, w-64, 32)
	bw := (w - 80) / 3
	place(idConnect, 32, 188, bw, 38)
	place(idDisconnect, 40+bw, 188, bw, 38)
	place(idSettings, 48+bw*2, 188, bw, 38)
	place(idStore, 32, 278, w-195, 34)
	place(idFolder, w-150, 278, 118, 34)
	place(idAdmin, 32, 334, (w-76)/2, 44)
	place(idPreview, 44+(w-76)/2, 334, (w-76)/2, 44)
	place(idStatus, 32, 397, w-64, 23)
	place(idSpeechToggle, 32, 423, 170, 25)
	place(idSpeechPreview, 212, 423, 145, 25)
	place(idSpeechStatus, 367, 425, w-399, 23)
	place(idHistory, 32, 490, w-64, h-560)
	place(idUnread, 32, h-52, 190, 34)
	place(idExit, w-130, h-52, 98, 34)
	win("InvalidateRect", d.Window, 0, 1)
}
func (d *winDesktop) set(id int, text string) {
	h := d.Controls[id]
	if h == 0 || d.Labels[id] == text {
		return
	}
	d.Labels[id] = text
	win("SetWindowTextW", h, uintptr(unsafe.Pointer(wide(text))))
}
func (d *winDesktop) addTray() {
	n := winNotifyIcon{Size: uint32(unsafe.Sizeof(winNotifyIcon{})), Window: d.Window, ID: 1, Flags: 7, Callback: wmTray, Icon: d.Icon}
	copyWide(n.Tip[:], productTitle(d.App.testMode))
	r, _, _ := shell32.NewProc("Shell_NotifyIconW").Call(0, uintptr(unsafe.Pointer(&n)))
	d.TrayAdded = r != 0
}
func (d *winDesktop) trayTip(text string) {
	n := winNotifyIcon{Size: uint32(unsafe.Sizeof(winNotifyIcon{})), Window: d.Window, ID: 1, Flags: 4}
	copyWide(n.Tip[:], text)
	shell32.NewProc("Shell_NotifyIconW").Call(1, uintptr(unsafe.Pointer(&n)))
}
func (d *winDesktop) removeTray() {
	if !d.TrayAdded {
		return
	}
	n := winNotifyIcon{Size: uint32(unsafe.Sizeof(winNotifyIcon{})), Window: d.Window, ID: 1}
	shell32.NewProc("Shell_NotifyIconW").Call(2, uintptr(unsafe.Pointer(&n)))
	d.TrayAdded = false
}
func (d *winDesktop) open(path string) { openBrowser(d.App.nativeAdminURL(path)) }
func (d *winDesktop) command(id int) {
	switch id {
	case idConnect:
		go d.App.startTunnel()
	case idDisconnect:
		d.App.mu.Lock()
		cancel := d.App.integration.CancelTunnel
		d.App.mu.Unlock()
		if cancel != nil {
			cancel()
		}
	case idSettings:
		d.open("/settings")
	case idAdmin:
		d.open("/")
	case idFolder:
		if err := openDataFolder(d.App.dir); err != nil {
			nativeError("엑셀 저장 폴더를 열지 못했습니다.\n\n" + err.Error() + "\n\n저장 위치: " + d.App.dir + "\nWindows 키 + R에서 위 경로를 입력해 열 수도 있습니다.")
		}
	case idPreview:
		d.enqueue(desktopEvent{Title: "세팅·출하 통합검사 신청", Inspection: "combined", Order: "SAMPLE-001", Project: "검사 알림 미리보기 프로젝트", Worker: "홍길동", Note: "신청 내용을 확인한 뒤 검사를 진행해 주세요.", When: time.Now().In(kst).Format("01.02 15:04"), Pending: true, Target: "/inspection-requests"})
	case idSpeechToggle:
		d.App.mu.Lock()
		c := d.App.ttsConfig
		c.Enabled = !c.Enabled
		err := d.App.saveTTSConfig(c)
		d.App.mu.Unlock()
		if err != nil {
			nativeError("음성 설정 저장 실패: " + err.Error())
		}
		d.refresh()
	case idSpeechPreview:
		d.open("/tts-settings")
	case idUnread:
		d.replay()
	case idExit:
		if win("MessageBoxW", d.Window, uintptr(unsafe.Pointer(wide(exitMessage(d.App.testMode)))), uintptr(unsafe.Pointer(wide(productTitle(d.App.testMode)+" 종료"))), 0x24) == 6 {
			d.shutdown()
		}
	}
}
func (d *winDesktop) shutdown() {
	if d.Closing {
		return
	}
	d.Closing = true
	if d.Speech != nil {
		d.Speech.close()
	}
	d.removeTray()
	win("KillTimer", d.Window, 1)
	for h := range d.Popups {
		win("DestroyWindow", h)
	}
	if d.Badge != 0 {
		win("DestroyWindow", d.Badge)
		d.Badge = 0
	}
	win("DestroyWindow", d.Window)
	d.Stop()
}
func (d *winDesktop) workArea() winRect {
	var r winRect
	if win("SystemParametersInfoW", 0x30, 0, uintptr(unsafe.Pointer(&r)), 0) == 0 {
		r = winRect{0, 0, int32(win("GetSystemMetrics", 0)), int32(win("GetSystemMetrics", 1)) - 48}
	}
	return r
}
func (d *winDesktop) refresh() {
	v := d.App.desktopSnapshot()
	if d.Speech != nil {
		d.App.mu.Lock()
		muted := !d.App.ttsConfig.Enabled || d.App.ttsConfigError != ""
		webSpeech := d.App.integration.SpeechLeases["local"].Expires.After(time.Now())
		d.App.mu.Unlock()
		d.Speech.setEnabled(!muted && !webSpeech)
		label := "음성 알림 켜짐"
		if muted {
			label = "음성 알림 꺼짐"
		}
		d.set(idSpeechToggle, label)
		status := d.Speech.state()
		if webSpeech {
			status = "웹 전자녀 음성 사용 중"
		}
		d.set(idSpeechStatus, status)
		if !d.SpeechReady {
			d.Speech.prime(v.Events)
			d.SpeechReady = true
		}
	}
	d.set(idURL, v.WorkerURL)
	if v.Connected {
		d.set(idConnectionBadge, "● 연결됨")
	} else {
		d.set(idConnectionBadge, "● 연결 안 됨")
	}
	if d.Last.Connected != v.Connected {
		win("InvalidateRect", d.Controls[idConnectionBadge], 0, 1)
	}
	d.set(idStatus, v.Connection)
	d.set(idStore, v.Storage)
	d.set(idHistory, v.History)
	d.set(idUnread, fmt.Sprintf("안 읽은 알림 %d", v.Unread))
	d.trayTip(fmt.Sprintf("알파작업완료확인 · 안 읽음 %d", v.Unread))
	d.Last = v
	for _, e := range v.Events {
		if !d.Seen[e.ID] {
			d.enqueue(e)
		}
	}
	d.updateBadge(v.Unread)
	d.tickPopups()
}
func (d *winDesktop) enqueue(e desktopEvent) {
	if e.ID > 0 {
		if d.Seen[e.ID] {
			return
		}
		d.Seen[e.ID] = true
	}
	if d.Speech != nil && (e.ID == 0 || !d.App.testMode) {
		d.App.mu.Lock()
		webSpeech := d.App.integration.SpeechLeases["local"].Expires.After(time.Now())
		muted := !d.App.ttsConfig.Enabled || d.App.ttsConfigError != ""
		d.App.mu.Unlock()
		d.Speech.setEnabled(!muted && !webSpeech)
		if webSpeech || muted {
			d.Speech.prime([]desktopEvent{e})
		} else {
			d.Speech.enqueue(e)
		}
	}
	if e.ID > 0 && inspectionSpeech(e) == "" {
		syscall.NewLazyDLL("winmm.dll").NewProc("PlaySoundW").Call(uintptr(unsafe.Pointer(wide("SystemNotification"))), 0, 0x10003)
	}
	d.Queue = append(d.Queue, e)
	if len(d.Queue) > 256 {
		d.Queue = d.Queue[len(d.Queue)-256:]
	}
	d.fillPopups()
}
func (d *winDesktop) fillPopups() {
	area := d.workArea()
	capacity := max(1, min(3, int((area.Bottom-area.Top-d.px(90))/max(1, d.px(342)))))
	for len(d.Popups) < capacity && len(d.Queue) > 0 {
		e := d.Queue[0]
		d.Queue = d.Queue[1:]
		h := win("CreateWindowExW", 0x08000088, uintptr(unsafe.Pointer(wide(toastClass))), 0, 0x80000000, 0, 0, uintptr(d.px(min(420, int(float64(area.Right-area.Left)/d.Scale)-36))), uintptr(d.px(popupHeight(e))), 0, 0, d.Instance, 0)
		if h == 0 {
			continue
		}
		d.Popups[h] = &nativePopup{Notice: e, Window: h, Created: time.Now(), Clock: popupClock{Remaining: 12 * time.Second, Last: time.Now()}}
		win("ShowWindow", h, 4)
		win("UpdateWindow", h)
	}
	d.positionPopups()
}
func (d *winDesktop) positionPopups() {
	r := d.workArea()
	bottom := r.Bottom - d.px(80)
	width := d.px(min(420, int(float64(r.Right-r.Left)/d.Scale)-36))
	handles := make([]uintptr, 0, len(d.Popups))
	for h := range d.Popups {
		handles = append(handles, h)
	}
	sort.Slice(handles, func(i, j int) bool {
		return d.Popups[handles[i]].Created.Before(d.Popups[handles[j]].Created) || d.Popups[handles[i]].Created.Equal(d.Popups[handles[j]].Created) && handles[i] < handles[j]
	})
	for _, h := range handles {
		height := d.px(popupHeight(d.Popups[h].Notice))
		win("SetWindowPos", h, ^uintptr(0), signed(r.Right-width-d.px(18)), signed(bottom-height), uintptr(width), uintptr(height), 0x10)
		bottom -= height + d.px(12)
	}
}
func (d *winDesktop) dismiss(h uintptr) { delete(d.Popups, h); win("DestroyWindow", h); d.fillPopups() }
func (d *winDesktop) tickPopups() {
	var mouse winPoint
	win("GetCursorPos", uintptr(unsafe.Pointer(&mouse)))
	expired := []uintptr{}
	for h, p := range d.Popups {
		var r winRect
		win("GetWindowRect", h, uintptr(unsafe.Pointer(&r)))
		hover := mouse.X >= r.Left && mouse.X < r.Right && mouse.Y >= r.Top && mouse.Y < r.Bottom
		if p.Clock.tick(time.Now(), hover) {
			expired = append(expired, h)
		}
	}
	for _, h := range expired {
		d.dismiss(h)
	}
}
func (d *winDesktop) updateBadge(n int) {
	if n == 0 {
		if d.Badge != 0 {
			win("DestroyWindow", d.Badge)
			d.Badge = 0
		}
		return
	}
	if d.Badge == 0 {
		d.Badge = win("CreateWindowExW", 0x08000088, uintptr(unsafe.Pointer(wide(badgeClass))), 0, 0x80000000, 0, 0, 0, 0, 0, 0, d.Instance, 0)
	}
	r := d.workArea()
	win("SetWindowPos", d.Badge, ^uintptr(0), signed(r.Right-d.px(184)), signed(r.Bottom-d.px(65)), uintptr(d.px(166)), uintptr(d.px(48)), 0x50)
	win("InvalidateRect", d.Badge, 0, 1)
}
func (d *winDesktop) replay() {
	for _, e := range d.Last.Events {
		active := false
		for _, p := range d.Popups {
			if p.Notice.ID == e.ID {
				active = true
			}
		}
		for _, queued := range d.Queue {
			if queued.ID == e.ID {
				active = true
			}
		}
		if !active {
			delete(d.Seen, e.ID)
			d.enqueue(e)
		}
	}
	if len(d.Last.Events) == 0 {
		d.open("/unread")
	}
}
func (d *winDesktop) trayMenu() {
	menu := win("CreatePopupMenu")
	defer win("DestroyMenu", menu)
	for _, m := range []struct {
		id    uintptr
		label string
	}{{201, "프로그램 열기"}, {idUnread, "안 읽은 알림 다시 보기"}, {idAdmin, "관리자 화면"}, {idExit, "프로그램 종료"}} {
		win("AppendMenuW", menu, 0, m.id, uintptr(unsafe.Pointer(wide(m.label))))
	}
	var p winPoint
	win("GetCursorPos", uintptr(unsafe.Pointer(&p)))
	win("SetForegroundWindow", d.Window)
	id := win("TrackPopupMenu", menu, 0x100|0x2, signed(p.X), signed(p.Y), 0, d.Window, 0)
	if id == 201 {
		win("ShowWindow", d.Window, 9)
		win("SetForegroundWindow", d.Window)
	} else if id != 0 {
		d.command(int(id))
	}
	win("PostMessageW", d.Window, 0, 0, 0)
}
func (d *winDesktop) paintMain(dc uintptr, r winRect) {
	fill(dc, r, cBack)
	scale := d.px
	fill(dc, winRect{0, 0, r.Right, scale(96)}, cBlue)
	drawText(dc, "α  알파작업완료확인", winRect{scale(26), scale(22), r.Right - scale(20), scale(60)}, cWhite, d.Large, 0x20|4)
	drawText(dc, "검사 신청부터 완료까지, 한눈에 · v"+releaseVersion, winRect{scale(30), scale(62), r.Right, scale(86)}, colorref(183, 213, 248), d.Font, 0x20|4)
	for _, box := range []winRect{{scale(16), scale(112), r.Right - scale(16), scale(240)}, {scale(16), scale(252), r.Right - scale(16), scale(320)}, {scale(16), scale(390), r.Right - scale(16), scale(452)}, {scale(16), scale(466), r.Right - scale(16), r.Bottom - scale(65)}} {
		roundedFill(dc, box, cWhite, colorref(218, 229, 243), scale(14))
	}
	drawText(dc, "휴대폰 접속 · 고정 QR", winRect{scale(32), scale(116), r.Right, scale(140)}, cText, d.Bold, 0x20|4)
	drawText(dc, "월별 검사 기록 · 자동 저장", winRect{scale(32), scale(252), r.Right, scale(277)}, cText, d.Bold, 0x20|4)
	drawText(dc, "최근 소식 · 검사 신청 / 완료 / 비고", winRect{scale(32), scale(464), r.Right, scale(490)}, cText, d.Bold, 0x20|4)
}
func roundedFill(dc uintptr, r winRect, color, border uintptr, radius int32) {
	brush := gdi("CreateSolidBrush", color)
	pen := gdi("CreatePen", 0, 1, border)
	oldBrush := gdi("SelectObject", dc, brush)
	oldPen := gdi("SelectObject", dc, pen)
	gdi("RoundRect", dc, signed(r.Left), signed(r.Top), signed(r.Right), signed(r.Bottom), signed(radius), signed(radius))
	gdi("SelectObject", dc, oldBrush)
	gdi("SelectObject", dc, oldPen)
	gdi("DeleteObject", brush)
	gdi("DeleteObject", pen)
}
func (d *winDesktop) paintPopup(dc uintptr, r winRect, p *nativePopup) {
	e := p.Notice
	s := d.px
	accent, soft := cAccent, colorref(234, 242, 255)
	if e.Inspection == "shipping" {
		accent, soft = colorref(180, 85, 10), colorref(255, 243, 226)
	}
	if e.Inspection == "combined" {
		accent, soft = colorref(111, 67, 184), colorref(243, 236, 255)
	}
	if !e.Pending && !e.IsNote {
		accent, soft = colorref(0, 130, 89), colorref(230, 247, 239)
	}
	if e.IsNote {
		accent, soft = colorref(145, 102, 8), colorref(255, 247, 216)
	}
	fill(dc, r, cBack)
	roundedFill(dc, winRect{s(1), s(1), r.Right - s(1), r.Bottom - s(1)}, cWhite, colorref(202, 217, 238), s(18))
	roundedFill(dc, winRect{s(14), s(14), r.Right - s(14), s(53)}, soft, soft, s(10))
	drawText(dc, e.Title, winRect{s(24), s(20), r.Right - s(45), s(46)}, accent, d.Bold, 0x20|4|0x8000)
	drawText(dc, "×", winRect{r.Right - s(44), s(18), r.Right - s(18), s(46)}, cMuted, d.Bold, 0x20|1|4)
	if e.Order != "" || e.Project != "" {
		drawText(dc, "수주번호", winRect{s(24), s(69), s(94), s(94)}, cMuted, d.Font, 0x20|4)
		drawText(dc, e.Order, winRect{s(100), s(69), r.Right - s(24), s(94)}, cBlue, d.Bold, 0x20|4|0x8000)
		drawText(dc, "프로젝트", winRect{s(24), s(105), s(94), s(130)}, cMuted, d.Font, 0x20|4)
		drawText(dc, e.Project, winRect{s(100), s(105), r.Right - s(24), s(153)}, cText, d.Bold, 0x10|0x8000)
		roundedFill(dc, winRect{s(20), s(161), r.Right - s(20), s(198)}, colorref(245, 248, 253), colorref(245, 248, 253), s(8))
		drawText(dc, e.workerLabel(), winRect{s(26), s(167), s(96), s(192)}, cMuted, d.Font, 0x20|4)
		drawText(dc, e.Worker, winRect{s(100), s(167), r.Right - s(26), s(192)}, cText, d.Bold, 0x20|4|0x8000)
		if e.Note != "" {
			drawText(dc, "비고", winRect{s(24), s(207), s(94), s(232)}, cMuted, d.Font, 0x20|4)
			drawText(dc, e.Note, winRect{s(100), s(207), r.Right - s(24), s(253)}, cText, d.Font, 0x10|0x8000)
		}
	} else {
		drawText(dc, e.Body, winRect{s(24), s(68), r.Right - s(24), r.Bottom - s(65)}, cText, d.Font, 0x10|0x8000)
	}
	drawText(dc, e.When, winRect{s(24), r.Bottom - s(52), r.Right - s(172), r.Bottom - s(25)}, cMuted, d.Font, 0x20|4)
	roundedFill(dc, winRect{r.Right - s(166), r.Bottom - s(59), r.Right - s(20), r.Bottom - s(17)}, accent, accent, s(10))
	action := "내용 확인"
	if e.Pending {
		action = "검사 신청 확인"
	}
	drawText(dc, action, winRect{r.Right - s(162), r.Bottom - s(56), r.Right - s(24), r.Bottom - s(20)}, cWhite, d.Bold, 0x20|1|4)
}

func nativeWindowProc(h, m, w, l uintptr) uintptr {
	desktopMu.RLock()
	d := currentDesktop
	desktopMu.RUnlock()
	if d == nil {
		return win("DefWindowProcW", h, m, w, l)
	}
	if h == d.Window && d.TaskbarMessage != 0 && m == d.TaskbarMessage {
		d.addTray()
		return 0
	}
	if h == d.Window {
		switch m {
		case wmSize:
			d.layout()
			return 0
		case wmCommand:
			d.command(int(w & 0xffff))
			return 0
		case wmClose:
			if d.TrayAdded {
				win("ShowWindow", h, 0)
			} else {
				d.shutdown()
			}
			return 0
		case wmDestroy:
			d.removeTray()
			win("PostQuitMessage", 0)
			return 0
		case wmNativeStop:
			d.shutdown()
			return 0
		case wmNotice:
			for {
				select {
				case e := <-d.Messages:
					d.enqueue(e)
				default:
					return 0
				}
			}
		case wmTimer:
			d.refresh()
			return 0
		case wmTray:
			if l&0xffff == wmLButtonDouble || l&0xffff == wmLButtonUp {
				win("ShowWindow", h, 9)
				win("SetForegroundWindow", h)
			} else if l&0xffff == wmRButtonUp {
				d.trayMenu()
			}
			return 0
		case wmDrawItem:
			var item winDrawItem
			// LPARAM references OS-owned memory, not a Go pointer. Copy it through
			// ReadProcessMemory rather than constructing a Go pointer from an integer.
			ok, _, _ := kernel32.NewProc("ReadProcessMemory").Call(^uintptr(0), l, uintptr(unsafe.Pointer(&item)), unsafe.Sizeof(item), 0)
			if ok == 0 {
				return 0
			}
			color := colorref(233, 239, 247)
			fg := cText
			if item.CtlID == idAdmin || item.CtlID == idConnect {
				color = cAccent
				fg = cWhite
			}
			if item.CtlID == idExit {
				color = colorref(253, 236, 233)
			}
			if item.ItemState&1 != 0 {
				color = colorref(202, 219, 243)
				fg = cText
			}
			fill(item.DC, item.Rect, color)
			drawText(item.DC, d.Labels[int(item.CtlID)], item.Rect, fg, d.Bold, 0x20|1|4)
			return 1
		case 0x138:
			gdi("SetBkColor", w, cWhite)
			fg := cText
			if l == d.Controls[idConnectionBadge] {
				fg = colorref(200, 45, 50)
				if d.Last.Connected {
					fg = colorref(0, 133, 83)
				}
			}
			gdi("SetTextColor", w, fg)
			return gdi("GetStockObject", 0)
		}
	}
	switch m {
	case wmPaint:
		var ps winPaint
		dc := win("BeginPaint", h, uintptr(unsafe.Pointer(&ps)))
		var r winRect
		win("GetClientRect", h, uintptr(unsafe.Pointer(&r)))
		if h == d.Window {
			d.paintMain(dc, r)
		} else if h == d.Badge {
			fill(dc, r, cAccent)
			drawText(dc, fmt.Sprintf("안 읽음  %d", d.Last.Unread), r, cWhite, d.Bold, 0x20|1|4)
		} else if p := d.Popups[h]; p != nil {
			d.paintPopup(dc, r, p)
		}
		win("EndPaint", h, uintptr(unsafe.Pointer(&ps)))
		return 0
	case wmLButtonUp:
		if h == d.Badge {
			d.replay()
			return 0
		}
		if p := d.Popups[h]; p != nil {
			var r winRect
			win("GetClientRect", h, uintptr(unsafe.Pointer(&r)))
			x, y := int32(int16(l&0xffff)), int32(int16(l>>16))
			if x >= r.Right-d.px(38) && y < d.px(53) {
				d.dismiss(h)
			} else {
				d.open(p.Notice.Target)
				d.dismiss(h)
			}
			return 0
		}
	case wmClose:
		if h == d.Badge {
			return 0
		}
		if _, ok := d.Popups[h]; ok {
			d.dismiss(h)
			return 0
		}
	case wmRButtonUp:
		if h == d.Badge {
			d.open("/unread")
			return 0
		}
	}
	return win("DefWindowProcW", h, m, w, l)
}
func nativeNotify(title, body, target string) {
	desktopMu.RLock()
	d := currentDesktop
	desktopMu.RUnlock()
	if d == nil {
		return
	}
	u, _ := url.Parse(target)
	next := "/inspection-requests"
	var id int64
	if u != nil {
		next = u.RequestURI()
		id, _ = strconv.ParseInt(u.Query().Get("request"), 10, 64)
	}
	e := desktopEvent{ID: id, Title: title, Body: body, Target: next}
	if id > 0 {
		d.App.mu.Lock()
		for _, record := range d.App.state.Events {
			if record.ID == id {
				if project := d.App.state.project(record.ProjectID); project != nil {
					record.Order = project.Number
					record.Project = project.Name
				}
				if !notifyEvent(record) {
					d.App.mu.Unlock()
					return
				}
				e = desktopNotice(record)
				break
			}
		}
		d.App.mu.Unlock()
	}
	select {
	case d.Messages <- e:
		win("PostMessageW", d.Window, wmNotice, 0, 0)
	default:
	}
}

func configureDesktop(test bool) {
	_, mainClass = desktopIdentity(test)
	if test {
		toastClass = "AlphaWorkRecoveryTestPopup"
		badgeClass = "AlphaWorkRecoveryTestUnread"
	} else {
		toastClass = "AlphaWorkRecoveredPopup"
		badgeClass = "AlphaWorkRecoveredUnread"
	}
}

func nativeFirstStart(message string) int {
	return int(win("MessageBoxW", 0, uintptr(unsafe.Pointer(wide(message))), uintptr(unsafe.Pointer(wide("알파 프로그램 · 최초 자료 선택"))), 0x23))
}
