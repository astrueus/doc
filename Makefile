# 根目录快捷入口。权威脚本在 deployments/scripts/，这里只转发。
# Windows 更适合用 justfile。

.PHONY: help build test release

SCRIPTS := deployments/scripts

help:
	@echo "make build                 编译"
	@echo "make test                  白名单单测 + 覆盖率门槛"
	@echo "make release VERSION=x.y.z 发版（编译、打包、可选打 tag）"

build:
	bash $(SCRIPTS)/build.sh

test:
	bash $(SCRIPTS)/test.sh

release:
	@test -n "$(VERSION)" || (echo "用法: make release VERSION=x.y.z" >&2; exit 1)
	bash $(SCRIPTS)/release.sh "$(VERSION)"
