# ⌨️ Keyboard Punch

**Turn your Android phone into a high-speed, undetectable USB keyboard for your laptop.**

Type or paste text on your phone, press "Start Typing", and watch it get typed out on your laptop — character by character, with human-like variance, smart IDE indentation, and bot-evasion mechanisms.

---

## How It Works

```
📱 Phone (Chrome)  ──USB Cable──►  💻 Laptop (Go Native)  ──►  🖥️ Active App
   Web App UI             ADB          Companion Engine          Keystroke Sim
                        Reverse        (Zero Dependencies)
```

1. The **native companion engine** runs on your laptop — lightweight, instant startup, zero external runtime dependencies.
2. It configures ADB reverse port forwarding over USB.
3. You open **Chrome on your phone** → `http://localhost:8080`.
4. Type or paste your code/text, set your speed, and press **"Start Typing"**.
5. Keystrokes are simulated directly into your active window (VS Code, terminal, browser, interview editor).

> **No Wi-Fi needed.** All communication happens through the USB cable.

---

## 🚀 Quick Start (Easiest Method — No Python or Git Needed)

1. **Connect your phone via USB**:
   - Turn on **USB Debugging** on your phone (*Settings → Developer Options → USB Debugging*).
   - If prompted on your phone, tap **"Always allow from this computer"**.
2. **Download & Run**:
   - Download `KeyboardPunch.exe` from the latest release.
   - Double-click to run! (No installation or terminal required).
3. **Open on your phone**:
   - Open Chrome on your phone and go to:
     ```
     http://localhost:8080
     ```
4. **Start typing!**

---

## 🛠️ Building From Source (Developers)

The companion engine is written in **Go** for native performance, zero-dependency distribution, and minimal binary size (~7 MB).

### Prerequisites

| Requirement | How to Get It |
|-------------|---------------|
| **Go 1.20+** | Download from [golang.org](https://golang.org) (Only if building from source) |
| **ADB** | **Windows:** `winget install Google.PlatformTools`<br>**macOS:** `brew install android-platform-tools`<br>**Linux:** `sudo apt install adb` |
| **USB Cable** | A data-capable USB cable (not charge-only) |

### Compile & Run

```bash
# 1. Clone the repository
git clone https://github.com/aditya-s45/autocoder.git
cd autocoder

# 2. Run directly
go run main.go

# 3. Or build the ultra-compact standalone binary (~7MB)
go build -ldflags="-s -w" -o dist/KeyboardPunch.exe main.go
```

---

## ✨ Features

- **🚀 Ultra-Compact Native Executable (~7 MB)** — Powered by Go. No Python, no heavy runtime, instant startup in <10ms.
- **🧠 Smart IDE Auto-Indent** — Understands code editors (VS Code, Cursor, LeetCode). Detects colons `:` and braces `{`, lets the IDE auto-indent naturally, and presses `Backspace` when dedenting.
- **👻 Human-like Jitter Mode** — Applies a natural Gaussian distribution to typing delays to mimic real muscle memory and bypass anti-bot timing checks.
- **🙈 Oops Mode (Typo Simulator)** — Mathematical bot-bypass. Has a 3% chance per keystroke to mistype a letter, pause, backspace, and correct it.
- **📋 Fast Paste Mode** — Instantly transfers large codebases via the system clipboard (`Ctrl+V`).
- **⚡ Adjustable Speed** — 10 to 500 WPM with real-time ETA calculation.
- **⏱️ Focus Countdown** — Configurable countdown giving you time to switch focus to your target editor.
- **📊 Multi-Device Synchronization** — Live sync between phone, tablet, and laptop browser tabs over WebSockets.
- **🛑 Emergency Stop** — Stop button on phone or instant `Ctrl+C` in terminal.
- **🎨 Glassmorphism Dark UI** — Ambient glow, smooth animations, and clean mobile interface.

---

## 🔍 Troubleshooting

### "No Android device detected"
- Ensure **USB Debugging** is turned ON in Developer Options.
- Re-plug your USB cable and check your phone's notification panel: change USB mode from *"Charging only"* to *"File Transfer (MTP)"*.
- Look at your phone screen for the **"Allow USB Debugging?"** popup and tap **Allow**.
- Test with `adb devices` in your terminal to verify your device is listed.

### "Site can't be reached at localhost:8080"
- Make sure `KeyboardPunch.exe` (or `go run main.go`) is currently running.
- Force plain HTTP instead of HTTPS (`http://localhost:8080`).
- Ensure ADB reverse port forwarding is active: `adb reverse tcp:8080 tcp:8080`.

---

## 📁 Project Structure

```
autocoder/
├── main.go            ← Native Go companion engine (Web server + WS + Win32 SendInput)
├── go.mod             ← Go module definitions
├── go.sum             ← Checksums for Go dependencies
├── app/               ← Embedded Web UI (served directly from the binary)
│   ├── index.html     ← Main frontend interface
│   ├── style.css      ← Modern dark theme styling
│   └── app.js         ← WebSocket client & mobile state manager
└── dist/              ← Standalone pre-built executable (KeyboardPunch.exe)
```

---

## 📄 License

MIT License — feel free to use, modify, and contribute!
