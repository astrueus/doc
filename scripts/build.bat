@echo off
chcp 65001 >nul
setlocal EnableExtensions EnableDelayedExpansion

REM ============================================================
REM  Doc multi-platform build script (Windows)
REM
REM  Usage: build.bat [target] [mode|version|mingw] [version]
REM
REM  target      : all | linux | windows  (default: all)
REM  mode        : debug | release        (default: debug)
REM  version     : optional, auto-detected from git
REM  win toolchain: mingw | mingw-w64    (default: zig for Windows & Linux)
REM
REM  Examples:
REM    build.bat
REM    build.bat windows
REM    build.bat windows mingw
REM    build.bat windows mingw 1.2.0
REM    build.bat all release
REM    build.bat linux release 2.0.0
REM ============================================================

set "ARG1=%~1"
set "ARG2=%~2"
set "ARG3=%~3"
set "ARG4=%~4"
set "TARGET=all"
set "MODE=debug"
set "VERSION="
set "WIN_TOOLCHAIN=zig"
set "BUILD_OK=1"

if /i "%ARG1%"=="help" goto :usage
if /i "%ARG1%"=="-h" goto :usage
if /i "%ARG1%"=="--help" goto :usage

call :parse_args
if errorlevel 1 exit /b 1

set "ROOT=%~dp0.."
pushd "%ROOT%"

call :resolve_version "%VERSION%"

for /f "tokens=3" %%V in ('go version 2^>nul') do set "GO_VER=%%V"
if not defined GO_VER set "GO_VER=unknown"

for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /value 2^>nul ^| find "="') do set "DT=%%I"
if not defined DT set "DT=unknown"
set "BUILD_TIME=!DT:~0,4!-!DT:~4,2!-!DT:~6,2!T!DT:~8,2!:!DT:~10,2!:!DT:~12,2!"

set "LDFLAGS_COMMON=-X 'git.itopcms.com/jackliu/doc/conf.VERSION=!VERSION!' -X 'git.itopcms.com/jackliu/doc/conf.BUILD_TIME=!BUILD_TIME!' -X 'git.itopcms.com/jackliu/doc/conf.GO_VERSION=!GO_VER!'"

if /i "%MODE%"=="release" (
    if not exist "dist" mkdir "dist"
    set "OUT_LINUX=dist\doc_linux_amd64"
    set "OUT_WINDOWS=dist\doc_windows_amd64.exe"
    set "LDFLAGS=-w -s !LDFLAGS_COMMON!"
) else (
    set "OUT_LINUX=doc"
    set "OUT_WINDOWS=doc.exe"
    set "LDFLAGS=-w !LDFLAGS_COMMON!"
)

echo.
echo ========================================
echo  Doc Build
echo  Mode           : %MODE%
echo  Target         : %TARGET%
echo  Version        : !VERSION!
echo  Build Time     : !BUILD_TIME!
echo  Linux toolchain: Zig
echo  Win toolchain  : !WIN_TOOLCHAIN!
echo ========================================
echo.

where go >nul 2>&1
if errorlevel 1 (
    echo [ERROR] go not found. Please install Go 1.25+
    popd
    exit /b 1
)

where zig >nul 2>&1
if errorlevel 1 (
    if /i "%TARGET%"=="linux" goto :zig_required
    if /i "%TARGET%"=="all" goto :zig_required
    if /i "%TARGET%"=="windows" if /i not "!WIN_TOOLCHAIN!"=="mingw" goto :zig_required
)
goto :after_zig_check

:zig_required
echo [ERROR] zig not found. Install Zig and add it to PATH
echo         https://ziglang.org/download/
popd
exit /b 1

:after_zig_check

echo [INFO] go mod tidy ...
go mod tidy
if errorlevel 1 (
    echo [ERROR] go mod tidy failed
    popd
    exit /b 1
)

if /i "%TARGET%"=="all" (
    call :do_linux
    call :do_windows
) else if /i "%TARGET%"=="linux" (
    call :do_linux
) else if /i "%TARGET%"=="windows" (
    call :do_windows
) else (
    echo [ERROR] unknown target: %TARGET%
    popd
    exit /b 1
)

popd
echo.
if "!BUILD_OK!"=="1" (
    echo [SUCCESS] build completed
    exit /b 0
) else (
    echo [FAILED] one or more targets failed to build
    exit /b 1
)

REM ---------- argument parsing ----------
:parse_args
for %%A in (%*) do call :detect_mingw "%%A"
if "%ARG1%"=="" goto :validate_target

if /i "%ARG1%"=="debug" (
    set "MODE=debug"
    set "TARGET=all"
    call :parse_optional_args "%ARG2%" "%ARG3%" "%ARG4%"
    goto :validate_target
)
if /i "%ARG1%"=="release" (
    set "MODE=release"
    set "TARGET=all"
    call :parse_optional_args "%ARG2%" "%ARG3%" "%ARG4%"
    goto :validate_target
)

if /i "%ARG1%"=="mingw" goto :parse_mingw_first
if /i "%ARG1%"=="mingw-w64" goto :parse_mingw_first

set "TARGET=%ARG1%"
if /i "%TARGET%"=="win" set "TARGET=windows"
call :parse_optional_args "%ARG2%" "%ARG3%" "%ARG4%"
goto :validate_target

:parse_mingw_first
set "TARGET=%ARG2%"
if /i "%TARGET%"=="win" set "TARGET=windows"
call :parse_optional_args "%ARG3%" "%ARG4%" ""
goto :validate_target

