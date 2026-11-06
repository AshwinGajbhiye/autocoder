# Keyboard Punch — Windows PowerShell Launcher
$Host.UI.RawUI.WindowTitle = "Keyboard Punch Companion - Windows"

Write-Host "===================================================" -ForegroundColor Cyan
Write-Host "   Keyboard Punch - Laptop Companion (Windows)     " -ForegroundColor Cyan
Write-Host "===================================================" -ForegroundColor Cyan
Write-Host ""

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# 1. Detect Python
$pythonCmd = $null
if (Get-Command python -ErrorAction SilentlyContinue) {
    $pythonCmd = "python"
} elseif (Get-Command py -ErrorAction SilentlyContinue) {
    $pythonCmd = "py -3"
} elseif (Get-Command python3 -ErrorAction SilentlyContinue) {
    $pythonCmd = "python3"
}

if (-not $pythonCmd) {
    Write-Host "[ERROR] Python 3 was not found on your system!" -ForegroundColor Red
    Write-Host ""
    Write-Host "To install Python on Windows:" -ForegroundColor Yellow
    Write-Host "  winget install Python.Python.3.11" -ForegroundColor Yellow
    Write-Host "  Or download from: https://www.python.org/downloads/" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "IMPORTANT: Ensure 'Add python.exe to PATH' is checked!" -ForegroundColor Yellow
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Host "[OK] Using Python: $pythonCmd" -ForegroundColor Green

# 2. Check for ADB in standard locations
if (-not (Get-Command adb -ErrorAction SilentlyContinue)) {
    $candidates = @(
        "$env:LOCALAPPDATA\Android\Sdk\platform-tools",
        "C:\platform-tools",
        "$env:ProgramFiles\Android\android-sdk\platform-tools",
        "${env:ProgramFiles(x86)}\Android\android-sdk\platform-tools"
    )
    foreach ($dir in $candidates) {
        if (Test-Path "$dir\adb.exe") {
            $env:PATH = "$dir;$env:PATH"
            Write-Host "[OK] Detected ADB in $dir" -ForegroundColor Green
            break
        }
    }
}

# 3. Check dependencies
Write-Host ""
Write-Host "[*] Checking Python dependencies..." -ForegroundColor Cyan
$reqPath = Join-Path $ScriptDir "requirements.txt"
Invoke-Expression "$pythonCmd -m pip install --quiet -r `"$reqPath`""
if ($LASTEXITCODE -ne 0) {
    Write-Host "[WARN] Global pip install failed, attempting --user install..." -ForegroundColor Yellow
    Invoke-Expression "$pythonCmd -m pip install --quiet --user -r `"$reqPath`""
}

# 4. Run server
Write-Host ""
Write-Host "[*] Starting companion server..." -ForegroundColor Green
Write-Host ""
$serverPy = Join-Path $ScriptDir "server.py"
Invoke-Expression "$pythonCmd `"$serverPy`" $args"

if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Host "[ERROR] Server stopped with error code $LASTEXITCODE" -ForegroundColor Red
    Read-Host "Press Enter to exit"
}
