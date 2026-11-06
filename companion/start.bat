@echo off
setlocal enabledelayedexpansion
title Keyboard Punch Companion - Windows

echo ===================================================
echo   Keyboard Punch - Laptop Companion (Windows)
echo ===================================================
echo.

set "SCRIPT_DIR=%~dp0"

:: 1. Detect Python
set "PY_CMD="
where python >nul 2>&1
if %errorlevel% equ 0 (
    set "PY_CMD=python"
) else (
    where py >nul 2>&1
    if %errorlevel% equ 0 (
        set "PY_CMD=py -3"
    ) else (
        where python3 >nul 2>&1
        if %errorlevel% equ 0 (
            set "PY_CMD=python3"
        )
    )
)

if "%PY_CMD%"=="" (
    echo [ERROR] Python was not found on your system!
    echo.
    echo To install Python on Windows:
    echo   1. Run in PowerShell: winget install Python.Python.3.11
    echo   2. Or download installer from: https://www.python.org/downloads/
    echo.
    echo IMPORTANT: Make sure to check the box:
    echo            "Add python.exe to PATH"
    echo.
    pause
    exit /b 1
)

echo [OK] Using Python: %PY_CMD%

:: 2. Check for ADB in standard locations if not on PATH
where adb >nul 2>&1
if %errorlevel% neq 0 (
    if exist "%LOCALAPPDATA%\Android\Sdk\platform-tools\adb.exe" (
        set "PATH=%LOCALAPPDATA%\Android\Sdk\platform-tools;!PATH!"
        echo [OK] Found ADB in Android SDK platform-tools
    ) else if exist "C:\platform-tools\adb.exe" (
        set "PATH=C:\platform-tools;!PATH!"
        echo [OK] Found ADB in C:\platform-tools
    ) else (
        echo [INFO] ADB not found on PATH.
        echo [INFO] For USB mode: winget install Google.PlatformTools
        echo [INFO] Continuing (Wi-Fi mode still works without ADB)...
    )
)

:: 3. Check and install dependencies
echo.
echo [*] Checking dependencies from requirements.txt...
%PY_CMD% -m pip install --quiet -r "%SCRIPT_DIR%requirements.txt"
if %errorlevel% neq 0 (
    echo [WARN] Global install failed, trying user install...
    %PY_CMD% -m pip install --quiet --user -r "%SCRIPT_DIR%requirements.txt"
)

:: 4. Start companion server
echo.
echo [*] Starting companion server...
echo.
%PY_CMD% "%SCRIPT_DIR%server.py" %*

if %errorlevel% neq 0 (
    echo.
    echo [ERROR] Server stopped with error code %errorlevel%.
    pause
)