:parse_optional_args
if /i "%~1"=="debug" (
    set "MODE=debug"
    call :try_set_version "%~2"
    call :try_set_version "%~3"
    goto :eof
)
if /i "%~1"=="release" (
    set "MODE=release"
    call :try_set_version "%~2"
    call :try_set_version "%~3"
    goto :eof
)
call :try_set_version "%~1"
call :try_set_version "%~2"
call :try_set_version "%~3"
goto :eof

:detect_mingw
if /i "%~1"=="mingw" set "WIN_TOOLCHAIN=mingw"
if /i "%~1"=="mingw-w64" set "WIN_TOOLCHAIN=mingw"
goto :eof

:try_set_version
if "%~1"=="" goto :eof
if /i "%~1"=="mingw" goto :eof
if /i "%~1"=="mingw-w64" goto :eof
if /i "%~1"=="debug" goto :eof
if /i "%~1"=="release" goto :eof
if /i "%~1"=="all" goto :eof
if /i "%~1"=="linux" goto :eof
if /i "%~1"=="windows" goto :eof
if /i "%~1"=="win" goto :eof
call :is_version "%~1"
if not errorlevel 1 set "VERSION=%~1"
goto :eof

:is_version
echo.%~1| findstr /r /c:"^[vV][0-9]" /c:"^[0-9][0-9]*\.[0-9]" >nul
exit /b %errorlevel%

:validate_target
if /i not "%TARGET%"=="all" if /i not "%TARGET%"=="linux" if /i not "%TARGET%"=="windows" (
    echo [ERROR] unknown target: %TARGET%
    exit /b 1
)
if /i not "%MODE%"=="debug" if /i not "%MODE%"=="release" (
    echo [ERROR] unknown mode: %MODE%
    exit /b 1
)
if /i not "!WIN_TOOLCHAIN!"=="zig" if /i not "!WIN_TOOLCHAIN!"=="mingw" (
    echo [ERROR] unknown win toolchain: !WIN_TOOLCHAIN!
    exit /b 1
)
exit /b 0

REM ---------- version resolution ----------
:resolve_version
if not "%~1"=="" (
    set "VERSION=%~1"
    goto :eof
)

set "VERSION="
for /f "delims=" %%V in ('git describe --tags --always --dirty 2^>nul') do set "VERSION=%%V"
if defined VERSION goto :eof

for /f "delims=" %%V in ('git rev-parse --short HEAD 2^>nul') do set "VERSION=%%V"
if defined VERSION goto :eof

set "VERSION=dev"
goto :eof

REM ---------- Linux amd64 (Zig) ----------
:do_linux
echo [BUILD] Linux amd64 -^> !OUT_LINUX! ^(Zig^)

set CGO_ENABLED=1
set GOOS=linux
set GOARCH=amd64
set "CC=zig cc -target x86_64-linux-gnu"

go build -ldflags "!LDFLAGS!" -o "!OUT_LINUX!" .
if errorlevel 1 (
    echo [ERROR] Linux build failed
    set "BUILD_OK=0"
) else (
    echo [OK]    !OUT_LINUX!
)

set CC=
set GOOS=
set GOARCH=
goto :eof

REM ---------- Windows amd64 (Zig or MinGW-w64) ----------
:do_windows
if /i "!WIN_TOOLCHAIN!"=="mingw" (
    echo [BUILD] Windows amd64 -^> !OUT_WINDOWS! ^(MinGW-w64^)
    where gcc >nul 2>&1
    if errorlevel 1 (
        echo [ERROR] gcc not found. Install MinGW-w64 and add its bin to PATH
        set "BUILD_OK=0"
        goto :eof
    )
    set CGO_ENABLED=1
    set GOOS=windows
    set GOARCH=amd64
    set CC=
) else (
    echo [BUILD] Windows amd64 -^> !OUT_WINDOWS! ^(Zig^)
    set CGO_ENABLED=1
    set GOOS=windows
    set GOARCH=amd64
    set "CC=zig cc -target x86_64-windows-gnu"
)

go build -ldflags "!LDFLAGS!" -o "!OUT_WINDOWS!" .
if errorlevel 1 (
    if /i "!WIN_TOOLCHAIN!"=="mingw" (
        echo [ERROR] Windows build failed. Check MinGW-w64 / gcc in PATH
    ) else (
        echo [ERROR] Windows build failed. Check Zig installation
    )
    set "BUILD_OK=0"
) else (
    echo [OK]    !OUT_WINDOWS!
)

set CC=
set GOOS=
set GOARCH=
goto :eof

REM ---------- help ----------
:usage
echo.
echo Usage: build.bat [target] [mode^|version^|mingw] [version]
echo.
echo  target         : all ^| linux ^| windows   (default: all)
echo  mode           : debug ^| release          (default: debug)
echo  version        : optional, auto-detected from git
echo  win toolchain  : mingw ^| mingw-w64       (default: zig)
echo.
echo  Toolchain:
echo    Linux         - always Zig (zig cc -target x86_64-linux-gnu)
echo    Windows       - Zig by default (zig cc -target x86_64-windows-gnu)
echo                    pass mingw / mingw-w64 to use MinGW-w64 gcc instead
echo.
echo  Build modes:
echo    debug   - output to project root: doc / doc.exe
echo    release - output to dist\ with -s strip
echo.
echo  Examples:
echo    build.bat
echo    build.bat windows
echo    build.bat windows mingw
echo    build.bat windows mingw 1.2.0
echo    build.bat mingw windows release
echo    build.bat all release
echo    build.bat linux release 2.0.0
echo.
exit /b 0
