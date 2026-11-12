package main

import (
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
const (
	mainClass  = "AlphaWorkRecoveryTestMain"
	toastClass = "AlphaWorkRecoveryTestPopup"
	badgeClass = "AlphaWorkRecoveryTestUnread"
)
const (
	idConnect    = 101
	idDisconnect = 102
	idSettings   = 103
	idAdmin      = 104
	idPreview    = 105
	idFolder     = 106
	idExit       = 107
	idUnread     = 108
	idHistory    = 109
	idURL        = 110
	idStatus     = 111
	idStore      = 112
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
	h, _, e := kernel32.NewProc("CreateMutexW").Call(0, 0, uintptr(unsafe.Pointer(wide(`Local\AlphaWorkRecoveryTest-v1`))))
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
		return nil, errors.New("테스트판이 이미 실행 중입니다. 기존 테스트판 창을 확인하세요")
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
	d.Window = win("CreateWindowExW", 0, uintptr(unsafe.Pointer(wide(mainClass))), uintptr(unsafe.Pointer(wide("알파 복구 테스트판 v1.1 · 기존 자료와 분리"))), 0x02cf0000, 0x80000000, 0x80000000, 900, 800, 0, 0, instance, 0)
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
	d.addTray()
	win("SetTimer", d.Window, 1, 250, 0)
	d.refresh()
	win("ShowWindow", d.Window, 5)
	win("UpdateWindow", d.Window)
	ready <- nil
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
	}{{idConnect, "인터넷 연결"}, {idDisconnect, "인터넷 끊기"}, {idSettings, "고정 QR 설정"}, {idAdmin, "수주 · 메시지 관리"}, {idPreview, "알림 미리보기"}, {idFolder, "엑셀 폴더"}, {idExit, "종료"}, {idUnread, "안 읽은 알림"}} {
		d.control(c.id, "BUTTON", c.label, 0x0b)
	}
	d.control(idURL, "EDIT", "", 0x00800880)
	d.control(idHistory, "EDIT", "", 0x00a008c4)
	d.control(idStatus, "STATIC", "", 0)
	d.control(idStore, "STATIC", "", 0)
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
	place(idURL, 32, 142, w-64, 32)
	bw := (w - 80) / 3
	place(idConnect, 32, 188, bw, 38)
	place(idDisconnect, 40+bw, 188, bw, 38)
	place(idSettings, 48+bw*2, 188, bw, 38)
	place(idStore, 32, 278, w-195, 34)
	place(idFolder, w-150, 278, 118, 34)
	place(idAdmin, 32, 334, (w-76)/2, 44)
	place(idPreview, 44+(w-76)/2, 334, (w-76)/2, 44)
	place(idStatus, 32, 400, w-64, 44)
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
	copyWide(n.Tip[:], "알파 복구 테스트판")
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
		openBrowser(d.App.dir)
	case idPreview:
		d.enqueue(desktopEvent{Title: "알림 미리보기", Body: "세팅·출하 통합검사 신청\n수주와 작업자 메시지를 이 창에서 확인합니다.\n메시지를 누르면 해당 검사 화면이 열립니다.", Target: "/inspection-requests"})
	case idUnread:
		d.replay()
	case idExit:
		if win("MessageBoxW", d.Window, uintptr(unsafe.Pointer(wide("테스트판을 종료하시겠습니까? 기존 알파 프로그램은 계속 실행됩니다."))), uintptr(unsafe.Pointer(wide("알파 복구 테스트판 종료"))), 0x24) == 6 {
			d.shutdown()
		}
	}
}
func (d *winDesktop) shutdown() {
	if d.Closing {
		return
	}
	d.Closing = true
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
	d.set(idURL, v.WorkerURL)
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
	if e.ID > 0 {
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
	capacity := max(1, min(3, int((area.Bottom-area.Top-d.px(90))/max(1, d.px(202)))))
	for len(d.Popups) < capacity && len(d.Queue) > 0 {
		e := d.Queue[0]
		d.Queue = d.Queue[1:]
		h := win("CreateWindowExW", 0x08000088, uintptr(unsafe.Pointer(wide(toastClass))), 0, 0x80000000, 0, 0, uintptr(d.px(370)), uintptr(d.px(190)), 0, 0, d.Instance, 0)
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
	i := 0
	handles := make([]uintptr, 0, len(d.Popups))
	for h := range d.Popups {
		handles = append(handles, h)
	}
	sort.Slice(handles, func(i, j int) bool {
		return d.Popups[handles[i]].Created.Before(d.Popups[handles[j]].Created) || d.Popups[handles[i]].Created.Equal(d.Popups[handles[j]].Created) && handles[i] < handles[j]
	})
	for _, h := range handles {
		win("SetWindowPos", h, ^uintptr(0), signed(r.Right-d.px(388)), signed(r.Bottom-d.px(80+(i+1)*202)), uintptr(d.px(370)), uintptr(d.px(190)), 0x10)
		i++
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
	drawText(dc, "α  알파 복구 테스트판", winRect{scale(26), scale(22), r.Right - scale(20), scale(60)}, cWhite, d.Large, 0x20|4)
	drawText(dc, "v1.1 TEST · 운영 자료와 분리", winRect{scale(30), scale(62), r.Right, scale(86)}, colorref(183, 213, 248), d.Font, 0x20|4)
	for _, box := range []winRect{{scale(16), scale(112), r.Right - scale(16), scale(240)}, {scale(16), scale(252), r.Right - scale(16), scale(320)}, {scale(16), scale(390), r.Right - scale(16), scale(452)}, {scale(16), scale(466), r.Right - scale(16), r.Bottom - scale(65)}} {
		fill(dc, box, cWhite)
	}
	drawText(dc, "휴대폰 접속 · 고정 QR", winRect{scale(32), scale(116), r.Right, scale(140)}, cText, d.Bold, 0x20|4)
	drawText(dc, "월별 검사 기록 · 자동 저장", winRect{scale(32), scale(252), r.Right, scale(277)}, cText, d.Bold, 0x20|4)
	drawText(dc, "완료·비고 수신 내역 · 최근 100건", winRect{scale(32), scale(464), r.Right, scale(490)}, cText, d.Bold, 0x20|4)
}
func (d *winDesktop) paintPopup(dc uintptr, r winRect, p *nativePopup) {
	fill(dc, r, cWhite)
	fill(dc, winRect{0, 0, r.Right, d.px(38)}, cBlue)
	drawText(dc, "테스트 · 검사 신청 / 작업 소식", winRect{d.px(14), d.px(5), r.Right - d.px(38), d.px(33)}, cWhite, d.Bold, 0x20|4)
	drawText(dc, "×", winRect{r.Right - d.px(34), d.px(4), r.Right - d.px(6), d.px(32)}, cWhite, d.Bold, 0x20|1|4)
	drawText(dc, p.Notice.Body, winRect{d.px(14), d.px(49), r.Right - d.px(14), r.Bottom - d.px(25)}, cText, d.Font, 0x10|0x8000)
	drawText(dc, "누르면 상세 확인 · 닫아도 안 읽은 알림 유지", winRect{d.px(14), r.Bottom - d.px(26), r.Right - d.px(10), r.Bottom - d.px(6)}, cMuted, d.Font, 0x20|4)
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
			gdi("SetTextColor", w, cText)
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
			if x >= r.Right-d.px(38) && y < d.px(38) {
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
	select {
	case d.Messages <- e:
		win("PostMessageW", d.Window, wmNotice, 0, 0)
	default:
	}
}
