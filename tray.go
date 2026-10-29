package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var (
	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")

	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procLoadIconW        = user32.NewProc("LoadIconW")
	procCreatePopupMenu  = user32.NewProc("CreatePopupMenu")
	procAppendMenuW      = user32.NewProc("AppendMenuW")
	procTrackPopupMenu   = user32.NewProc("TrackPopupMenu")
	procDestroyMenu      = user32.NewProc("DestroyMenu")
	procSetForegroundWnd = user32.NewProc("SetForegroundWindow")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
)

const (
	NIM_ADD     = 0
	NIM_MODIFY  = 1
	NIM_DELETE  = 2
	NIF_MESSAGE = 1
	NIF_ICON    = 2
	NIF_TIP     = 4

	WM_USER         = 0x0400
	WM_TRAYICON     = WM_USER + 1
	WM_RBUTTONUP    = 0x0205
	WM_LBUTTONUP    = 0x0202
	WM_LBUTTONDBLCLK = 0x0203

	MF_STRING    = 0x00000000
	MF_SEPARATOR = 0x00000800
	MF_GRAYED    = 0x00000001
	MF_DISABLED  = 0x00000002

	TPM_RETURNCMD = 0x0100
	TPM_NONOTIFY  = 0x0080
	TPM_BOTTOMALIGN = 0x0020

	ID_TRAY_TITLE  = 2001
	ID_TRAY_STATUS = 2002
	ID_TRAY_OPEN   = 2003
	ID_TRAY_STOP   = 2004
	ID_TRAY_EXIT   = 2005
	ID_TRAY_PAIR   = 2006

	WS_POPUP = 0x80000000
)

type NOTIFYICONDATAW struct {
	cbSize           uint32
	hWnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            uintptr
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uTimeoutOrVer    uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     uintptr
}

type WNDCLASSEXW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type POINT struct {
	X, Y int32
}

type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

var (
	trayHWND uintptr
	trayNID  NOTIFYICONDATAW
	trayMu   sync.Mutex
	trayInit bool
)

// openBrowser opens the mobile/web interface in the user's default browser
func openBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

// showContextMenu displays the right-click tray menu
func showContextMenu(hwnd uintptr) {
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)

	titleStr, _ := syscall.UTF16PtrFromString("⌨️ Keyboard Punch")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING|MF_DISABLED), uintptr(ID_TRAY_TITLE), uintptr(unsafe.Pointer(titleStr)))

	statusText := "🟢 Status: Ready"
	if isCurrentlyPunching() {
		statusText = "⌨️ Status: Typing..."
	}
	statusStr, _ := syscall.UTF16PtrFromString(statusText)
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING|MF_DISABLED), uintptr(ID_TRAY_STATUS), uintptr(unsafe.Pointer(statusStr)))

	sep, _ := syscall.UTF16PtrFromString("")
	procAppendMenuW.Call(hMenu, uintptr(MF_SEPARATOR), 0, uintptr(unsafe.Pointer(sep)))

	pairStr, _ := syscall.UTF16PtrFromString("📱 Pair Phone (Wi-Fi QR Code)")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_TRAY_PAIR), uintptr(unsafe.Pointer(pairStr)))

	openStr, _ := syscall.UTF16PtrFromString("🌐 Open Web UI (http://localhost:8080)")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_TRAY_OPEN), uintptr(unsafe.Pointer(openStr)))

	stopStr, _ := syscall.UTF16PtrFromString("🛑 Stop Typing")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_TRAY_STOP), uintptr(unsafe.Pointer(stopStr)))

	procAppendMenuW.Call(hMenu, uintptr(MF_SEPARATOR), 0, uintptr(unsafe.Pointer(sep)))

	exitStr, _ := syscall.UTF16PtrFromString("❌ Exit")
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_TRAY_EXIT), uintptr(unsafe.Pointer(exitStr)))

	procSetForegroundWnd.Call(hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(
		hMenu,
		uintptr(TPM_RETURNCMD|TPM_NONOTIFY|TPM_BOTTOMALIGN),
		uintptr(pt.X),
		uintptr(pt.Y),
		0,
		hwnd,
		0,
	)

	switch cmd {
	case ID_TRAY_PAIR:
		openBrowser(fmt.Sprintf("http://localhost:%d/pair.html", HTTP_PORT))
	case ID_TRAY_OPEN:
		openBrowser(fmt.Sprintf("http://localhost:%d", HTTP_PORT))
	case ID_TRAY_STOP:
		stopPunchingGlobally()
	case ID_TRAY_EXIT:
		cleanupAndExit()
	}
}

func isCurrentlyPunching() bool {
	punchMutex.Lock()
	defer punchMutex.Unlock()
	return punchCancelFunc != nil
}

func stopPunchingGlobally() {
	punchMutex.Lock()
	if punchCancelFunc != nil {
		punchCancelFunc()
		punchCancelFunc = nil
	}
	punchMutex.Unlock()
	broadcast(map[string]string{"type": "stopped"})
	logMsg("⏹", "Typing stopped from system tray", ColorYellow)
}

func cleanupAndExit() {
	trayMu.Lock()
	if trayInit {
		procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&trayNID)))
		trayInit = false
	}
	trayMu.Unlock()

	cleanupADBReverse()
	os.Exit(0)
}

// startSystemTray initializes the system tray icon and runs the Windows message loop
func startSystemTray() {
	runtime.LockOSThread()

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	className, _ := syscall.UTF16PtrFromString("KeyboardPunchGhostClass")
	hIcon, _, _ := procLoadIconW.Call(0, uintptr(IDI_APPLICATION))

	wndProc := syscall.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
		if msg == WM_TRAYICON {
			if lParam == WM_RBUTTONUP {
				showContextMenu(hwnd)
				return 0
			} else if lParam == WM_LBUTTONDBLCLK || lParam == WM_LBUTTONUP {
				openBrowser(fmt.Sprintf("http://localhost:%d", HTTP_PORT))
				return 0
			}
		}
		r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
		return r
	})

	var wc WNDCLASSEXW
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	wc.lpfnWndProc = wndProc
	wc.hInstance = hInstance
	wc.lpszClassName = className
	wc.hIcon = hIcon

	atom, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return
	}

	wndName, _ := syscall.UTF16PtrFromString("KeyboardPunchGhost")
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(wndName)),
		WS_POPUP,
		0, 0, 0, 0,
		0, 0, hInstance, 0,
	)
	if hwnd == 0 {
		return
	}

	trayHWND = hwnd
	trayNID.cbSize = uint32(unsafe.Sizeof(trayNID))
	trayNID.hWnd = hwnd
	trayNID.uID = 1001
	trayNID.uFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	trayNID.uCallbackMessage = WM_TRAYICON
	trayNID.hIcon = hIcon
	tip, _ := syscall.UTF16FromString("Keyboard Punch (Ghost Mode)")
	copy(trayNID.szTip[:], tip)

	ret, _, _ := procShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&trayNID)))
	if ret != 0 {
		trayMu.Lock()
		trayInit = true
		trayMu.Unlock()
	}

	// Windows Message Loop
	var msg MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
