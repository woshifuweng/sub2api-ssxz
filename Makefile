.PHONY: build build-backend build-frontend build-datamanagementd test test-backend test-frontend test-datamanagementd secret-scan security-preflight docker-build docker-build-compat

# 一键编译前后端
build: build-backend build-frontend

# 编译后端（复用 backend/Makefile）
build-backend:
	@$(MAKE) -C backend build

# 编译前端（需要已安装依赖）
build-frontend:
	@pnpm --dir frontend run build

# 运行测试（后端 + 前端）
test: test-backend test-frontend

test-backend:
	@$(MAKE) -C backend test

test-frontend:
	@pnpm --dir frontend run lint:check
	@pnpm --dir frontend run typecheck
	@$(MAKE) test-frontend-critical

test-datamanagementd:
	@cd datamanagement && go test ./...

secret-scan:
	@python3 tools/secret_scan.py

# 发布前安全闸门（Windows/PowerShell 构建机）
security-preflight:
	@powershell -NoProfile -ExecutionPolicy Bypass -File tools/security_preflight.ps1

docker-build:
	@./deploy/build_image.sh

docker-build-compat:
	@./deploy/build_compat_image.sh
