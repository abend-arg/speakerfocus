SHELL := /bin/sh

GO_SERVICE_DIR := services/orchestrator-go
OBS_DIR := dev/observability
AUDIO_CORE_DIR := services/audio-core-python
PROTO_DIR := proto

MAKE ?= make
PROTOC ?= protoc

INPUT ?= $(GO_SERVICE_DIR)/test.wav
OUTPUT ?= $(GO_SERVICE_DIR)/output.wav
CHUNK_MS ?= 20
METRICS_ADDR ?= 0.0.0.0:2112
METRICS_PATH ?= /metrics
REALTIME ?= false
INPUT_LOOPS ?= 1
DISCARD_OUTPUT ?= false
AUDIO_CORE_ADDR ?=
AUDIO_CORE_BIND ?= 127.0.0.1:50051
AUDIO_CORE_VAD_BACKEND ?= silero
ARGS ?=

.PHONY: help test build run run-bin tidy clean dev proto audio-core audio-core-sync obs-up obs-down obs-restart obs-logs obs-ps obs-config

help:
	@printf '%s\n' 'Root targets:'
	@printf '%s\n' '  make test          Run all tests'
	@printf '%s\n' '  make build         Build Go daemon'
	@printf '%s\n' '  make run           Run Go daemon'
	@printf '%s\n' '  make run-bin       Build and run Go daemon'
	@printf '%s\n' '  make dev           Start observability, audio core, and daemon'
	@printf '%s\n' '  make proto         Regenerate gRPC stubs'
	@printf '%s\n' '  make audio-core    Run Python audio core gRPC service'
	@printf '%s\n' '  make audio-core-sync Install Python audio core dependencies'
	@printf '%s\n' '  make obs-up        Start Prometheus and Grafana'
	@printf '%s\n' '  make obs-down      Stop Prometheus and Grafana'
	@printf '%s\n' '  make obs-logs      Follow Prometheus/Grafana logs'
	@printf '%s\n' '  make tidy          Run go mod tidy'
	@printf '%s\n' '  make clean         Remove Go build output'
	@printf '%s\n' ''
	@printf '%s\n' 'Common variables:'
	@printf '%s\n' '  INPUT=services/orchestrator-go/test.wav OUTPUT=services/orchestrator-go/output.wav'
	@printf '%s\n' '  CHUNK_MS=20 METRICS_ADDR=0.0.0.0:2112 METRICS_PATH=/metrics ARGS="..."'
	@printf '%s\n' '  REALTIME=false INPUT_LOOPS=1 DISCARD_OUTPUT=false AUDIO_CORE_ADDR='
	@printf '%s\n' '  AUDIO_CORE_BIND=127.0.0.1:50051 AUDIO_CORE_VAD_BACKEND=silero'
	@printf '%s\n' '  GOOS=linux GOARCH=arm64 CGO_ENABLED=0 LDFLAGS=... TAGS=...'

test:
	$(MAKE) -C $(GO_SERVICE_DIR) test

build:
	$(MAKE) -C $(GO_SERVICE_DIR) build

run:
	$(MAKE) -C $(GO_SERVICE_DIR) run \
		INPUT='$(abspath $(INPUT))' \
		OUTPUT='$(abspath $(OUTPUT))' \
		CHUNK_MS=$(CHUNK_MS) \
		METRICS_ADDR='$(METRICS_ADDR)' \
		METRICS_PATH='$(METRICS_PATH)' \
		REALTIME='$(REALTIME)' \
		INPUT_LOOPS='$(INPUT_LOOPS)' \
		DISCARD_OUTPUT='$(DISCARD_OUTPUT)' \
		AUDIO_CORE_ADDR='$(AUDIO_CORE_ADDR)' \
		ARGS='$(ARGS)'

run-bin:
	$(MAKE) -C $(GO_SERVICE_DIR) run-bin \
		INPUT='$(abspath $(INPUT))' \
		OUTPUT='$(abspath $(OUTPUT))' \
		CHUNK_MS=$(CHUNK_MS) \
		METRICS_ADDR='$(METRICS_ADDR)' \
		METRICS_PATH='$(METRICS_PATH)' \
		REALTIME='$(REALTIME)' \
		INPUT_LOOPS='$(INPUT_LOOPS)' \
		DISCARD_OUTPUT='$(DISCARD_OUTPUT)' \
		AUDIO_CORE_ADDR='$(AUDIO_CORE_ADDR)' \
		ARGS='$(ARGS)'

dev: obs-up
	@set -e; \
	printf '%s\n' 'Audio core: $(AUDIO_CORE_BIND) ($(AUDIO_CORE_VAD_BACKEND))'; \
	(cd $(AUDIO_CORE_DIR) && uv run speakerfocus-audio-core --addr "$(AUDIO_CORE_BIND)" --vad-backend "$(AUDIO_CORE_VAD_BACKEND)") & \
	audio_core_pid=$$!; \
	trap 'kill $$audio_core_pid >/dev/null 2>&1 || true; wait $$audio_core_pid >/dev/null 2>&1 || true' INT TERM EXIT; \
	sleep 2; \
	$(MAKE) run REALTIME=true INPUT_LOOPS=0 DISCARD_OUTPUT=true AUDIO_CORE_ADDR='$(AUDIO_CORE_BIND)'

proto:
	PATH="$$HOME/go/bin:$$PATH" $(PROTOC) -I $(PROTO_DIR) \
		--go_out=$(GO_SERVICE_DIR) \
		--go_opt=module=github.com/abend-arg/speakerfocus/services/orchestrator-go \
		--go-grpc_out=$(GO_SERVICE_DIR) \
		--go-grpc_opt=module=github.com/abend-arg/speakerfocus/services/orchestrator-go \
		$(PROTO_DIR)/audio_core/v1/audio_core.proto
	cd $(AUDIO_CORE_DIR) && uv run python -m grpc_tools.protoc \
		-I ../../$(PROTO_DIR) \
		--python_out=src \
		--grpc_python_out=src \
		../../$(PROTO_DIR)/audio_core/v1/audio_core.proto

audio-core-sync:
	cd $(AUDIO_CORE_DIR) && uv sync

audio-core:
	cd $(AUDIO_CORE_DIR) && uv run speakerfocus-audio-core --addr "$(AUDIO_CORE_BIND)" --vad-backend "$(AUDIO_CORE_VAD_BACKEND)"

obs-up:
	$(MAKE) -C $(OBS_DIR) up
	@printf '%s\n' 'Grafana:    http://localhost:3000'
	@printf '%s\n' 'Prometheus: http://localhost:9090'

obs-down:
	$(MAKE) -C $(OBS_DIR) down

obs-restart:
	$(MAKE) -C $(OBS_DIR) restart

obs-logs:
	$(MAKE) -C $(OBS_DIR) logs

obs-ps:
	$(MAKE) -C $(OBS_DIR) ps

obs-config:
	$(MAKE) -C $(OBS_DIR) config

tidy:
	$(MAKE) -C $(GO_SERVICE_DIR) tidy

clean:
	$(MAKE) -C $(GO_SERVICE_DIR) clean
