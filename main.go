package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
	"unsafe"

	"github.com/atotto/clipboard"
	"github.com/gorilla/websocket"
)

// ─── Embedded Web Assets ─────────────────────────────────────

//go:embed app/*
var embeddedApp embed.FS

// ─── Constants & Ports ───────────────────────────────────────

const (
	HTTP_PORT = 8080
	WS_PORT   = 8765

	// Windows Virtual-Key Codes
	VK_BACK    = 0x08
	VK_TAB     = 0x09
	VK_RETURN  = 0x0D
	VK_CONTROL = 0x11
	VK_V       = 0x56

	// SendInput flags
	INPUT_KEYBOARD    = 1
	KEYEVENTF_KEYUP   = 0x0002
	KEYEVENTF_UNICODE = 0x0004

	IDI_APPLICATION = 32512
)

// ─── Windows Win32 API ───────────────────────────────────────

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procAttachConsole    = kernel32.NewProc("AttachConsole")
	procAllocConsole     = kernel32.NewProc("AllocConsole")

	user32    = syscall.NewLazyDLL("user32.dll")
	sendInput = user32.NewProc("SendInput")

	shell32 = syscall.NewLazyDLL("shell32.dll")

	isConsoleAttached = false
)

func initConsole() {
	for _, arg := range os.Args[1:] {
		if arg == "--console" || arg == "-c" {
			// Try attaching to parent console (e.g. if run from PowerShell/cmd)
			r, _, _ := procAttachConsole.Call(uintptr(^uintptr(0))) // ATTACH_PARENT_PROCESS = -1
			if r == 0 {
				procAllocConsole.Call()
			}
			isConsoleAttached = true
			break
		}
	}
}

// ─── Windows SendInput Structures ────────────────────────────

type keybdInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

// 40 bytes on 64-bit Windows
type input struct {
	inputType uint32
	_         [4]byte
	ki        keybdInput
	_         [8]byte
}

func sendWinInput(inputs []input) {
	if len(inputs) == 0 {
		return
	}
	sendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		uintptr(unsafe.Sizeof(inputs[0])),
	)
}

// pressKey presses and releases a key with realistic human dwell time (15-35ms)
func pressKey(vk uint16) {
	down := []input{
		{
			inputType: INPUT_KEYBOARD,
			ki: keybdInput{
				wVk:     vk,
				dwFlags: 0,
			},
		},
	}
	sendWinInput(down)

	// Realistic human finger dwell time
	time.Sleep(time.Duration(15+rand.Intn(20)) * time.Millisecond)

	up := []input{
		{
			inputType: INPUT_KEYBOARD,
			ki: keybdInput{
				wVk:     vk,
				dwFlags: KEYEVENTF_KEYUP,
			},
		},
	}
	sendWinInput(up)
}

// writeRune sends a Unicode character with human-like dwell time
func writeRune(r rune) {
	down := []input{
		{
			inputType: INPUT_KEYBOARD,
			ki: keybdInput{
				wScan:   uint16(r),
				dwFlags: KEYEVENTF_UNICODE,
			},
		},
	}
	sendWinInput(down)

	// Realistic human finger dwell time
	time.Sleep(time.Duration(15+rand.Intn(20)) * time.Millisecond)

	up := []input{
		{
			inputType: INPUT_KEYBOARD,
			ki: keybdInput{
				wScan:   uint16(r),
				dwFlags: KEYEVENTF_UNICODE | KEYEVENTF_KEYUP,
			},
		},
	}
	sendWinInput(up)
}

func pasteClipboard(text string) error {
	if err := clipboard.WriteAll(text); err != nil {
		return err
	}
	time.Sleep(30 * time.Millisecond)

	// Send Ctrl + V with natural delay
	down := []input{
		{inputType: INPUT_KEYBOARD, ki: keybdInput{wVk: VK_CONTROL, dwFlags: 0}},
		{inputType: INPUT_KEYBOARD, ki: keybdInput{wVk: VK_V, dwFlags: 0}},
	}
	sendWinInput(down)
	time.Sleep(25 * time.Millisecond)
	up := []input{
		{inputType: INPUT_KEYBOARD, ki: keybdInput{wVk: VK_V, dwFlags: KEYEVENTF_KEYUP}},
		{inputType: INPUT_KEYBOARD, ki: keybdInput{wVk: VK_CONTROL, dwFlags: KEYEVENTF_KEYUP}},
	}
	sendWinInput(up)
	return nil
}

