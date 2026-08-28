# 根目录快捷入口。权威脚本在 deployments/scripts/，这里只转发。
# Windows 更适合用 justfile。

.PHONY: help build test run release

SCRIPTS := deployments/scripts

help:
	@echo "make build                 编译"
	@echo "make test                  白名单单测 + 覆盖率门槛"
	@echo "make run                   开发启动（go run，不落盘二进制）"
	@echo "make run ARGS=install      开发启动并传子命令"
	@echo "make release VERSION=x.y.z 发版（编译、打包、可选打 tag）"

build:
	bash $(SCRIPTS)/build.sh

test:
	bash $(SCRIPTS)/test.sh

run:
	bash $(SCRIPTS)/run.sh $(ARGS)

release:
	@test -n "$(VERSION)" || (echo "用法: make release VERSION=x.y.z" >&2; exit 1)
	bash $(SCRIPTS)/release.sh "$(VERSION)"
