# 构建脚本

Windows 下的本地构建脚本，支持调试与发布两种模式，以及 Linux / Windows 多平台编译。

默认使用 **Zig** 作为 C 工具链（Windows 与 Linux 均支持）。编译 Windows 时若显式传入 `mingw` 或 `mingw-w64`，则改用 **MinGW-w64**（`gcc`）。

## 前置依赖

| 目标平台 | 默认工具链 | 依赖 |
|----------|------------|------|
| Windows amd64 | Zig | Go 1.25+、[Zig](https://ziglang.org/download/) |
| Linux amd64（Windows 上交叉编译） | Zig | 同上 |
| Windows amd64（显式 `mingw`） | MinGW-w64 | Go 1.25+、MinGW-w64（`gcc` 在 PATH 中） |

## 用法

在项目根目录执行：

```bat
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

| 平台 | 默认 | 显式指定 |
|------|------|----------|
| Linux | `zig cc -target x86_64-linux-gnu` | 不支持切换（始终 Zig） |
| Windows | `zig cc -target x86_64-windows-gnu` | 传 `mingw` 使用 PATH 中的 `gcc`（MinGW-w64） |

### 构建模式

| 项目 | debug（默认） | release |
|------|---------------|---------|
| 输出位置 | 项目根目录 `doc` / `doc.exe` | `dist/doc_linux_amd64`、`dist/doc_windows_amd64.exe` |
| `go mod tidy` | 执行 | 执行 |
| 版本注入 | 是（自动或传参） | 是（自动或传参） |
| ldflags | `-w` + 版本信息 | `-w -s` + 版本信息 |

### 常用示例

```bat
REM 默认 Zig：构建 Linux + Windows
scripts\build.bat

REM 默认 Zig：只构建 Windows
scripts\build.bat windows

REM 使用 MinGW-w64 构建 Windows
scripts\build.bat windows mingw

REM Zig 构建 Windows，指定版本
scripts\build.bat windows 1.2.0

REM MinGW-w64 构建 Windows，指定版本
scripts\build.bat windows mingw 1.2.0

REM 发布构建
scripts\build.bat all release
scripts\build.bat windows mingw release 1.0.0

REM 查看帮助
scripts\build.bat help
```

### 验证构建结果

```bat
doc.exe version
```

Linux 产物 `doc` 需在 Linux 环境或 WSL 中运行 `./doc version`。

## 说明

- 调试模式产物 `doc` / `doc.exe` 已在 `.gitignore` 中忽略。
- 发布模式产物输出到 `dist/` 目录。
- 若不使用 Zig，仅编 Windows 且已安装 MinGW-w64：`scripts\build.bat windows mingw`
- 也可用项目根目录 `Dockerfile` 构建 Linux 镜像。