// ─── Colors & Logging ────────────────────────────────────────

const (
	ColorPurple = "\033[95m"
	ColorBlue   = "\033[94m"
	ColorCyan   = "\033[96m"
	ColorGreen  = "\033[92m"
	ColorYellow = "\033[93m"
	ColorRed    = "\033[91m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
	ColorEnd    = "\033[0m"
)

func logMsg(icon, message, color string) {
	if isConsoleAttached {
		fmt.Printf("  %s  %s%s%s\n", icon, color, message, ColorEnd)
	}
}

func logHeader(title string) {
	if isConsoleAttached {
		sep := strings.Repeat("─", 50)
		fmt.Printf("\n%s%s%s%s\n", ColorPurple, ColorBold, sep, ColorEnd)
		fmt.Printf("  %s%s%s%s\n", ColorPurple, ColorBold, title, ColorEnd)
		fmt.Printf("%s%s%s%s\n\n", ColorPurple, ColorBold, sep, ColorEnd)
	}
}

// ─── ADB Management ──────────────────────────────────────────

func checkADB() (bool, string) {
	cmd := exec.Command("adb", "version")
	if err := cmd.Run(); err != nil {
		return false, "ADB not found on PATH.\n  Install via: winget install Google.PlatformTools (Windows)\n  Or: brew install android-platform-tools (macOS)"
	}

	out, err := exec.Command("adb", "devices").Output()
	if err != nil {
		return false, "Failed to run 'adb devices'"
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.Contains(line, "device") && !strings.Contains(line, "offline") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				return true, parts[0]
			}
		}
	}

	return false, "No Android device detected.\n  1. Connect your phone via USB cable\n  2. Enable USB Debugging (Settings → Developer Options)\n  3. Accept the USB debugging prompt on your phone"
}

func setupADBReverse() bool {
	logMsg("🔌", "Setting up ADB reverse port forwarding...", ColorCyan)
	ports := []struct {
		port int
		name string
	}{
		{WS_PORT, "WebSocket"},
		{HTTP_PORT, "HTTP"},
	}

	for _, p := range ports {
		cmd := exec.Command("adb", "reverse", fmt.Sprintf("tcp:%d", p.port), fmt.Sprintf("tcp:%d", p.port))
		if err := cmd.Run(); err != nil {
			logMsg("✕", fmt.Sprintf("%s port %d forward failed", p.name, p.port), ColorRed)
			return false
		}
		logMsg("✓", fmt.Sprintf("%s port %d forwarded", p.name, p.port), ColorGreen)
	}
	return true
}

func cleanupADBReverse() {
	_ = exec.Command("adb", "reverse", "--remove-all").Run()
}

// ─── Shared State & Concurrency ──────────────────────────────

type SharedState struct {
	Text               string `json:"text"`
	WPM                int    `json:"wpm"`
	FocusDelaySec      int    `json:"focus_delay_sec"`
	PreserveFormatting bool   `json:"preserve_formatting"`
	SmartIndent        bool   `json:"smart_indent"`
	PasteMode          bool   `json:"paste_mode"`
	JitterMode         bool   `json:"jitter_mode"`
	OopsMode           bool   `json:"oops_mode"`
}

type ClientMessage struct {
	Type               string `json:"type"`
	Text               string `json:"text,omitempty"`
	WPM                int    `json:"wpm,omitempty"`
	DelayMS            int    `json:"delay_ms,omitempty"`
	FocusDelaySec      int    `json:"focus_delay_sec,omitempty"`
	PreserveFormatting *bool  `json:"preserve_formatting,omitempty"`
	SmartIndent        *bool  `json:"smart_indent,omitempty"`
	PasteMode          *bool  `json:"paste_mode,omitempty"`
	JitterMode         *bool  `json:"jitter_mode,omitempty"`
	OopsMode           *bool  `json:"oops_mode,omitempty"`
}

