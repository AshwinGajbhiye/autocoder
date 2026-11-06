#!/usr/bin/env python3
"""
Keyboard Punch — Laptop Companion Script

Receives text from the Android phone app via WebSocket (over ADB reverse USB or Wi-Fi LAN)
and simulates keyboard input using pyautogui.

Full cross-platform support: Windows, macOS, and Linux.

Usage:
    python server.py          # Normal mode
    python server.py --test   # Test mode (verify setup)

Architecture:
    Phone (Chrome) ←→ ADB Reverse / Wi-Fi ←→ This Script ←→ pyautogui → Active App
"""

import asyncio
import json
import os
import shutil
import signal
import socket
import subprocess
import sys
import threading
import time
from http.server import HTTPServer, SimpleHTTPRequestHandler
from pathlib import Path

# Enable ANSI escape sequences on Windows console
IS_WINDOWS = sys.platform == 'win32'
if IS_WINDOWS:
    try:
        os.system('')  # Triggers VT100 console mode in Windows 10/11
    except Exception:
        pass

# ─── Dependency Check ─────────────────────────────────────────
def check_dependencies():
    """Check and install required Python packages."""
    missing = []
    try:
        import websockets
    except ImportError:
        missing.append('websockets')
    
    try:
        import pyautogui
    except ImportError:
        missing.append('pyautogui')
    
    try:
        import pyperclip
    except ImportError:
        missing.append('pyperclip')
    
    if missing:
        print(f"📦 Installing missing packages: {', '.join(missing)}")
        subprocess.check_call([
            sys.executable, '-m', 'pip', 'install', '--quiet', *missing
        ])
        print("✅ Packages installed successfully")

check_dependencies()

import websockets
import pyautogui
import pyperclip

# ─── Configuration ────────────────────────────────────────────
WS_PORT = 8765
HTTP_PORT = 8080
APP_DIR = Path(__file__).parent.parent / 'app'

# pyautogui safety settings
pyautogui.FAILSAFE = True    # Move mouse to corner to abort
pyautogui.PAUSE = 0          # We handle our own delays

def ws_is_open(websocket):
    """Check if a WebSocket connection is still open (compatible with all versions)."""
    try:
        return websocket.state.name == 'OPEN'
    except AttributeError:
        try:
            return websocket.open
        except AttributeError:
            return True

# ─── Colors for terminal output ───────────────────────────────
class Color:
    PURPLE = '\033[95m'
    BLUE = '\033[94m'
    CYAN = '\033[96m'
    GREEN = '\033[92m'
    YELLOW = '\033[93m'
    RED = '\033[91m'
    BOLD = '\033[1m'
    DIM = '\033[2m'
    END = '\033[0m'

def log(icon, message, color=Color.END):
    """Pretty-print a log message."""
    print(f"  {icon}  {color}{message}{Color.END}")

def log_header(title):
    """Print a section header."""
    print(f"\n{Color.PURPLE}{Color.BOLD}{'─' * 50}{Color.END}")
    print(f"  {Color.PURPLE}{Color.BOLD}{title}{Color.END}")
    print(f"{Color.PURPLE}{Color.BOLD}{'─' * 50}{Color.END}\n")

# ─── ADB Setup & Detection ────────────────────────────────────
def find_adb():
    """
    Find ADB executable in PATH or standard installation locations.
    On Windows, checks Android Studio SDK platform-tools and standalone directories.
    """
    # 1. Check existing PATH
    adb_bin = shutil.which('adb') or shutil.which('adb.exe')
    if adb_bin:
        return adb_bin

    # 2. Check common Windows paths
    if IS_WINDOWS:
        candidate_dirs = [
            os.path.expandvars(r'%LOCALAPPDATA%\Android\Sdk\platform-tools'),
            os.path.expandvars(r'%USERPROFILE%\AppData\Local\Android\Sdk\platform-tools'),
            r'C:\platform-tools',
            os.path.expandvars(r'%PROGRAMFILES%\Android\android-sdk\platform-tools'),
            os.path.expandvars(r'%PROGRAMFILES(X86)%\Android\android-sdk\platform-tools'),
            str(Path(__file__).parent / 'platform-tools'),
            str(Path(__file__).parent.parent / 'platform-tools'),
        ]
        for cdir in candidate_dirs:
            candidate = os.path.join(cdir, 'adb.exe')
            if os.path.isfile(candidate):
                # Dynamically prepend to PATH so subsequent commands work
                os.environ['PATH'] = cdir + os.pathsep + os.environ.get('PATH', '')
                return candidate

    return None

