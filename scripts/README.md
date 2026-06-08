# 构建脚本

本地构建脚本，支持调试与发布两种模式，以及 Linux / Windows 多平台编译。

| 脚本 | 平台 | 说明 |
|------|------|------|
| `build.bat` | Windows | Linux 交叉编译使用 Zig |
| `build.sh` | Linux / macOS | Linux 本机构建使用系统 gcc/clang |

编译 Windows 时若显式传入 `mingw` 或 `mingw-w64`，则改用 **MinGW-w64** 交叉编译器。

## 前置依赖

### Windows（`build.bat`）

| 目标平台 | 默认工具链 | 依赖 |
|----------|------------|------|
| Windows amd64 | Zig | Go 1.25+、[Zig](https://ziglang.org/download/) |
| Linux amd64（交叉编译） | Zig | 同上 |
| Windows amd64（显式 `mingw`） | MinGW-w64 | Go 1.25+、MinGW-w64（`gcc` 在 PATH 中） |

### Linux / macOS（`build.sh`）

| 目标平台 | 默认工具链 | 依赖 |
|----------|------------|------|
| Linux amd64（本机） | 系统 gcc/clang | Go 1.25+、`build-essential`（或等价 C 编译器） |
| Windows amd64 | Zig | Go 1.25+、[Zig](https://ziglang.org/download/) |
| Windows amd64（显式 `mingw`） | MinGW-w64 | Go 1.25+、`mingw-w64`（`x86_64-w64-mingw32-gcc`） |

Linux 安装示例：

```bash
# 本机构建 Linux
sudo apt install build-essential golang

# 交叉编译 Windows（二选一）
sudo apt install zig                    # Zig
sudo apt install mingw-w64              # MinGW-w64
```

## 用法

在项目根目录执行：

```bash
# Linux / macOS
./scripts/build.sh [target] [mode|version|mingw] [version]
```

```bat
REM Windows
scripts\build.bat [target] [mode|version|mingw] [version]
```

### 参数说明

| 参数 | 可选值 | 默认值 | 说明 |
|------|--------|--------|------|
| `target` | `all` / `linux` / `windows` | `all` | 构建目标平台 |
| `mode` | `debug` / `release` | `debug` | 构建模式 |
| `version` | 任意字符串 | 自动获取 | 写入二进制的版本号 |
| Windows 工具链 | `mingw` / `mingw-w64` | Zig | 仅影响 Windows 构建 |

### 工具链说明

| 平台 | Windows (`build.bat`) | Linux (`build.sh`) |
|------|----------------------|-------------------|
| Linux | `zig cc -target x86_64-linux-gnu`（交叉编译） | 本机 gcc/clang（无需 Zig） |
| Windows | `zig cc -target x86_64-windows-gnu` | 同上 |
| Windows（`mingw`） | PATH 中的 `gcc` | `x86_64-w64-mingw32-gcc` |

### 构建模式

| 项目 | debug（默认） | release |
|------|---------------|---------|
| 输出位置 | 项目根目录 `doc` / `doc.exe` | `dist/doc_linux_amd64`、`dist/doc_windows_amd64.exe` |
| `go mod tidy` | 执行 | 执行 |
| 版本注入 | 是（自动或传参） | 是（自动或传参） |
| ldflags | `-w` + 版本信息 | `-w -s` + 版本信息 |

### 常用示例

```bash
# Linux：本机构建 Linux + 交叉编译 Windows（需 Zig）
./scripts/build.sh

# 只构建 Linux（仅需 gcc/clang）
./scripts/build.sh linux

# 使用 MinGW-w64 交叉编译 Windows
./scripts/build.sh windows mingw

# 发布构建
./scripts/build.sh all release
./scripts/build.sh linux release 2.0.0

# 查看帮助
./scripts/build.sh help
```

```bat
REM Windows：默认 Zig 构建 Linux + Windows
scripts\build.bat

REM 只构建 Windows
scripts\build.bat windows

REM 使用 MinGW-w64 构建 Windows
scripts\build.bat windows mingw

REM 发布构建
scripts\build.bat all release
scripts\build.bat windows mingw release 1.0.0

REM 查看帮助
scripts\build.bat help
```

### 验证构建结果

```bash
# Linux
./doc version
```

```bat
REM Windows
doc.exe version
```

## 说明

- 首次使用 `build.sh` 需赋予执行权限：`chmod +x scripts/build.sh`
- 调试模式产物 `doc` / `doc.exe` 已在 `.gitignore` 中忽略。
- 发布模式产物输出到 `dist/` 目录。
- Windows 上若不使用 Zig，仅编 Windows 且已安装 MinGW-w64：`scripts\build.bat windows mingw`
- Linux 上若不使用 Zig，可只编本机 Linux：`./scripts/build.sh linux`
- 也可用项目根目录 `Dockerfile` 构建 Linux 镜像。