var (
	stateMutex  sync.RWMutex
	sharedState = SharedState{
		Text:               "",
		WPM:                100,
		FocusDelaySec:      3,
		PreserveFormatting: true,
		SmartIndent:        true,
		PasteMode:          false,
		JitterMode:         false,
		OopsMode:           false,
	}

	clientsMutex sync.Mutex
	clients      = make(map[*websocket.Conn]bool)

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow local connections from phone/browser
		},
	}

	punchCancelFunc func()
	punchMutex      sync.Mutex
)

func broadcast(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}

	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	for conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			conn.Close()
			delete(clients, conn)
		}
	}
}

// checkMouseFailsafe returns true if mouse cursor is slammed into the top-left corner (0,0)
func checkMouseFailsafe() bool {
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return pt.X == 0 && pt.Y == 0
}

// ─── Typing Engine ───────────────────────────────────────────

func executePunch(stopChan chan struct{}, text string, delayMs int, preserve, smartIndent, paste, jitter, oops bool) {
	defer broadcast(map[string]interface{}{"type": "complete"})

	totalChars := len(text)
	if totalChars == 0 {
		return
	}

	if paste {
		select {
		case <-stopChan:
			broadcast(map[string]interface{}{"type": "stopped"})
			logMsg("■", "Pasting stopped by user", ColorYellow)
			return
		default:
		}

		if err := pasteClipboard(text); err != nil {
			broadcast(map[string]interface{}{
				"type":    "error",
				"message": fmt.Sprintf("Paste error: %v", err),
			})
			return
		}

		broadcast(map[string]interface{}{
			"type":    "progress",
			"current": totalChars,
			"total":   totalChars,
		})
		logMsg("✅", "Paste complete!", ColorGreen)
		return
	}

	// ─── Smart IDE Auto-Indent Typing ─────────────────────────
	if smartIndent && preserve {
		lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n"), "\n")
		curIDEIndent := 0
		tabSize := 4
		typedChars := 0

		for lineIdx, line := range lines {
			// Check emergency stop and mouse corner failsafe
			if checkMouseFailsafe() {
				broadcast(map[string]interface{}{"type": "stopped"})
				logMsg("🛡️", "Emergency stop triggered by mouse in corner!", ColorYellow)
				return
			}

			select {
			case <-stopChan:
				broadcast(map[string]interface{}{"type": "stopped"})
				logMsg("■", "Typing stopped by user", ColorYellow)
				return
			default:
			}

			strippedLine := strings.TrimLeft(line, " \t")
			leadingWS := line[:len(line)-len(strippedLine)]
			targetIndent := 0
			for _, ch := range leadingWS {
				if ch == '\t' {
					targetIndent += tabSize
				} else {
					targetIndent++
				}
			}

			if lineIdx == 0 {
				if len(leadingWS) > 0 {
					for _, ch := range leadingWS {
						if ch == '\t' {
							pressKey(VK_TAB)
						} else {
							writeRune(ch)
						}
					}
					curIDEIndent = targetIndent
					typedChars += len(leadingWS)
				}
			} else {
				pressKey(VK_RETURN)
				typedChars++

				// Brief pause for IDE to process auto-indentation
				time.Sleep(35 * time.Millisecond)

				prevLine := lines[lineIdx-1]
				cleanPrev := strings.TrimSpace(strings.Split(strings.Split(prevLine, "#")[0], "//")[0])
				if strings.HasSuffix(cleanPrev, ":") || strings.HasSuffix(cleanPrev, "{") {
					curIDEIndent += tabSize
				}

				if len(strippedLine) > 0 {
					if targetIndent < curIDEIndent {
						// Dedent: press Backspace
						dedents := (curIDEIndent - targetIndent) / tabSize
						rem := (curIDEIndent - targetIndent) % tabSize
						for d := 0; d < dedents; d++ {
							if jitter {
								time.Sleep(time.Duration(40+rand.Intn(40)) * time.Millisecond)
							} else {
								time.Sleep(40 * time.Millisecond)
							}
							pressKey(VK_BACK)
						}
						for r := 0; r < rem; r++ {
							time.Sleep(20 * time.Millisecond)
							pressKey(VK_BACK)
						}
						curIDEIndent = targetIndent
					} else if targetIndent > curIDEIndent {
						extra := targetIndent - curIDEIndent
						for e := 0; e < extra; e++ {
							writeRune(' ')
						}
						curIDEIndent = targetIndent
					}
					// If target == curIDEIndent: 0 spaces typed! Exactly matches IDE auto-indent!
				}

				typedChars += len(leadingWS)
				broadcast(map[string]interface{}{
					"type":    "progress",
					"current": min(totalChars, typedChars),
					"total":   totalChars,
				})
			}

			// Type characters in the line
			for _, ch := range strippedLine {
				if checkMouseFailsafe() {
					broadcast(map[string]interface{}{"type": "stopped"})
					logMsg("🛡️", "Emergency stop triggered by mouse in corner!", ColorYellow)
					return
				}

				select {
				case <-stopChan:
					broadcast(map[string]interface{}{"type": "stopped"})
					logMsg("■", "Typing stopped by user", ColorYellow)
					return
				default:
				}

				if oops && unicode.IsLetter(ch) && rand.Float64() < 0.03 {
					wrongRune := 'x'
					if unicode.ToLower(ch) == 'x' {
						wrongRune = 'z'
					}
					writeRune(wrongRune)
					time.Sleep(time.Duration(100+rand.Intn(100)) * time.Millisecond)
					pressKey(VK_BACK)
					if isConsoleAttached {
						fmt.Printf("  🙈 Oops! Made a typo on '%c', fixed it.\n", ch)
					}
					time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)
				}

				writeRune(ch)
				typedChars++

				if typedChars%5 == 0 || typedChars == totalChars {
					broadcast(map[string]interface{}{
						"type":    "progress",
						"current": min(totalChars, typedChars),
						"total":   totalChars,
					})
				}

				if delayMs > 0 {
					var sleepTime time.Duration
					if jitter {
						jittered := rand.NormFloat64()*(float64(delayMs)*0.4) + float64(delayMs)
						sleepTime = time.Duration(math.Max(1.0, jittered)) * time.Millisecond
					} else {
						sleepTime = time.Duration(delayMs) * time.Millisecond
					}
					time.Sleep(sleepTime)
				}
			}
		}
		logMsg("✅", "Typing complete!", ColorGreen)
		return
	}

	// ─── Fallback Raw Character-by-Character Typing ───────────
	for i, ch := range text {
		if checkMouseFailsafe() {
			broadcast(map[string]interface{}{"type": "stopped"})
			logMsg("🛡️", "Emergency stop triggered by mouse in corner!", ColorYellow)
			return
		}

		select {
		case <-stopChan:
			broadcast(map[string]interface{}{"type": "stopped"})
			logMsg("■", "Typing stopped by user", ColorYellow)
			return
		default:
		}

		if preserve && ch == '\n' {
			pressKey(VK_RETURN)
		} else if preserve && ch == '\t' {
			pressKey(VK_TAB)
		} else if ch == '\r' {
			continue
		} else {
			if oops && unicode.IsLetter(ch) && rand.Float64() < 0.03 {
				wrongRune := 'x'
				if unicode.ToLower(ch) == 'x' {
					wrongRune = 'z'
				}
				writeRune(wrongRune)
				time.Sleep(time.Duration(100+rand.Intn(100)) * time.Millisecond)
				pressKey(VK_BACK)
				if isConsoleAttached {
					fmt.Printf("  🙈 Oops! Made a typo on '%c', fixed it.\n", ch)
				}
				time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)
			}
			writeRune(ch)
		}

		if i%5 == 0 || i == totalChars-1 {
			broadcast(map[string]interface{}{
				"type":    "progress",
				"current": i + 1,
				"total":   totalChars,
			})
		}

		if delayMs > 0 && i < totalChars-1 {
			var sleepTime time.Duration
			if jitter {
				jittered := rand.NormFloat64()*(float64(delayMs)*0.4) + float64(delayMs)
				sleepTime = time.Duration(math.Max(1.0, jittered)) * time.Millisecond
			} else {
				sleepTime = time.Duration(delayMs) * time.Millisecond
			}
			time.Sleep(sleepTime)
		}
	}
	logMsg("✅", "Typing complete!", ColorGreen)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── WebSocket Handler ───────────────────────────────────────

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	clientIP := r.RemoteAddr
	clientsMutex.Lock()
	clients[conn] = true
	totalClients := len(clients)
	clientsMutex.Unlock()

	logMsg("📱", fmt.Sprintf("Device connected from %s (Total: %d)", clientIP, totalClients), ColorGreen)

	// Send current shared state
	stateMutex.RLock()
	initState := sharedState
	stateMutex.RUnlock()

	_ = conn.WriteJSON(map[string]interface{}{
		"type":  "sync_state",
		"state": initState,
	})

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg ClientMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "ping":
			_ = conn.WriteJSON(map[string]string{"type": "pong"})

		case "update_code":
			stateMutex.Lock()
			if msg.Text != "" || msg.Text == "" {
				sharedState.Text = msg.Text
			}
			if msg.WPM > 0 {
				sharedState.WPM = msg.WPM
			}
			if msg.FocusDelaySec >= 0 {
				sharedState.FocusDelaySec = msg.FocusDelaySec
			}
			if msg.PreserveFormatting != nil {
				sharedState.PreserveFormatting = *msg.PreserveFormatting
			}
			if msg.SmartIndent != nil {
				sharedState.SmartIndent = *msg.SmartIndent
			}
			if msg.PasteMode != nil {
				sharedState.PasteMode = *msg.PasteMode
			}
			if msg.JitterMode != nil {
				sharedState.JitterMode = *msg.JitterMode
			}
			if msg.OopsMode != nil {
				sharedState.OopsMode = *msg.OopsMode
			}
			currentState := sharedState
			stateMutex.Unlock()

			logMsg("🔄", fmt.Sprintf("Code updated (%d chars). Syncing across devices...", len(currentState.Text)), ColorCyan)
			broadcast(map[string]interface{}{
				"type":                "code_updated",
				"text":                currentState.Text,
				"wpm":                 currentState.WPM,
				"focus_delay_sec":     currentState.FocusDelaySec,
				"preserve_formatting": currentState.PreserveFormatting,
				"smart_indent":        currentState.SmartIndent,
				"paste_mode":          currentState.PasteMode,
				"jitter_mode":         currentState.JitterMode,
				"oops_mode":           currentState.OopsMode,
				"sender":              clientIP,
			})

		case "punch":
			text := msg.Text
			if text == "" {
				_ = conn.WriteJSON(map[string]string{
					"type":    "error",
					"message": "No text to type",
				})
				continue
			}

			wpm := msg.WPM
			if wpm <= 0 {
				wpm = 100
			}
			delayMs := msg.DelayMS
			if delayMs <= 0 {
				delayMs = int(60000.0 / (float64(wpm) * 5.0))
			}

			preserve := true
			if msg.PreserveFormatting != nil {
				preserve = *msg.PreserveFormatting
			}
			smartIndent := true
			if msg.SmartIndent != nil {
				smartIndent = *msg.SmartIndent
			}
			paste := false
			if msg.PasteMode != nil {
				paste = *msg.PasteMode
			}
			jitter := false
			if msg.JitterMode != nil {
				jitter = *msg.JitterMode
			}
			oops := false
			if msg.OopsMode != nil {
				oops = *msg.OopsMode
			}

			stateMutex.Lock()
			sharedState.Text = text
			sharedState.WPM = wpm
			sharedState.PreserveFormatting = preserve
			sharedState.SmartIndent = smartIndent
			sharedState.PasteMode = paste
			sharedState.JitterMode = jitter
			sharedState.OopsMode = oops
			stateMutex.Unlock()

			if paste {
				logMsg("📋", fmt.Sprintf("Pasting %d chars via clipboard", len(text)), ColorCyan)
			} else {
				var modes []string
				if smartIndent {
					modes = append(modes, "SMART_INDENT")
				}
				if jitter {
					modes = append(modes, "JITTER")
				}
				if oops {
					modes = append(modes, "OOPS")
				}
				modesStr := ""
				if len(modes) > 0 {
					modesStr = fmt.Sprintf(" [%s]", strings.Join(modes, "+"))
				}
				logMsg("⌨️ ", fmt.Sprintf("Typing %d chars at %d WPM (%dms delay)%s", len(text), wpm, delayMs, modesStr), ColorCyan)
			}

			// Stop any current punch task
			punchMutex.Lock()
			if punchCancelFunc != nil {
				punchCancelFunc()
			}
			stopChan := make(chan struct{})
			punchCancelFunc = func() {
				close(stopChan)
			}
			punchMutex.Unlock()

			go executePunch(stopChan, text, delayMs, preserve, smartIndent, paste, jitter, oops)

		case "stop":
			stopPunchingGlobally()
		}
	}

	clientsMutex.Lock()
	delete(clients, conn)
	remaining := len(clients)
	clientsMutex.Unlock()
	logMsg("📱", fmt.Sprintf("Device disconnected (%s). Remaining: %d", clientIP, remaining), ColorDim)
}