def check_adb():
    """Verify ADB is available and a device is connected."""
    adb_bin = find_adb()
    if not adb_bin:
        if IS_WINDOWS:
            return False, (
                "ADB not found on PATH.\n"
                "  To install on Windows:\n"
                "    Option A (Recommended): winget install Google.PlatformTools\n"
                "    Option B: choco install adb\n"
                "    Option C: Download SDK Platform-Tools from:\n"
                "              https://developer.android.com/tools/releases/platform-tools\n"
                "              Extract to C:\\platform-tools and add to your PATH."
            )
        elif sys.platform == 'darwin':
            return False, (
                "ADB not found on PATH.\n"
                "  Install via: brew install android-platform-tools  (macOS)\n"
                "  Or download: https://developer.android.com/tools/releases/platform-tools"
            )
        else:
            return False, (
                "ADB not found on PATH.\n"
                "  Install via your package manager (e.g. sudo apt install adb)\n"
                "  Or download: https://developer.android.com/tools/releases/platform-tools"
            )

    # Check adb version
    try:
        result = subprocess.run(
            ['adb', 'version'],
            capture_output=True, text=True, timeout=5
        )
        if result.returncode != 0:
            return False, f"ADB version check failed: {result.stderr.strip()}"
    except Exception as e:
        return False, f"ADB error: {str(e)}"

    # Check for connected devices
    try:
        result = subprocess.run(
            ['adb', 'devices'],
            capture_output=True, text=True, timeout=10
        )
        lines = [l.strip() for l in result.stdout.strip().split('\n')[1:] if l.strip()]
        devices = [l for l in lines if 'device' in l and 'offline' not in l]

        if not devices:
            if IS_WINDOWS:
                return False, (
                    "No Android device detected.\n"
                    "  1. Connect your phone via USB cable\n"
                    "  2. Enable USB Debugging (Settings → Developer Options)\n"
                    "  3. Tap 'Allow USB debugging' on your phone\n"
                    "  4. On Windows: If device still isn't detected, install Google / OEM USB Driver\n"
                    "     (Check Windows Update → Optional Updates, or your phone maker's driver)"
                )
            else:
                return False, (
                    "No Android device detected.\n"
                    "  1. Connect your phone via USB cable\n"
                    "  2. Enable USB Debugging (Settings → Developer Options)\n"
                    "  3. Accept the USB debugging prompt on your phone"
                )

        return True, devices[0].split('\t')[0]
    except Exception as e:
        return False, f"Error checking ADB devices: {str(e)}"

def setup_adb_reverse():
    """Set up ADB reverse port forwarding for both WebSocket and HTTP."""
    log('🔌', 'Setting up ADB reverse port forwarding...', Color.CYAN)
    
    errors = []
    for port, name in [(WS_PORT, 'WebSocket'), (HTTP_PORT, 'HTTP')]:
        try:
            result = subprocess.run(
                ['adb', 'reverse', f'tcp:{port}', f'tcp:{port}'],
                capture_output=True, text=True, timeout=10
            )
            if result.returncode == 0:
                log('✓', f'{name} port {port} forwarded', Color.GREEN)
            else:
                errors.append(f'{name} port {port}: {result.stderr.strip()}')
        except Exception as e:
            errors.append(f'{name} port {port}: {str(e)}')
    
    if errors:
        for err in errors:
            log('✕', err, Color.RED)
        return False
    return True

def cleanup_adb_reverse():
    """Remove ADB reverse port forwarding."""
    try:
        subprocess.run(
            ['adb', 'reverse', '--remove-all'],
            capture_output=True, timeout=5
        )
    except Exception:
        pass

