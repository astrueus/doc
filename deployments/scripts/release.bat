@echo off
setlocal EnableExtensions EnableDelayedExpansion

REM Usage:
REM   deployments\scripts\release.bat <version> [all^|linux^|windows] [options...]
REM Options:
REM   --env=PATH     env file (default: deployments\scripts\.env.release if present)
REM   --draft        create draft release
REM   --dry-run      build+zip only
REM   --skip-tag     skip git tag/push
REM
REM Examples:
REM   deployments\scripts\release.bat 0.0.1-test windows --dry-run
REM   deployments\scripts\release.bat 0.0.1-test windows --env=deployments\scripts\.env.release --draft

REM 必须在任何 shift 之前保存：shift 后 %~dp0 会变成当前目录
set "SCRIPT_DIR=%~dp0"
set "PS1=%SCRIPT_DIR%release.ps1"

if "%~1"=="" (
  echo Usage: release.bat ^<version^> [all^|linux^|windows] [--env=PATH] [--draft] [--dry-run] [--skip-tag]
  exit /b 1
)

set "VERSION=%~1"
shift

set "TARGET=windows"
set "ENVFILE="
set "DRAFT=0"
set "DRYRUN=0"
set "SKIPTAG=0"

:parse
if "%~1"=="" goto :run

set "ARG=%~1"
if /i "!ARG!"=="all" set "TARGET=all" & shift & goto :parse
if /i "!ARG!"=="linux" set "TARGET=linux" & shift & goto :parse
if /i "!ARG!"=="windows" set "TARGET=windows" & shift & goto :parse
if /i "!ARG!"=="--draft" set "DRAFT=1" & shift & goto :parse
if /i "!ARG!"=="--dry-run" set "DRYRUN=1" & shift & goto :parse
if /i "!ARG!"=="--skip-tag" set "SKIPTAG=1" & shift & goto :parse

echo !ARG!| findstr /b /i /c:"--env=" >nul
if not errorlevel 1 (
  set "ENVFILE=!ARG:~6!"
  shift
  goto :parse
)

echo [ERROR] unknown argument: !ARG!
exit /b 1

:run
if not exist "%PS1%" (
  echo [ERROR] release.ps1 not found: %PS1%
  echo         SCRIPT_DIR was: %SCRIPT_DIR%
  exit /b 1
)

chcp 65001 >nul

set "SWITCHES="
if "!DRAFT!"=="1" set "SWITCHES=!SWITCHES! -Draft"
if "!DRYRUN!"=="1" set "SWITCHES=!SWITCHES! -DryRun"
if "!SKIPTAG!"=="1" set "SWITCHES=!SWITCHES! -SkipTagPush"

if defined ENVFILE (
  powershell -NoProfile -ExecutionPolicy Bypass -File "%PS1%" -Version "%VERSION%" -Target "%TARGET%" -EnvFile "!ENVFILE!" !SWITCHES!
) else (
  powershell -NoProfile -ExecutionPolicy Bypass -File "%PS1%" -Version "%VERSION%" -Target "%TARGET%" !SWITCHES!
)
exit /b !ERRORLEVEL!
