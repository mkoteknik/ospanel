.PHONY: build build-frontend build-backend clean dev dev-backend dev-frontend install test lint release

# Proje bilgileri
APP_NAME    := ospanel
VERSION    ?= 0.1.0-dev
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Dizinler
BUILD_DIR  := build
WEB_DIR    := web
BACKEND_DIR := cmd/ospanel

# Go build flags
LDFLAGS := -s -w \
	-X 'main.Version=$(VERSION)' \
	-X 'main.BuildDate=$(BUILD_DATE)' \
	-X 'main.GitCommit=$(GIT_COMMIT)'

# Varsayılan hedef: tümünü build et
all: build

# === Geliştirme ===

dev-backend:
	go run ./cmd/ospanel/ --config ./installer/templates/config.dev.yaml

dev-frontend:
	cd $(WEB_DIR) && npm run dev

dev:
	@echo "Ayrı terminallerde çalıştırın:"
	@echo "  Terminal 1: make dev-backend"
	@echo "  Terminal 2: make dev-frontend"

# === Build ===

build-frontend:
	cd $(WEB_DIR) && npm install && npm run build

build-backend:
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./$(BACKEND_DIR)/

build: build-frontend build-backend
	@echo "✓ Binary: $(BUILD_DIR)/$(APP_NAME)"
	@ls -lh $(BUILD_DIR)/$(APP_NAME)

build-linux-amd64:
	cd $(WEB_DIR) && npm install && npm run build
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./$(BACKEND_DIR)/

build-linux-arm64:
	cd $(WEB_DIR) && npm install && npm run build
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 ./$(BACKEND_DIR)/

build-all: build-linux-amd64 build-linux-arm64
	@echo "✓ Tüm platformlar için build tamamlandı"

# === Temizlik ===

clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(BACKEND_DIR)/web-dist
	rm -f $(APP_NAME).exe

clean-all: clean
	cd $(WEB_DIR) && rm -rf node_modules dist

# === Test ===

test:
	go test ./... -v -count=1 -timeout 60s

test-race:
	go test ./... -race -count=1 -timeout 120s

test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage raporu: coverage.html"

# === Lint ===

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run ./... --fix

# === Release ===

release: clean build-all
	@mkdir -p $(BUILD_DIR)/release
	cd $(BUILD_DIR) && tar czf release/$(APP_NAME)-linux-amd64.tar.gz $(APP_NAME)-linux-amd64
	cd $(BUILD_DIR) && tar czf release/$(APP_NAME)-linux-arm64.tar.gz $(APP_NAME)-linux-arm64
	@echo "✓ Release paketleri: $(BUILD_DIR)/release/"

# === Kurulum ===

install:
	cp $(BUILD_DIR)/$(APP_NAME) /opt/ospanel/ospanel
	chmod +x /opt/ospanel/ospanel
	cp installer/ospanel.service /etc/systemd/system/
	systemctl daemon-reload

uninstall:
	systemctl stop ospanel 2>/dev/null || true
	systemctl disable ospanel 2>/dev/null || true
	rm -f /etc/systemd/system/ospanel.service
	rm -rf /opt/ospanel
	@echo "OpenSpeed Panel kaldırıldı. Veri dizini korundu: /var/lib/ospanel"