def get_local_ip():
    """Get the primary local network IP address."""
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(('8.8.8.8', 80))
        ip = s.getsockname()[0]
        s.close()
        return ip
    except Exception:
        try:
            hostname = socket.gethostname()
            ip = socket.gethostbyname(hostname)
            if ip and not ip.startswith('127.'):
                return ip
            for addr in socket.gethostbyname_ex(hostname)[2]:
                if not addr.startswith('127.'):
                    return addr
        except Exception:
            pass
        return 'localhost'

async def monitor_adb_loop():
    """Periodically check for newly connected ADB devices and configure reverse forwarding."""
    adb_reversed = False
    while True:
        try:
            ok, info = check_adb()
            if ok and not adb_reversed:
                log('📱', f'Android device detected via USB: {info}', Color.GREEN)
                if setup_adb_reverse():
                    adb_reversed = True
                    log('✨', f'ADB reverse active! You can now open http://localhost:{HTTP_PORT} on your phone', Color.GREEN)
            elif not ok and adb_reversed:
                adb_reversed = False
                log('⚠️ ', 'Android device disconnected from ADB.', Color.YELLOW)
        except Exception:
            pass
        await asyncio.sleep(3)

# ─── HTTP Server (serves the phone web app) ──────────────────
class ReusableHTTPServer(HTTPServer):
    """
    HTTP server with SO_REUSEADDR.
    Guards SO_REUSEPORT so it runs without errors on Windows (where SO_REUSEPORT is absent).
    """
    allow_reuse_address = True
    allow_reuse_port = hasattr(socket, 'SO_REUSEPORT')

class QuietHTTPHandler(SimpleHTTPRequestHandler):
    """HTTP handler that serves the app/ directory silently."""
    
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(APP_DIR), **kwargs)
    
    def log_message(self, format, *args):
        """Suppress HTTP access logs."""
        pass

