# ⌨️ Keyboard Punch

**Turn your Android phone into a high-speed, undetectable USB keyboard for your laptop.**

Type or paste text on your phone, press "Start Typing", and watch it get typed out on your laptop — character by character, with human-like variance, smart IDE indentation, and bot-evasion mechanisms.

---

## How It Works

```
📱 Phone (Chrome / Safari) ──[Wi-Fi / Hotspot QR Code]──► 💻 Laptop (Go Native) ──► 🖥️ Active App
            OR                                               Companion Engine         Keystroke Sim
📱 Phone (Android)         ──[USB Cable (ADB Reverse)]───►   (Zero Dependencies)
```

1. The **native companion engine** runs silently on your laptop in **Ghost Mode** (System Tray near clock, zero console window).
2. Connect either:
   - **Zero-Cable Wi-Fi / Hotspot:** Scan the pairing QR code with your phone camera!
   - **USB Cable (ADB reverse):** Instant tethered connection with no local network required.
3. Open the web app on your phone → enter or paste code → click **"Start Typing"**.
4. Keystrokes are realistically simulated into your active window (VS Code, Cursor, LeetCode, terminal, or interview code editors).

---

## 🚀 Quick Start

### Option A: Zero-Cable Wi-Fi Pairing (Recommended)
1. **Download & Run**:
   - Double-click `KeyboardPunch.exe` (runs quietly in your System Tray near the clock).
2. **Pair Your Phone**:
   - Right-click the keyboard icon in the System Tray → click **"📱 Pair Phone (Wi-Fi QR Code)"**.
   - Ensure your laptop and phone are on the same Wi-Fi (or turn on your phone's **Personal Hotspot** and connect your laptop).
   - Point your phone camera at the QR code and tap to open!
3. **Start typing!**

### Option B: USB Cable (ADB Reverse)
1. Turn on **USB Debugging** on your phone (*Settings → Developer Options → USB Debugging*).
2. Connect your phone via USB cable and run `KeyboardPunch.exe`.
3. Open Chrome on your phone to `http://localhost:8080`.

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

- **📶 Zero-Cable Wi-Fi / QR Code Pairing** — Instant pairing over local Wi-Fi or phone hotspot. Scan the QR code with phone camera to connect immediately with no USB or ADB setup required.
- **👻 Ghost / Headless Mode** — Silent System Tray operation near the clock (`go build -ldflags="-H=windowsgui"`). Zero black console window, zero taskbar presence — interview and screen-share safe.
- **🚀 Ultra-Compact Native Executable (~7.6 MB)** — Powered by Go. No Python, no heavy runtime, instant startup in <10ms.
- **🧠 Smart IDE Auto-Indent** — Understands code editors (VS Code, Cursor, LeetCode). Detects colons `:` and braces `{`, lets the IDE auto-indent naturally, and presses `Backspace` when dedenting.
- **⌨️ Realistic Human Key Dwell Time** — Simulates realistic 15–35ms physical key-press dwell times between `KEYEVENTF_KEYDOWN` and `KEYEVENTF_KEYUP`.
- **🛡️ Mouse Corner Failsafe** — Emergency abort: simply slam your mouse into the top-left corner of the screen `(0,0)` to instantly abort typing.
- **👻 Human-like Jitter Mode** — Applies a natural Gaussian distribution to typing delays to mimic real muscle memory and bypass anti-bot timing checks.
- **🙈 Oops Mode (Typo Simulator)** — Mathematical bot-bypass. Has a 3% chance per keystroke to mistype a letter, pause, backspace, and correct it.
- **📋 Fast Paste Mode** — Instantly transfers large codebases via the system clipboard (`Ctrl+V`).
- **⚡ Adjustable Speed** — 10 to 500 WPM with real-time ETA calculation.
- **⏱️ Focus Countdown** — Configurable countdown giving you time to switch focus to your target editor.
- **📊 Multi-Device Synchronization** — Live sync between phone, tablet, and laptop browser tabs over WebSockets.
- **🎨 Glassmorphism Dark UI** — Ambient glow, smooth animations, and clean mobile interface.

---

## 🔍 Troubleshooting

### "No Android device detected"
- Ensure **USB Debugging** is turned ON in Developer Options.
- Re-plug your USB cable and check your phone's notification panel: change USB mode from *"Charging only"* to *"File Transfer (MTP)"*.
- Look at your phone screen for the **"Allow USB Debugging?"** popup and tap **Allow**.
- Test with `adb devices` in your terminal to verify your device is listed.

### "Site can't be reached at localhost:8080"
- Make sure `KeyboardPunch.exe` is currently running (check your System Tray near the clock).
- Force plain HTTP instead of HTTPS (`http://localhost:8080` or `http://<laptop-ip>:8080`).
- If using USB, ensure ADB reverse port forwarding is active: `adb reverse tcp:8080 tcp:8080`.
- If using Wi-Fi, ensure your phone and laptop are on the same Wi-Fi network, or use your phone's Personal Hotspot.

---

## 📁 Project Structure

```
autocoder/
├── main.go            ← Companion engine (Web server + WS + Win32 SendInput + IDE Indentation)
├── network.go         ← Local IP detection (filters virtual NICs) & dynamic QR generator
├── tray.go            ← Pure Win32 system tray menu (Shell_NotifyIconW, TrackPopupMenu)
├── go.mod             ← Go module definitions
├── go.sum             ← Checksums for Go dependencies
├── app/               ← Embedded Web UI (served directly from the binary)
│   ├── index.html     ← Main frontend interface with in-app QR modal
│   ├── pair.html      ← Standalone full-screen QR pairing page
│   ├── style.css      ← Modern dark theme styling
│   └── app.js         ← WebSocket client & mobile state manager
└── dist/              ← Standalone pre-built executable (KeyboardPunch.exe)
```

---

## 📄 License

MIT License — feel free to use, modify, and contribute!
