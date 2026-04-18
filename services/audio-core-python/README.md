# SpeakerFocus Audio Core

Python gRPC service for audio processing stages that are expected to grow into
ML-backed functionality.

The first stage is VAD-based gating:

- Input: mono PCM S16LE audio chunks.
- Default backend: Silero VAD.
- Output: VAD decision with probability, boolean speech flag, and smoothed state.
- The Go orchestrator decides whether to keep the chunk or replace it with PCM
  zeros.

Run with uv:

```sh
uv run speakerfocus-audio-core --addr 127.0.0.1:50051
```

For local wiring tests without downloading/loading Silero:

```sh
uv run speakerfocus-audio-core --vad-backend energy
```

For a lightweight production-style baseline:

```sh
uv run speakerfocus-audio-core --vad-backend webrtc
```
