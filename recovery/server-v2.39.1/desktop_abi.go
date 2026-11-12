package main

// Win32 ABI layouts, independently checked against original x64 RTTI.
type winPoint struct{ X, Y int32 }
type winRect struct{ Left, Top, Right, Bottom int32 }
type winMessage struct {
	Window         uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             winPoint
	Private        uint32
}
type winClass struct {
	Size, Style                        uint32
	Proc                               uintptr
	ClsExtra, WndExtra                 int32
	Instance, Icon, Cursor, Background uintptr
	Menu, Class                        *uint16
	SmallIcon                          uintptr
}
type winPaint struct {
	DC              uintptr
	Erase           int32
	Paint           winRect
	Restore, Update int32
	Reserved        [32]byte
}
type winNotifyIcon struct {
	Size                uint32
	Window              uintptr
	ID, Flags, Callback uint32
	Icon                uintptr
	Tip                 [128]uint16
	State, StateMask    uint32
	Info                [256]uint16
	Version             uint32
	Title               [64]uint16
	InfoFlags           uint32
	GUID                [16]byte
	Balloon             uintptr
}
type winDrawItem struct {
	CtlType, CtlID, ItemID, ItemAction, ItemState uint32
	Window, DC                                    uintptr
	Rect                                          winRect
	Data                                          uintptr
}
