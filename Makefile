SHELL := /bin/sh

GO_SERVICE_DIR := services/orchestrator-go
OBS_DIR := dev/observability

MAKE ?= make

INPUT ?= $(GO_SERVICE_DIR)/test.wav
OUTPUT ?= $(GO_SERVICE_DIR)/output.wav
CHUNK_MS ?= 20
METRICS_ADDR ?= 0.0.0.0:2112
METRICS_PATH ?= /metrics
REALTIME ?= false
INPUT_LOOPS ?= 1
DISCARD_OUTPUT ?= false
ARGS ?=

.PHONY: help test build run run-bin tidy clean dev obs-up obs-down obs-restart obs-logs obs-ps obs-config

help:
	@printf '%s\n' 'Root targets:'
	@printf '%s\n' '  make test          Run all tests'
	@printf '%s\n' '  make build         Build Go daemon'
	@printf '%s\n' '  make run           Run Go daemon'
	@printf '%s\n' '  make run-bin       Build and run Go daemon'
	@printf '%s\n' '  make dev           Start observability and run daemon'
	@printf '%s\n' '  make obs-up        Start Prometheus and Grafana'
	@printf '%s\n' '  make obs-down      Stop Prometheus and Grafana'
	@printf '%s\n' '  make obs-logs      Follow Prometheus/Grafana logs'
	@printf '%s\n' '  make tidy          Run go mod tidy'
	@printf '%s\n' '  make clean         Remove Go build output'
	@printf '%s\n' ''
	@printf '%s\n' 'Common variables:'
	@printf '%s\n' '  INPUT=services/orchestrator-go/test.wav OUTPUT=services/orchestrator-go/output.wav'
	@printf '%s\n' '  CHUNK_MS=20 METRICS_ADDR=0.0.0.0:2112 METRICS_PATH=/metrics ARGS="..."'
	@printf '%s\n' '  REALTIME=false INPUT_LOOPS=1 DISCARD_OUTPUT=false'
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
		ARGS='$(ARGS)'

dev: obs-up
	$(MAKE) run REALTIME=true INPUT_LOOPS=0 DISCARD_OUTPUT=true

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
