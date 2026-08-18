@echo off
chcp 65001 >nul
setlocal EnableExtensions EnableDelayedExpansion

REM ============================================================
REM  Doc multi-platform build script (Windows)
REM
REM  Usage:
REM    build.bat [--target=all|linux|windows]      (-t)
REM              [--mode=debug|release]            (-m)
REM              [--version=X.Y.Z]                 (-v)
REM              [--toolchain=zig|mingw]           (-x)
REM              [-h|--help]
REM
REM  Defaults:
REM    --target=all
REM    --mode=debug
REM    --toolchain=zig            (only affects Windows target)
REM    --version                  auto-detected via `git describe`
REM
REM  Toolchain:
REM    Linux    Zig (zig cc -target x86_64-linux-gnu, cross-compile)
REM    Windows  Zig by default (zig cc -target x86_64-windows-gnu)
REM             --toolchain=mingw uses MinGW-w64 gcc instead
REM             (alias: mingw-w64)
REM
REM  Build modes:
REM    debug    output to project root: doc / doc.exe
REM    release  output to dist\ with -s strip
REM
REM  Examples:
REM    build.bat
REM    build.bat --target=windows
REM    build.bat --target=windows --toolchain=mingw
REM    build.bat --target=windows --toolchain=mingw --version=1.2.0
REM    build.bat --mode=release
REM    build.bat -t linux -m release -v 2.0.0
REM    build.bat -t windows -x mingw -v 1.2.0
REM ============================================================

set "TARGET=all"
set "MODE=debug"
set "VERSION="
set "TOOLCHAIN=zig"
set "BUILD_OK=1"

REM ---------- parse args (支持 --key=value / --key value / -k=value / -k value) ----------
:parse_loop
if "%~1"=="" goto :parse_done

set "ARG=%~1"
if /i "%ARG%"=="-h"     goto :usage
if /i "%ARG%"=="--help" goto :usage
if /i "%ARG%"=="help"   goto :usage

set "KEY="
set "VAL="
set "CONSUME_NEXT=0"

for /f "tokens=1,* delims==" %%A in ("%ARG%") do (
    set "KEY=%%A"
    set "VAL=%%B"
)

REM 短标签归一化到长标签
if /i "!KEY!"=="-t" set "KEY=--target"
if /i "!KEY!"=="-m" set "KEY=--mode"
if /i "!KEY!"=="-v" set "KEY=--version"
if /i "!KEY!"=="-x" set "KEY=--toolchain"

REM `--key value` / `-k value`：VAL 为空且下一个参数不是 flag
if "!VAL!"=="" (
    set "NEXT=%~2"
    if defined NEXT (
        set "NEXT_FIRST=!NEXT:~0,1!"
        if not "!NEXT_FIRST!"=="-" (
            set "VAL=%~2"
            set "CONSUME_NEXT=1"
        )
    )
)

if /i "!KEY!"=="--target" (
    if "!VAL!"=="" ( echo [ERROR] --target/-t requires a value & exit /b 1 )
    set "TARGET=!VAL!"
    if /i "!TARGET!"=="win" set "TARGET=windows"
) else if /i "!KEY!"=="--mode" (
    if "!VAL!"=="" ( echo [ERROR] --mode/-m requires a value & exit /b 1 )
    set "MODE=!VAL!"
) else if /i "!KEY!"=="--version" (
    if "!VAL!"=="" ( echo [ERROR] --version/-v requires a value & exit /b 1 )
    set "VERSION=!VAL!"
) else if /i "!KEY!"=="--toolchain" (
    if "!VAL!"=="" ( echo [ERROR] --toolchain/-x requires a value & exit /b 1 )
    set "TOOLCHAIN=!VAL!"
    if /i "!TOOLCHAIN!"=="mingw-w64" set "TOOLCHAIN=mingw"
) else (
    echo [ERROR] unknown option: !KEY!
    exit /b 1
)

shift
if "!CONSUME_NEXT!"=="1" shift
goto :parse_loop

:parse_done

REM ---------- validate ----------
if /i not "%TARGET%"=="all" if /i not "%TARGET%"=="linux" if /i not "%TARGET%"=="windows" (
    echo [ERROR] unknown target: %TARGET% ^(expect all^|linux^|windows^)
    exit /b 1
)
if /i not "%MODE%"=="debug" if /i not "%MODE%"=="release" (
    echo [ERROR] unknown mode: %MODE% ^(expect debug^|release^)
    exit /b 1
)
if /i not "!TOOLCHAIN!"=="zig" if /i not "!TOOLCHAIN!"=="mingw" (
    echo [ERROR] unknown toolchain: !TOOLCHAIN! ^(expect zig^|mingw^)
    exit /b 1
)

