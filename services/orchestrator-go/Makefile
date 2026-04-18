SHELL := /bin/sh

APP_NAME ?= speakerfocusd
CMD ?= ./cmd/speakerfocusd
BUILD_DIR ?= ./bin
BIN ?= $(BUILD_DIR)/$(APP_NAME)

GO ?= go
GOOS ?=
GOARCH ?=
CGO_ENABLED ?=
TAGS ?=
LDFLAGS ?=
ARGS ?=

INPUT ?= test.wav
OUTPUT ?= output.wav
CHUNK_MS ?= 20
METRICS_ADDR ?= 127.0.0.1:2112
METRICS_PATH ?= /metrics
REALTIME ?= false
INPUT_LOOPS ?= 1
DISCARD_OUTPUT ?= false
AUDIO_CORE_ADDR ?=

GO_ENV := GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED)
GO_BUILD_FLAGS :=
ifneq ($(strip $(TAGS)),)
GO_BUILD_FLAGS += -tags "$(TAGS)"
endif
ifneq ($(strip $(LDFLAGS)),)
GO_BUILD_FLAGS += -ldflags "$(LDFLAGS)"
endif

.PHONY: help test build run run-bin tidy clean

help:
	@printf '%s\n' 'Go service targets:'
	@printf '%s\n' '  make test      Run Go tests'
	@printf '%s\n' '  make build     Build speakerfocusd into ./bin'
	@printf '%s\n' '  make run       Run with go run'
	@printf '%s\n' '  make run-bin   Build and run the binary'
	@printf '%s\n' '  make tidy      Run go mod tidy'
	@printf '%s\n' '  make clean     Remove build output'
	@printf '%s\n' ''
	@printf '%s\n' 'Common variables:'
	@printf '%s\n' '  INPUT=test.wav OUTPUT=output.wav CHUNK_MS=20 METRICS_ADDR=127.0.0.1:2112'
	@printf '%s\n' '  REALTIME=false INPUT_LOOPS=1 DISCARD_OUTPUT=false AUDIO_CORE_ADDR='
	@printf '%s\n' '  GOOS=linux GOARCH=arm64 CGO_ENABLED=0 BUILD_DIR=./bin LDFLAGS=... TAGS=...'
	@printf '%s\n' '  ARGS="extra flags"'

test:
	$(GO) test ./...

build:
	mkdir -p $(BUILD_DIR)
	$(GO_ENV) $(GO) build $(GO_BUILD_FLAGS) -o $(BIN) $(CMD)

run:
	$(GO) run $(CMD) \
		-input "$(INPUT)" \
		-output "$(OUTPUT)" \
		-chunk-ms "$(CHUNK_MS)" \
		-metrics-addr "$(METRICS_ADDR)" \
		-metrics-path "$(METRICS_PATH)" \
		-realtime="$(REALTIME)" \
		-input-loops="$(INPUT_LOOPS)" \
		-discard-output="$(DISCARD_OUTPUT)" \
		-audio-core-addr="$(AUDIO_CORE_ADDR)" \
		$(ARGS)

run-bin: build
	$(BIN) \
		-input "$(INPUT)" \
		-output "$(OUTPUT)" \
		-chunk-ms "$(CHUNK_MS)" \
		-metrics-addr "$(METRICS_ADDR)" \
		-metrics-path "$(METRICS_PATH)" \
		-realtime="$(REALTIME)" \
		-input-loops="$(INPUT_LOOPS)" \
		-discard-output="$(DISCARD_OUTPUT)" \
		-audio-core-addr="$(AUDIO_CORE_ADDR)" \
		$(ARGS)

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BUILD_DIR)
