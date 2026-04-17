# Realtime Latency Decisions

This document records the current latency targets for SpeakerFocus while the
prototype is still chunk-based and processes one chunk at a time.

## Goals

SpeakerFocus targets conversational audio. The current working goals are:

- Keep per-chunk processing below the chunk duration, so the pipeline does not
  build backlog.
- Keep total end-to-end latency below 150 ms for a usable conversation.

These are separate constraints. A system can process each chunk faster than
realtime and still have too much conversational latency if buffering, output
latency, or model lookahead are too large.

## Chunk Budget

The current default chunk duration is 20 ms.

For 20 ms chunks:

- `p50` chunk processing should be below 5 ms.
- `p95` chunk processing should be below 10-12 ms.
- `p99` chunk processing should be below 15 ms.
- 20 ms is the hard limit for sustained per-chunk processing.

If sustained chunk processing exceeds 20 ms, the pipeline is slower than
realtime and will accumulate delay unless chunks are skipped or dropped.

The dashboard marks these thresholds on the chunk and stage latency panels:

- 10 ms: warning
- 15 ms: warning
- 20 ms: hard limit

## End-To-End Budget

The target end-to-end latency is:

```text
p99 end-to-end latency < 150 ms
```

The preferred operating target is:

```text
p95 end-to-end latency < 100 ms
```

End-to-end latency includes more than model inference:

```text
input buffering
+ chunk duration
+ feature extraction
+ ML inference
+ postprocessing
+ model lookahead
+ output buffering
+ scheduling and I/O overhead
```

Model lookahead is a key risk. A causal or near-causal model can keep this
small. A model that needs large future context can meet the per-chunk processing
budget while still violating the total latency budget.

## Jetson Feasibility

Running ML noise removal on a Jetson is considered feasible under these
conditions:

- The model is streaming-oriented.
- The model is causal or uses small lookahead.
- The model is small enough to keep inference `p99` below the chunk budget.
- Tensor and audio buffers are reused across chunks.
- CPU/GPU copies are minimized.
- The hot path avoids Python or IPC overhead where possible.

The current assumption is that a lightweight streaming denoising/enhancement
model can fit this budget on a modern Jetson, especially if optimized with
TensorRT or another low-overhead runtime.

Large non-streaming models, transformer-heavy models, or models that require
long future context are higher risk for the 150 ms total latency target.

## Metrics To Watch

The main dashboard queries are based on Prometheus histograms.

Total chunk latency:

```promql
histogram_quantile(
  0.99,
  sum by (le) (rate(speakerfocus_chunk_duration_seconds_bucket[5m]))
)
```

Per-stage latency:

```promql
histogram_quantile(
  0.99,
  sum by (le, stage) (rate(speakerfocus_stage_duration_seconds_bucket[5m]))
)
```

Throughput for 20 ms chunks should be close to 50 chunks per second:

```promql
rate(speakerfocus_chunks_total[1m])
```

If chunks are changed to 10 ms, expected throughput becomes 100 chunks per
second. The lower chunk size reduces buffering latency but increases per-chunk
overhead.

## Future Metrics

The following metrics should be added when the pipeline grows beyond the current
file passthrough prototype:

```text
speakerfocus_realtime_lag_seconds
speakerfocus_dropped_chunks_total
speakerfocus_skipped_stages_total
speakerfocus_stage_duration_seconds{stage="feature_extract"}
speakerfocus_stage_duration_seconds{stage="ml_inference"}
speakerfocus_stage_duration_seconds{stage="postprocess"}
```

`speakerfocus_realtime_lag_seconds` should capture whether the system is
falling behind wall-clock realtime. If this grows over time, the pipeline is not
keeping up even if individual stage metrics look acceptable.

## Current Decision

Start with 20 ms chunks.

The working acceptance criteria are:

- Per-chunk processing `p99 < 15 ms`.
- Sustained per-chunk processing never exceeds 20 ms.
- End-to-end conversational latency `p99 < 150 ms`.
- Prefer end-to-end conversational latency `p95 < 100 ms`.

Only consider 10 ms chunks after the 20 ms pipeline is stable and measured.