REM Resolve repo root: prefer directory that contains go.mod + cmd\doc
set "SCRIPT_DIR=%~dp0"
set "ROOT="
if exist "%SCRIPT_DIR%go.mod" if exist "%SCRIPT_DIR%cmd\doc" set "ROOT=%SCRIPT_DIR%"
if not defined ROOT if exist "%SCRIPT_DIR%..\go.mod" if exist "%SCRIPT_DIR%..\cmd\doc" (
    for %%I in ("%SCRIPT_DIR%..") do set "ROOT=%%~fI"
)
if not defined ROOT if exist "%SCRIPT_DIR%..\..\go.mod" if exist "%SCRIPT_DIR%..\..\cmd\doc" (
    for %%I in ("%SCRIPT_DIR%..\..") do set "ROOT=%%~fI"
)
if not defined ROOT if exist "go.mod" if exist "cmd\doc" set "ROOT=%CD%"
if not defined ROOT (
    echo [ERROR] cannot locate repo root ^(need go.mod and cmd\doc^)
    echo         script dir: %SCRIPT_DIR%
    echo         cwd: %CD%
    exit /b 1
)
pushd "%ROOT%"
if not exist "go.mod" (
    echo [ERROR] go.mod not found under: %ROOT%
    popd
    exit /b 1
)

call :resolve_version "%VERSION%"

for /f "tokens=3" %%V in ('go version 2^>nul') do set "GO_VER=%%V"
if not defined GO_VER set "GO_VER=unknown"

for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /value 2^>nul ^| find "="') do set "DT=%%I"
if not defined DT set "DT=unknown"
set "BUILD_TIME=!DT:~0,4!-!DT:~4,2!-!DT:~6,2!T!DT:~8,2!:!DT:~10,2!:!DT:~12,2!"

set "LDFLAGS_COMMON=-X 'git.itopcms.com/astrueus/doc/internal/config.VERSION=!VERSION!' -X 'git.itopcms.com/astrueus/doc/internal/config.BUILD_TIME=!BUILD_TIME!' -X 'git.itopcms.com/astrueus/doc/internal/config.GO_VERSION=!GO_VER!'"

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
echo  Win toolchain  : !TOOLCHAIN!
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
    if /i "%TARGET%"=="windows" if /i not "!TOOLCHAIN!"=="mingw" goto :zig_required
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
go build -ldflags "!LDFLAGS!" -o "!OUT_LINUX!" ./cmd/doc
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
if /i "!TOOLCHAIN!"=="mingw" (
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

go build -ldflags "!LDFLAGS!" -o "!OUT_WINDOWS!" ./cmd/doc
if errorlevel 1 (
    if /i "!TOOLCHAIN!"=="mingw" (
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
echo Usage: build.bat [--target=all^|linux^|windows]      (-t)
echo                  [--mode=debug^|release]            (-m)
echo                  [--version=X.Y.Z]                 (-v)
echo                  [--toolchain=zig^|mingw]           (-x)
echo                  [-h^|--help]
echo.
echo Defaults:
echo   --target=all
echo   --mode=debug
echo   --toolchain=zig            (only affects Windows target)
echo   --version                  auto-detected via `git describe`
echo.
echo Toolchain:
echo   Linux    Zig (zig cc -target x86_64-linux-gnu, cross-compile)
echo   Windows  Zig by default (zig cc -target x86_64-windows-gnu)
echo            --toolchain=mingw uses MinGW-w64 gcc instead
echo            (alias: mingw-w64)
echo.
echo Build modes:
echo   debug    output to project root: doc / doc.exe
echo   release  output to dist\ with -s strip
echo.
echo All flags support four forms: --key=value, --key value, -k=value, -k value.
echo.
echo Examples:
echo   build.bat
echo   build.bat --target=windows
echo   build.bat --target=windows --toolchain=mingw
echo   build.bat --target=windows --toolchain=mingw --version=1.2.0
echo   build.bat --mode=release
echo   build.bat -t linux -m release -v 2.0.0
echo   build.bat -t windows -x mingw -v 1.2.0
echo.
exit /b 0
