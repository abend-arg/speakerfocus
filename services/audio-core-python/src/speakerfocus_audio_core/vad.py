from __future__ import annotations

from dataclasses import dataclass
from enum import Enum

import numpy as np


class VadState(Enum):
    SILENCE = "silence"
    SPEECH = "speech"
    MAYBE_SILENCE = "maybe_silence"


@dataclass(frozen=True)
class VadResult:
    speech_probability: float
    is_speech: bool


class VadBackend:
    name = "unknown"

    def predict(self, pcm_s16le: bytes, sample_rate_hz: int) -> VadResult:
        raise NotImplementedError


class EnergyVadBackend(VadBackend):
    name = "energy"

    def __init__(self, threshold: float = 0.015) -> None:
        self._threshold = threshold

    def predict(self, pcm_s16le: bytes, sample_rate_hz: int) -> VadResult:
        del sample_rate_hz
        samples = pcm_s16le_to_float32(pcm_s16le)
        if samples.size == 0:
            return VadResult(speech_probability=0.0, is_speech=False)

        rms = float(np.sqrt(np.mean(samples * samples)))
        probability = min(1.0, rms / self._threshold)
        return VadResult(
            speech_probability=probability,
            is_speech=probability >= 0.5,
        )


class SileroVadBackend(VadBackend):
    name = "silero"

    def __init__(self, threshold: float = 0.5) -> None:
        try:
            from silero_vad import load_silero_vad
            import torch
        except ImportError as exc:
            raise RuntimeError(
                "Silero VAD is not installed. Run `uv sync` in services/audio-core-python "
                "or start with `--vad-backend energy` for local wiring tests."
            ) from exc

        self._threshold = threshold
        self._model = load_silero_vad(onnx=True)
        self._torch = torch

    def predict(self, pcm_s16le: bytes, sample_rate_hz: int) -> VadResult:
        if sample_rate_hz != 16000:
            raise ValueError(f"silero backend requires 16000 Hz audio, got {sample_rate_hz}")

        samples = pcm_s16le_to_float32(pcm_s16le)
        if samples.size < 512:
            samples = np.pad(samples, (0, 512 - samples.size))
        tensor = self._torch.from_numpy(samples)
        probability = float(self._model(tensor, sample_rate_hz).item())
        return VadResult(
            speech_probability=probability,
            is_speech=probability >= self._threshold,
        )


class WebRtcVadBackend(VadBackend):
    name = "webrtc"

    def __init__(self, aggressiveness: int = 2) -> None:
        try:
            import webrtcvad
        except ImportError as exc:
            raise RuntimeError(
                "WebRTC VAD is not installed. Run `uv sync` in services/audio-core-python."
            ) from exc

        self._vad = webrtcvad.Vad(aggressiveness)

    def predict(self, pcm_s16le: bytes, sample_rate_hz: int) -> VadResult:
        if sample_rate_hz not in {8000, 16000, 32000, 48000}:
            raise ValueError(
                "webrtc backend requires 8000, 16000, 32000, or 48000 Hz audio, "
                f"got {sample_rate_hz}"
            )

        frame = pad_webrtc_frame(pcm_s16le, sample_rate_hz)
        duration_ms = frame_duration_ms(frame, sample_rate_hz)
        if duration_ms not in {10, 20, 30}:
            raise ValueError(
                f"webrtc backend requires 10, 20, or 30 ms frames, got {duration_ms} ms"
            )

        is_speech = self._vad.is_speech(frame, sample_rate_hz)
        return VadResult(
            speech_probability=1.0 if is_speech else 0.0,
            is_speech=is_speech,
        )


@dataclass
class VadSmoother:
    negative_hangover_frames: int = 10

    _state: VadState = VadState.SILENCE
    _negative_frames: int = 0

    def update(self, raw_is_speech: bool) -> VadState:
        if raw_is_speech:
            self._state = VadState.SPEECH
            self._negative_frames = 0
            return self._state

        if self._state == VadState.SPEECH:
            self._negative_frames = 1
            self._state = VadState.MAYBE_SILENCE
            return self._state

        if self._state == VadState.MAYBE_SILENCE:
            self._negative_frames += 1
            if self._negative_frames >= self.negative_hangover_frames:
                self._state = VadState.SILENCE
            return self._state

        self._negative_frames = 0
        self._state = VadState.SILENCE
        return self._state

    def should_pass_audio(self, raw_is_speech: bool) -> bool:
        return raw_is_speech or self._state in {VadState.SPEECH, VadState.MAYBE_SILENCE}


def create_vad_backend(name: str) -> VadBackend:
    normalized = name.strip().lower()
    if normalized == "silero":
        return SileroVadBackend()
    if normalized == "webrtc":
        return WebRtcVadBackend()
    if normalized == "energy":
        return EnergyVadBackend()
    raise ValueError(f"unsupported VAD backend {name!r}; expected silero, webrtc, or energy")


def pcm_s16le_to_float32(pcm_s16le: bytes) -> np.ndarray:
    samples = np.frombuffer(pcm_s16le, dtype="<i2")
    return samples.astype(np.float32) / 32768.0


def frame_duration_ms(pcm_s16le: bytes, sample_rate_hz: int) -> int:
    samples = len(pcm_s16le) // 2
    return int((samples * 1000) / sample_rate_hz)


def pad_webrtc_frame(pcm_s16le: bytes, sample_rate_hz: int) -> bytes:
    duration_ms = frame_duration_ms(pcm_s16le, sample_rate_hz)
    if duration_ms in {10, 20, 30}:
        return pcm_s16le
    if duration_ms <= 10:
        target_ms = 10
    elif duration_ms <= 20:
        target_ms = 20
    elif duration_ms <= 30:
        target_ms = 30
    else:
        raise ValueError(
            f"webrtc backend requires frames up to 30 ms, got {duration_ms} ms"
        )

    target_bytes = int(sample_rate_hz * target_ms / 1000) * 2
    return pcm_s16le + bytes(target_bytes - len(pcm_s16le))
