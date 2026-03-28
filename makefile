# Directories
BUILD_DIR := ./build
COVERAGE_DIR := ./go_test

# Binary output paths
BS_OUT := $(BUILD_DIR)/server

# Source files
SRC_SERVER := ./cmd/main/main.go

# Database DSN
DSN_DB := postgres://postgres:admin54321localhost:5678/postgres

# Coverage files
COVERAGE_OUT := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html

# Build tags
BUILD_VERSION="1.2.3"
# BUILD_DATE=today_hehe
# Попробуем вытащить из шелла
BUILD_DATE    := $(shell date -u '+%Y-%m-%d %H:%M:%S')
BUILD_COMMIT  := $(shell git rev-parse HEAD)

recreate_build_dir:
	echo "Cleaning build directory..."
	rm -rf $(BUILD_DIR)
	mkdir -p $(BUILD_DIR)/server_out
	mkdir -p $(BUILD_DIR)/agent_out
	echo "Build directory prepared"


recreate_coverage_dir:
	echo "Cleaning coverage directory..."
	rm -rf $(COVERAGE_DIR)
	mkdir -p $(COVERAGE_DIR)
	echo "Coverage directory prepared"

recreate_dirs: recreate_build_dir recreate_coverage_dir
	echo "recreate directories"


build_server:
	echo "Building server..."
	go build \
		-ldflags "\
			-X 'main.buildVersion=$(BUILD_VERSION)' \
			-X 'main.buildDate=\"$(BUILD_DATE)\"' \
			-X 'main.buildCommit=\"$(BUILD_COMMIT)\"' \
		" \
		-o $(BS_OUT) $(SRC_SERVER)
	echo "Server built: $(BS_OUT)"


build_all: build_server
	echo "Build complete: server"

# Clean all artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	echo "Clean completed: removed $(BUILD_DIR) and $(COVERAGE_DIR)"


test_coverage: recreate_coverage_dir
	echo "Running tests with coverage..."
	go test ./... -coverprofile=$(COVERAGE_OUT) -covermode=atomic
	echo "Generating coverage report..."
	go tool cover -func=$(COVERAGE_OUT)
	go tool cover -html=$(COVERAGE_OUT) -o=$(COVERAGE_HTML)
	echo "Coverage report generated: $(COVERAGE_HTML)"


# Тесты
test_integration:
	go test -v -tags=integration ./...

test_local:
	go test -v ./...

test_all: test_local test_integration
	echo "AllTested"