// ─── Main Entry Point ────────────────────────────────────────

func main() {
	initConsole()
	logHeader("⌨️  Keyboard Punch — Companion (Go Native)")

	// 1. Check ADB
	logMsg("🔍", "Checking ADB connection...", ColorCyan)
	ok, adbInfo := checkADB()
	if !ok {
		logMsg("⚠️ ", adbInfo, ColorYellow)
		logMsg("💡", "No USB phone detected yet. Connect phone with USB Debugging for ADB reverse.", ColorDim)
		logMsg("🌐", "Starting web & WebSocket servers for browser & local network access...", ColorCyan)
	} else {
		logMsg("✅", fmt.Sprintf("Device connected: %s", adbInfo), ColorGreen)
		if !setupADBReverse() {
			logMsg("⚠️ ", "ADB reverse setup failed. Try: adb reverse --remove-all", ColorYellow)
		}
	}

	// 2. Start HTTP Server serving embedded app/
	appFS, err := fs.Sub(embeddedApp, "app")
	if err != nil {
		fmt.Printf("Failed to load embedded web app: %v\n", err)
		return
	}
	http.Handle("/", http.FileServer(http.FS(appFS)))
	http.HandleFunc("/ws", handleWS)
	http.HandleFunc("/ws/", handleWS)

	go func() {
		addr := fmt.Sprintf("0.0.0.0:%d", HTTP_PORT)
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()
	logMsg("🌐", fmt.Sprintf("HTTP server running on port %d", HTTP_PORT), ColorGreen)

	// 3. Start Dedicated WebSocket Server on WS_PORT (for backward compatibility)
	wsMux := http.NewServeMux()
	wsMux.HandleFunc("/", handleWS)
	go func() {
		addr := fmt.Sprintf("0.0.0.0:%d", WS_PORT)
		if err := http.ListenAndServe(addr, wsMux); err != nil {
			fmt.Printf("WebSocket server error: %v\n", err)
		}
	}()
	logMsg("🔗", fmt.Sprintf("WebSocket server running on port %d", WS_PORT), ColorGreen)

	logHeader("🚀 Ready!")
	if isConsoleAttached {
		fmt.Printf("  📱  %sOpen on your phone:  %shttp://localhost:%d%s\n", ColorCyan, ColorBold, HTTP_PORT, ColorEnd)
		fmt.Printf("  💡  %sOpen Chrome on your phone and go to the URL above%s\n", ColorDim, ColorEnd)
		fmt.Printf("  🛡️  %sMouse Failsafe: Slam mouse to top-left corner to abort typing%s\n", ColorDim, ColorEnd)
		fmt.Printf("  🛑  %sPress %sCtrl+C%s to stop%s\n\n", ColorDim, ColorBold, ColorDim, ColorEnd)
	}

	// Start Windows System Tray in background goroutine
	go startSystemTray()

	// Handle Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	if isConsoleAttached {
		fmt.Println()
		logMsg("👋", "Shutting down...", ColorDim)
	}
	cleanupADBReverse()
	if isConsoleAttached {
		logMsg("✅", "Goodbye!", ColorGreen)
	}
}
