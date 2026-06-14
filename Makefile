.PHONY: build test test-cover lint clean init

# 构建
build:
	go build -o bin/xianyuapis ./cmd/demo/

# 测试
test:
	go test -race -count=1 ./pkg/...

# 测试覆盖率
test-cover:
	go test -race -count=1 -coverprofile=coverage.out ./pkg/...
	go tool cover -html=coverage.out -o coverage.html

# 静态分析
lint:
	golangci-lint run ./...

# 清理
clean:
	if exist bin rmdir /s /q bin
	if exist coverage.out del coverage.out
	if exist coverage.html del coverage.html

# 初始化（首次使用）
init:
	go mod tidy
	go build ./...
