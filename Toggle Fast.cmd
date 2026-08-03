@echo off
setlocal
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\Toggle-Fast.ps1"
set "TOGGLE_EXIT=%ERRORLEVEL%"
echo.
if not "%TOGGLE_EXIT%"=="0" echo Fast mode could not be fully verified. Review the message above.
pause
exit /b %TOGGLE_EXIT%
