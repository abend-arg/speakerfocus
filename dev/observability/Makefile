SHELL := /bin/sh

COMPOSE ?= docker compose
COMPOSE_FILE ?= compose.yaml
PROJECT ?= speakerfocus
PROMETHEUS_IMAGE ?= prom/prometheus:latest
GRAFANA_IMAGE ?= grafana/grafana:latest

.PHONY: help up down restart logs ps config

help:
	@printf '%s\n' 'Observability targets:'
	@printf '%s\n' '  make up       Start Prometheus and Grafana'
	@printf '%s\n' '  make down     Stop Prometheus and Grafana'
	@printf '%s\n' '  make restart  Restart the stack'
	@printf '%s\n' '  make logs     Follow logs'
	@printf '%s\n' '  make ps       Show containers'
	@printf '%s\n' '  make config   Render Compose config'
	@printf '%s\n' ''
	@printf '%s\n' 'Variables:'
	@printf '%s\n' '  COMPOSE="docker compose"'
	@printf '%s\n' '  PROJECT=speakerfocus'
	@printf '%s\n' '  PROMETHEUS_IMAGE=prom/prometheus:latest GRAFANA_IMAGE=grafana/grafana:latest'

up:
	$(COMPOSE) -p $(PROJECT) -f $(COMPOSE_FILE) up -d

down:
	$(COMPOSE) -p $(PROJECT) -f $(COMPOSE_FILE) down

restart: down up

logs:
	$(COMPOSE) -p $(PROJECT) -f $(COMPOSE_FILE) logs -f

ps:
	$(COMPOSE) -p $(PROJECT) -f $(COMPOSE_FILE) ps

config:
	$(COMPOSE) -p $(PROJECT) -f $(COMPOSE_FILE) config
