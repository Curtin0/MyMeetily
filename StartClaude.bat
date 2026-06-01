@echo off
title Claude Code - Full Access Mode
color 0B

:: ================================================================
::  Claude Code Launcher - Full Permission Mode
::  Double-click this .bat to start Claude Code with ALL permissions
::  granted for the current directory. No more approval prompts.
:: ================================================================

cd /d "%~dp0"

:: Ensure .claude directory exists
if not exist ".claude" mkdir ".claude"

:: Write full-permission settings.json only if it doesn't already exist
if not exist ".claude\settings.json" (
    (
    echo {
    echo   "permissions": {
    echo     "allow": [
    echo       "Bash(*:*)" ,
    echo       "Bash(*)",
    echo       "Read(*)",
    echo       "Write(*)",
    echo       "Edit(*)",
    echo       "Glob(*)",
    echo       "Grep(*)",
    echo       "WebFetch(*)",
    echo       "WebSearch(*)",
    echo       "Skill(*)",
    echo       "Agent(*)",
    echo       "Task(*)",
    echo       "AskUserQuestion(*)"
    echo     ]
    echo   }
    echo }
    ) > ".claude\settings.json"
)

:: Remove any local overrides that might restrict permissions
if exist ".claude\settings.local.json" del ".claude\settings.local.json"

echo.
echo ================================================================
echo   Claude Code - Full Access Mode
echo ================================================================
echo.
echo   All permissions granted for: %cd%
echo   No approval prompts will be shown.
echo.
echo   Starting Claude Code...
echo ================================================================
echo.

:: Launch Claude Code
claude