def start_http_server():
    """Start the HTTP server in a background thread."""
    server = ReusableHTTPServer(('0.0.0.0', HTTP_PORT), QuietHTTPHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    log('🌐', f'HTTP server running on port {HTTP_PORT}', Color.GREEN)
    return server

# ─── Keyboard Simulation ─────────────────────────────────────

# Map of special characters to their pyautogui key names
SPECIAL_KEYS = {
    '\n': 'enter',
    '\t': 'tab',
}

def type_single_character(char, preserve=True):
    """
    Simulate typing a single character with support for Windows, macOS, and Linux.
    Handles special keys, ASCII, Unicode characters, and emojis cleanly.
    """
    if char == '\r':
        # Skip carriage returns (normalized newlines are handled by '\n')
        return

    if char in SPECIAL_KEYS and preserve:
        pyautogui.press(SPECIAL_KEYS[char])
        return

    if char == '\n' and not preserve:
        pyautogui.typewrite(' ', interval=0)
        return

    if char == '\t' and not preserve:
        pyautogui.typewrite('    ', interval=0)
        return

    # Regular character
    try:
        pyautogui.write(char)
    except Exception:
        # Fallback for Unicode or characters not mapped in the active keyboard layout
        try:
            pyperclip.copy(char)
            if sys.platform == 'darwin':
                pyautogui.hotkey('command', 'v')
            else:
                pyautogui.hotkey('ctrl', 'v')
        except Exception as err:
            raise RuntimeError(f"Could not type character {repr(char)}: {err}")

async def type_text(text, delay_ms, preserve_formatting, ws_client):
    """
    Type text character by character with the specified delay.
    Sends progress updates back to the phone via WebSocket.
    """
    # Normalize Windows CRLF and Mac CR newlines to standard LF
    normalized_text = text.replace('\r\n', '\n').replace('\r', '\n')
    total = len(normalized_text)
    
    for i, char in enumerate(normalized_text):
        if not ws_is_open(ws_client):
            return False
        
        try:
            type_single_character(char, preserve=preserve_formatting)
        except Exception as e:
            await ws_client.send(json.dumps({
                'type': 'error',
                'message': f'Typing error at char {i}: {str(e)}'
            }))
            return False
        
        # Send progress update every 5 characters or at the end
        if i % 5 == 0 or i == total - 1:
            try:
                await ws_client.send(json.dumps({
                    'type': 'progress',
                    'current': i + 1,
                    'total': total
                }))
            except Exception:
                return False
        
        # Delay between keystrokes
        if delay_ms > 0 and i < total - 1:
            await asyncio.sleep(delay_ms / 1000.0)
    
    return True

# ─── WebSocket Server & Multi-Device Sync ─────────────────────

# Track all connected WebSocket clients (mobile phone, laptop browser tabs, etc.)
connected_clients = set()

# Shared state synchronized across all devices
shared_state = {
    'text': '',
    'wpm': 100,
    'focus_delay_sec': 3,
    'preserve_formatting': True
}

# Track active punching task so we can cancel it
active_task = None
should_stop = False

async def broadcast(message_dict, sender=None, exclude_sender=False):
    """Send a message to all connected clients."""
    payload = json.dumps(message_dict)
    disconnected = []
    for client in list(connected_clients):
        if exclude_sender and client == sender:
            continue
        if ws_is_open(client):
            try:
                await client.send(payload)
            except Exception:
                disconnected.append(client)
        else:
            disconnected.append(client)
    for client in disconnected:
        connected_clients.discard(client)

async def handle_client(websocket):
    """Handle a WebSocket connection from a phone or laptop app."""
    global active_task, should_stop, shared_state
    
    client_ip = websocket.remote_address[0] if websocket.remote_address else 'unknown'
    connected_clients.add(websocket)
    log('📱', f'Device connected from {client_ip} (Total connected: {len(connected_clients)})', Color.GREEN)
    
    try:
        # Send current shared state to newly connected client
        await websocket.send(json.dumps({
            'type': 'sync_state',
            'state': shared_state
        }))

        async for raw_message in websocket:
            try:
                msg = json.loads(raw_message)
            except json.JSONDecodeError:
                continue
            
            msg_type = msg.get('type', '')

            if msg_type == 'update_code':
                new_text = msg.get('text', '')
                shared_state['text'] = new_text
                if 'wpm' in msg:
                    shared_state['wpm'] = msg['wpm']
                if 'focus_delay_sec' in msg:
                    shared_state['focus_delay_sec'] = msg['focus_delay_sec']
                if 'preserve_formatting' in msg:
                    shared_state['preserve_formatting'] = msg['preserve_formatting']

                log('🔄', f'Code updated ({len(new_text)} chars) from {client_ip}. Syncing to {len(connected_clients)} device(s)', Color.CYAN)
                
                # Broadcast updated code to all connected clients
                await broadcast({
                    'type': 'code_updated',
                    'text': new_text,
                    'wpm': shared_state['wpm'],
                    'focus_delay_sec': shared_state['focus_delay_sec'],
                    'preserve_formatting': shared_state['preserve_formatting'],
                    'sender': client_ip
                })
            
            elif msg_type == 'punch':
                raw_text = msg.get('text', '')
                delay_ms = msg.get('delay_ms', 50)
                preserve = msg.get('preserve_formatting', True)
                wpm = msg.get('wpm', 100)
                
                if not raw_text:
                    await websocket.send(json.dumps({
                        'type': 'error',
                        'message': 'No text to type'
                    }))
                    continue
                
                # Normalize line endings to avoid double newlines on Windows
                text = raw_text.replace('\r\n', '\n').replace('\r', '\n')
                
                # Save into shared state
                shared_state['text'] = text
                shared_state['wpm'] = wpm
                shared_state['preserve_formatting'] = preserve

                log('⌨️ ', f'Typing {len(text)} chars at {wpm} WPM ({delay_ms}ms delay)', Color.CYAN)
                
                should_stop = False
                
                async def do_punch():
                    global should_stop
                    try:
                        total = len(text)
                        for i, char in enumerate(text):
                            if should_stop:
                                await broadcast({'type': 'stopped'})
                                log('■', 'Typing stopped by user', Color.YELLOW)
                                return
                            
                            # Check if at least one client is still open
                            if not any(ws_is_open(c) for c in connected_clients):
                                return
                            
                            try:
                                type_single_character(char, preserve=preserve)
                            except Exception as e:
                                await broadcast({
                                    'type': 'error',
                                    'message': f'Typing error at position {i}: {str(e)}'
                                })
                                return
                            
                            # Progress update every 5 chars or at the end
                            if i % 5 == 0 or i == total - 1:
                                await broadcast({
                                    'type': 'progress',
                                    'current': i + 1,
                                    'total': total
                                })
                            
                            if delay_ms > 0 and i < total - 1:
                                await asyncio.sleep(delay_ms / 1000.0)
                        
                        # Done!
                        await broadcast({'type': 'complete'})
                        log('✅', 'Typing complete!', Color.GREEN)
                    
                    except asyncio.CancelledError:
                        log('■', 'Typing cancelled', Color.YELLOW)
                    except Exception as e:
                        log('❌', f'Error: {str(e)}', Color.RED)
                        await broadcast({
                            'type': 'error',
                            'message': str(e)
                        })
                
                # Cancel any existing punch task
                if active_task and not active_task.done():
                    active_task.cancel()
                    try:
                        await active_task
                    except (asyncio.CancelledError, Exception):
                        pass
                
                active_task = asyncio.create_task(do_punch())
            
            elif msg_type == 'stop':
                should_stop = True
                log('⏹', 'Stop requested', Color.YELLOW)
                await broadcast({'type': 'stopped'})
            
            elif msg_type == 'ping':
                await websocket.send(json.dumps({'type': 'pong'}))
    
    except websockets.exceptions.ConnectionClosed:
        pass
    finally:
        connected_clients.discard(websocket)
        log('📱', f'Device disconnected ({client_ip}). Remaining: {len(connected_clients)}', Color.DIM)
        if not connected_clients:
            should_stop = True

async def start_ws_server():
    """Start the WebSocket server."""
    server = await websockets.serve(
        handle_client,
        '0.0.0.0',
        WS_PORT,
        ping_interval=60,
        ping_timeout=120,
        max_size=10 * 1024 * 1024,  # 10MB max message size
    )
    log('🔗', f'WebSocket server running on port {WS_PORT}', Color.GREEN)
    return server

# ─── Test Mode ────────────────────────────────────────────────
def run_tests():
    """Verify the setup is working correctly across platforms."""
    log_header('Keyboard Punch — Setup Test')
    
    all_ok = True
    
    # Test 1: Python packages
    log('1️⃣ ', 'Checking Python packages...', Color.CYAN)
    for pkg in ['websockets', 'pyautogui', 'pyperclip']:
        try:
            mod = __import__(pkg)
            ver = getattr(mod, '__version__', 'installed')
            log('  ✓', f'{pkg} ({ver})', Color.GREEN)
        except Exception as e:
            log('  ✕', f'{pkg}: {e}', Color.RED)
            all_ok = False
    
    # Test 2: ADB
    log('2️⃣ ', 'Checking ADB...', Color.CYAN)
    ok, info = check_adb()
    if ok:
        log('  ✓', f'Device found: {info}', Color.GREEN)
    else:
        log('  ✕', info, Color.RED)
        all_ok = False
    
    # Test 3: App directory
    log('3️⃣ ', 'Checking app files...', Color.CYAN)
    if APP_DIR.exists():
        files = ['index.html', 'style.css', 'app.js']
        for f in files:
            if (APP_DIR / f).exists():
                log('  ✓', f'{f} found', Color.GREEN)
            else:
                log('  ✕', f'{f} missing', Color.RED)
                all_ok = False
    else:
        log('  ✕', f'App directory not found: {APP_DIR}', Color.RED)
        all_ok = False
    
    # Test 4: Keyboard simulation & Platform check
    log('4️⃣ ', 'Checking keyboard simulation & permissions...', Color.CYAN)
    try:
        size = pyautogui.size()
        log('  ✓', f'Screen detected: {size.width}x{size.height}', Color.GREEN)
        if IS_WINDOWS:
            log('  ℹ️ ', 'On Windows: No special Accessibility permission is needed.', Color.CYAN)
            log('  💡', 'Tip: If typing into an elevated app (e.g. Admin Command Prompt or VS Code),', Color.YELLOW)
            log('     run Keyboard Punch as Administrator as well.', Color.YELLOW)
        elif sys.platform == 'darwin':
            log('  💡', 'On macOS: System Settings → Privacy → Accessibility → Enable Terminal', Color.YELLOW)
    except Exception as e:
        log('  ✕', f'pyautogui error: {e}', Color.RED)
        all_ok = False
    
    print()
    if all_ok:
        log('🎉', 'All tests passed! Run without --test to start.', Color.GREEN)
    else:
        log('⚠️ ', 'Some checks had warnings or issues. Check the details above.', Color.YELLOW)
    print()
    
    return all_ok

# ─── Main ─────────────────────────────────────────────────────
async def main():
    """Main entry point."""
    os_name = "Windows" if IS_WINDOWS else ("macOS" if sys.platform == 'darwin' else "Linux")
    log_header(f'⌨️  Keyboard Punch — Companion ({os_name})')
    
    # Check ADB
    log('🔍', 'Checking ADB connection...', Color.CYAN)
    ok, info = check_adb()
    if not ok:
        log('⚠️ ', info, Color.YELLOW)
        log('💡', 'No USB phone detected yet. Connect phone with USB Debugging for ADB reverse.', Color.DIM)
        log('🌐', 'Starting web & WebSocket servers for browser & local network access...', Color.CYAN)
    else:
        log('✅', f'Device connected: {info}', Color.GREEN)
        if not setup_adb_reverse():
            log('⚠️ ', 'ADB reverse setup failed. Try running: adb reverse --remove-all', Color.YELLOW)
    
    # Start HTTP server
    http_server = start_http_server()
    
    # Start WebSocket server
    ws_server = await start_ws_server()
    
    # Start ADB device watcher loop
    asyncio.create_task(monitor_adb_loop())
    
    local_ip = get_local_ip()
    
    print()
    log_header('🚀 Ready!')
    device_label = "Windows PC" if IS_WINDOWS else ("Mac" if sys.platform == 'darwin' else "Laptop")
    log('💻', f'Open on {device_label}:      {Color.BOLD}http://localhost:{HTTP_PORT}{Color.END}', Color.CYAN)
    log('📱', f'Open on Phone (USB):   {Color.BOLD}http://localhost:{HTTP_PORT}{Color.END} (requires USB Debugging)', Color.CYAN)
    if local_ip != 'localhost':
        log('📶', f'Open on Phone (Wi-Fi): {Color.BOLD}http://{local_ip}:{HTTP_PORT}{Color.END} (must be on same Wi-Fi)', Color.GREEN)
    log('🛑', f'Press {Color.BOLD}Ctrl+C{Color.END} to stop', Color.DIM)
    log('🛡️', f'Move mouse to screen corner to emergency-stop typing', Color.DIM)
    print()
    
    # Keep running until Ctrl+C (periodic sleep allows prompt Windows signal dispatch)
    try:
        while True:
            await asyncio.sleep(1)
    except (asyncio.CancelledError, KeyboardInterrupt):
        pass
    finally:
        log('👋', 'Shutting down...', Color.DIM)
        ws_server.close()
        await ws_server.wait_closed()
        http_server.shutdown()
        cleanup_adb_reverse()
        log('✅', 'Goodbye!', Color.GREEN)

def signal_handler(sig, frame):
    """Handle Ctrl+C gracefully."""
    print()
    log('👋', 'Shutting down...', Color.DIM)
    cleanup_adb_reverse()
    sys.exit(0)

if __name__ == '__main__':
    try:
        signal.signal(signal.SIGINT, signal_handler)
    except Exception:
        pass
    if hasattr(signal, 'SIGTERM'):
        try:
            signal.signal(signal.SIGTERM, signal_handler)
        except Exception:
            pass
    if hasattr(signal, 'SIGBREAK'):
        try:
            signal.signal(signal.SIGBREAK, signal_handler)
        except Exception:
            pass
    
    if '--test' in sys.argv:
        success = run_tests()
        sys.exit(0 if success else 1)
    else:
        try:
            asyncio.run(main())
        except KeyboardInterrupt:
            cleanup_adb_reverse()
