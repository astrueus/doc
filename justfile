# 根目录快捷入口（跨平台）。权威脚本在 deployments/scripts/，这里只转发。
# 安装：https://github.com/casey/just

set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

help:
    @echo "just build / just test / just release 1.0.0"

# Windows 走 bat/ps1，其它系统走 bash。
build:
    {{ if os() == "windows" { "cmd /c deployments\\scripts\\build.bat" } else { "bash deployments/scripts/build.sh" } }}

test:
    {{ if os() == "windows" { "powershell -NoProfile -ExecutionPolicy Bypass -File deployments/scripts/test.ps1" } else { "bash deployments/scripts/test.sh" } }}

release version:
    {{ if os() == "windows" { "powershell -NoProfile -ExecutionPolicy Bypass -File deployments/scripts/release.ps1 -Version" } else { "bash deployments/scripts/release.sh" } }} {{ version }}
