# VAD Audio Core

SpeakerFocus uses a Python audio core service for ML-oriented audio stages. The
Go orchestrator talks to it over gRPC.

## Current Behavior

The first audio core capability is VAD.

For each input chunk:

- The Go orchestrator sends mono PCM S16LE audio to the Python service.
- The Python service runs VAD.
- The Python service returns only the VAD decision.
- If speech is detected, the Go orchestrator keeps the original PCM chunk.
- If speech is not detected, the Go orchestrator replaces the chunk with PCM
  zeros of the same length.

The orchestrator owns the pipeline decision. This means downstream sinks still
receive one output chunk for every input chunk. Silence is represented as
explicit PCM zeros, not as a missing chunk.

## VAD Backend

The default backend is Silero VAD:

```sh
make audio-core
```

This expands to:

```sh
uv run speakerfocus-audio-core --addr 127.0.0.1:50051 --vad-backend silero
```

There is also an `energy` backend for local wiring tests:

```sh
make audio-core AUDIO_CORE_VAD_BACKEND=energy
```

The energy backend is not intended as the production VAD. It exists so gRPC and
the orchestrator integration can be tested without loading Silero.

There is also a WebRTC VAD backend:

```sh
make audio-core AUDIO_CORE_VAD_BACKEND=webrtc
```

WebRTC VAD is much lighter than Silero and useful as a realistic low-latency
baseline. It accepts 10, 20, or 30 ms frames at 8, 16, 32, or 48 kHz. The
current service pads short final WAV chunks to the next supported frame size for
VAD inference only.

## Audio Contract

The current audio core requires:

```text
sample rate: 16000 Hz
channels: 1
format: PCM S16LE
chunk duration: 20 ms preferred
```

Silero requires a minimum inference window internally. The service pads short 20
ms chunks to the minimum model window for VAD inference, but the response still
maps to the original input sequence number.

## gRPC Contract

The proto is defined in:

```text
proto/audio_core/v1/audio_core.proto
```

The main RPC is bidirectional streaming:

```text
AudioCore.DetectVoice(stream AudioChunk) returns (stream VadResponse)
```

Each response includes:

```text
sequence_number
vad.speech_probability
vad.is_speech
vad.state
```

The stream remains open while audio is flowing. The orchestrator sends one chunk
and receives one VAD decision with the same sequence number.

The Go-observed VAD latency, including local gRPC roundtrip, is recorded as:

```text
speakerfocus_stage_duration_seconds{stage="vad_detect"}
```

## Orchestrator Policy

VAD business rules live in the Go orchestrator under:

```text
services/orchestrator-go/internal/policy
```

The current policy maps a `VoiceDecision` to an audio action:

```text
SPEECH or MAYBE_SILENCE -> pass through original PCM
SILENCE                 -> replace the chunk with PCM zeros
```

This keeps orchestration decisions out of the Python audio core. The Python
service reports VAD information; the Go policy decides what the pipeline does
with that information.

The policy decision is exported as:

```text
speakerfocus_vad_decisions_total{action="pass_through|silence",state="..."}
```

Grafana uses this to show voice chunks and silenced chunks in the VAD stage
section.

## Running Locally

Start the full dev stack:

```sh
make dev
```

This starts observability, the audio core, and the Go orchestrator. The default
VAD backend is Silero.

Use WebRTC for a lower-overhead dev baseline:

```sh
make dev AUDIO_CORE_VAD_BACKEND=webrtc
```

Start only the audio core:

```sh
make audio-core
```

In another terminal, run the Go orchestrator with the audio core enabled:

```sh
make run AUDIO_CORE_ADDR=127.0.0.1:50051
```

Then run the Go orchestrator manually:

```sh
make run AUDIO_CORE_ADDR=127.0.0.1:50051
```

## Regenerating Stubs

After editing the proto:

```sh
make proto
```

This regenerates both:

- Go stubs under `services/orchestrator-go/internal/audiocorepb`
- Python stubs under `services/audio-core-python/src/audio_core/v1`

The Go generators must be installed:

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Python generation uses the `uv` environment for `services/audio-core-python`.

## Current Decision

The current implementation keeps timing simple:

- Do not drop chunks during silence.
- Do not close/reopen streams during silence.
- Let the Python audio core return only VAD decisions.
- Let the Go orchestrator write explicit PCM zeros for non-speech chunks.
- Use VAD to skip expensive future stages.
- Keep the output timeline continuous.
