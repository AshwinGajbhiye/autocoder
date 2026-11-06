# ⌨️ Keyboard Punch (Windows Edition)

**Turn your Android phone into a USB / Wi-Fi keyboard for your Windows PC or laptop.**

Type or paste text on your phone, press "Start Typing", and watch it get typed out on your laptop — character by character, at your chosen speed (10–1000 WPM).

---

## How It Works

```
📱 Phone (Chrome)  ──USB-C / Wi-Fi──►  💻 Laptop (Python)  ──►  🖥️ Active App
   Web App UI          ADB Reverse          Companion Server        Keystroke Sim
                       / Local LAN
```

1. The **companion server** runs on your Windows PC — it sets up ADB reverse port forwarding over USB-C or serves over Wi-Fi
2. Open **Chrome on your phone** → `http://localhost:8080` (over USB) or the displayed Wi-Fi IP
3. Type or paste your code/text, configure speed (10–1000 WPM), and press **"Start Typing"**
4. Text is sent via WebSocket and typed into whatever application has focus on your PC (VS Code, Notepad, Terminal, etc.)

> **No Wi-Fi needed over USB.** All communication happens through the USB cable via ADB reverse. Wi-Fi mode is also available when on the same network.

---

## Windows Prerequisites

| Requirement | How to Get It on Windows |
|-------------|--------------------------|
| **Python 3.8+** | `winget install Python.Python.3.11`<br>Or download from [python.org](https://www.python.org/downloads/).<br>*(⚠️ Check the box **"Add python.exe to PATH"** during setup)* |
| **ADB (Platform Tools)** | `winget install Google.PlatformTools`<br>Or download [Android SDK Platform-Tools](https://developer.android.com/tools/releases/platform-tools).<br>*(Auto-detected if you have Android Studio installed!)* |
| **USB Debugging** | On your phone: Settings → About Phone → Tap "Build Number" 7 times → Back → Developer Options → Enable **"USB Debugging"** |
| **Android USB Driver** | Usually automatic on Windows 10/11. If ADB doesn't detect your phone, install the OEM USB driver via Windows Update (Optional updates) or your phone maker's site. |
| **USB Cable** | A data-capable USB cable (not charge-only) |

*(For macOS/Linux users: Python 3 and `brew install android-platform-tools` or `sudo apt install adb` are also supported).*

---

## Quick Start on Windows

### 1. Connect your phone via USB
Plug in your Android phone. If a popup appears on your phone asking to **"Allow USB Debugging"**, tap **Allow** (check "Always allow from this computer").

### 2. Launch the companion server
You can start Keyboard Punch with a single click:

- **Option A (Double-Click):** Double-click **`start.bat`** in the repository root.
- **Option B (Command Prompt):**
  ```cmd
  start.bat
  ```
- **Option C (PowerShell):**
  ```powershell
  .\start.ps1
  ```
- **Option D (Manual):**
  ```cmd
  pip install -r companion\requirements.txt
  python companion\server.py
  ```

### 3. Open the app on your phone
Open **Chrome** on your phone and go to:
```
http://localhost:8080
```
*(If connecting over Wi-Fi, use the `http://<YOUR-PC-IP>:8080` link shown in the console).*

### 4. Start typing!
1. Type or paste your text or code in the text box
2. Adjust typing speed (10–1000 WPM)
3. Click into your target app on your PC (e.g., VS Code, Notepad, Windows Terminal)
4. Press **"Start Typing"** on your phone
5. The 3-second countdown gives you time to focus your target window

---

## Features

- **⚡ Configurable Speed** — 10 to 1000 WPM with real-time ETA calculation
- **🪟 Windows Optimized** — Smart ADB path auto-detection, CRLF newline normalization, and Unicode typing fallback
- **📝 Full Formatting** — Preserves tabs, newlines, spaces, indentation, and special characters
- **⏱️ Focus Delay Countdown** — Configurable delay (0 to 60s) to switch to your target editor
- **📊 Real-Time Progress** — Live progress bar showing typed characters and remaining time
- **🔄 Multi-Device Sync** — Edit on PC browser or phone with instant live synchronization
- **🛑 Emergency Failsafe** — Move mouse cursor to any corner of the screen to immediately abort typing, or press Stop
- **📋 Clipboard & History** — Remembers recent punch history with one-tap reuse
- **🎨 Modern Dark UI** — Glassmorphism responsive design with smooth animations

---

## Verify Setup

Run the built-in diagnostic test to verify your Windows environment:

```cmd
python companion\server.py --test
```

This verifies:
- ✅ Python dependencies installed (`websockets`, `pyautogui`, `pyperclip`)
- ✅ ADB found and Android device detected
- ✅ Web application files present
- ✅ Keyboard simulation and screen resolution detected

---

## Windows Troubleshooting & Tips

### 1. "No Android device detected"
- Make sure **USB Debugging** is turned on in Developer Options on your phone.
- Verify that your USB cable supports data transfer (some cables are charge-only).
- In Windows Device Manager, check if your phone appears with a yellow exclamation mark under "Portable Devices" or "Other Devices". If so, install the Google USB Driver or OEM driver (Samsung, Google, Xiaomi, etc.).
- Run `adb devices` in Command Prompt to see if the device shows as `device` or `unauthorized`.

### 2. Windows Firewall Prompt
- When running the server for the first time, Windows Defender Firewall may display a prompt asking to allow network communication.
- Click **"Allow access"** on Private networks to allow phone browser access.

### 3. Typing into Elevated / Admin Windows (VS Code or CMD as Administrator)
- Due to Windows User Interface Privilege Isolation (UIPI), a standard process cannot send simulated keystrokes to a window running with Administrator privileges.
- If your target editor is running as Administrator, right-click `start.bat` and select **"Run as Administrator"**.

### 4. ADB Not in PATH
- The companion script automatically detects ADB if installed in:
  - `%LOCALAPPDATA%\Android\Sdk\platform-tools` (default Android Studio location)
  - `C:\platform-tools`
- If you have ADB elsewhere, either add its folder to your Windows system `PATH` or install via `winget install Google.PlatformTools`.

### 5. Line Endings & Formatting
- The Windows edition automatically normalizes CRLF (`\r\n`) and CR (`\r`) line endings to prevent double newlines.
- If typing code into an editor with auto-indentation enabled, you can toggle off "Preserve newlines & whitespace" or disable temporary auto-closing quotes in the editor.

---

## Project Structure

```
autocoder/
├── start.bat                  ← Windows 1-click launcher (root)
├── start.ps1                  ← Windows PowerShell launcher (root)
├── README.md                  ← Documentation
├── app/                       ← Web application (phone & desktop)
│   ├── index.html            ← Main HTML layout
│   ├── style.css             ← Dark glassmorphism theme
│   └── app.js                ← Client WebSocket logic & state sync
└── companion/                 ← Laptop companion server
    ├── server.py             ← Cross-platform server (ADB + WebSocket + keystrokes)
    ├── requirements.txt      ← Python packages (websockets, pyautogui, pyperclip)
    ├── start.bat             ← Windows Command Prompt launcher
    ├── start.ps1             ← Windows PowerShell launcher
    └── start.sh              ← macOS / Linux bash launcher
```

---

## Safety

- **Mouse Corner Failsafe**: Slam the mouse cursor into any corner of the screen to immediately abort typing.
- **Stop Button**: Press the red "Stop" button in the web interface at any time.
- **Ctrl+C**: Cleanly terminates the server and cleans up ADB reverse bindings.

---

## License

MIT
