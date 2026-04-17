# speakerfocus
Target speaker enhancement under real-time constraints

## Current prototype

The first iteration lives in `services/orchestrator-go` and implements a file-to-file audio passthrough:

```text
audioio/wav.WavFileSource -> PipelineRunner -> audioio/wav.WavFileSink
```

It reads 16-bit PCM WAV input in fixed-duration chunks and writes those chunks to a 16-bit PCM WAV output without processing. This keeps the capture/sink boundary in Go, which is the intended owner of orchestration, timing, backpressure, metrics, and future gRPC streaming to the Python audio core.

Core audio contracts live in `internal/audio`; concrete file/device integrations live under `internal/audioio`. That keeps the pipeline independent from WAV now and leaves room for future OS-specific sources and sinks such as ALSA/PipeWire on Linux, CoreAudio on macOS, and WASAPI on Windows.

Run it with:

```sh
make run
```

The daemon exposes Prometheus metrics on `127.0.0.1:2112/metrics` by default. The root Makefile uses `0.0.0.0:2112` so Prometheus in Docker can scrape it:

```sh
make run METRICS_ADDR=0.0.0.0:2112
```

Disable metrics with:

```sh
make run METRICS_ADDR=
```

Common run variables:

```sh
make run INPUT=services/orchestrator-go/test.wav OUTPUT=/tmp/output.wav CHUNK_MS=20
```

For dashboards while using file input, run in simulated realtime and loop the WAV so Prometheus has enough samples to scrape:

```sh
make run REALTIME=true INPUT_LOOPS=0 DISCARD_OUTPUT=true
```

`INPUT_LOOPS=0` loops forever. `DISCARD_OUTPUT=true` avoids writing an unbounded WAV file. Use `Ctrl-C` to stop it.

Build the daemon with:

```sh
make build
```

Build variables can be passed through `make`, for example:

```sh
make build GOOS=linux GOARCH=arm64 CGO_ENABLED=0
```

## Development observability

Prometheus and Grafana can be started locally with Docker Compose:

```sh
make obs-up
```

Then run `speakerfocusd` with `-metrics-addr 0.0.0.0:2112`. Prometheus scrapes `host.docker.internal:2112`, and Grafana is available at:

```text
http://localhost:3000
```

The provisioned dashboard is `SpeakerFocus / SpeakerFocus`. Prometheus is available at:

```text
http://localhost:9090
```

For a Jetson on the network, change `dev/observability/prometheus/prometheus.yml` from `host.docker.internal:2112` to `JETSON_IP:2112`, and run the daemon on the Jetson with `-metrics-addr 0.0.0.0:2112`.

Start observability and then run the daemon from one command:

```sh
make dev
```

`make dev` starts Prometheus/Grafana, then runs the daemon with `REALTIME=true INPUT_LOOPS=0 DISCARD_OUTPUT=true`.

Stop Prometheus and Grafana with:

```sh
make obs-down
```

Current metrics include:

```text
speakerfocus_chunk_duration_seconds{result="ok|error"}
speakerfocus_stage_duration_seconds{stage="source_read|sink_write|..."}
speakerfocus_chunks_total{result="ok|error"}
speakerfocus_stage_errors_total{stage="...",reason="..."}
```

Run tests with:

```sh
make test
```
