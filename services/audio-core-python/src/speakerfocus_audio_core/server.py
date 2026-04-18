from __future__ import annotations

import argparse
import logging
from concurrent import futures

import grpc

from audio_core.v1 import audio_core_pb2, audio_core_pb2_grpc
from speakerfocus_audio_core.vad import VadState, VadSmoother, create_vad_backend


LOGGER = logging.getLogger("speakerfocus_audio_core")


class AudioCoreService(audio_core_pb2_grpc.AudioCoreServicer):
    def __init__(self, vad_backend_name: str) -> None:
        self._vad = create_vad_backend(vad_backend_name)

    def Health(self, request, context):
        del request, context
        return audio_core_pb2.HealthResponse(status="ok", vad_backend=self._vad.name)

    def DetectVoice(self, request_iterator, context):
        smoother = VadSmoother()

        for chunk in request_iterator:
            try:
                self._validate_chunk(chunk)
                raw_decision = self._vad.predict(chunk.pcm, int(chunk.sample_rate_hz))
            except Exception as exc:
                context.abort(grpc.StatusCode.INVALID_ARGUMENT, str(exc))

            state = smoother.update(raw_decision.is_speech)

            yield audio_core_pb2.VadResponse(
                sequence_number=chunk.sequence_number,
                vad=audio_core_pb2.VadDecision(
                    speech_probability=raw_decision.speech_probability,
                    is_speech=raw_decision.is_speech,
                    state=_proto_state(state),
                ),
            )

    def _validate_chunk(self, chunk) -> None:
        if chunk.sample_format != audio_core_pb2.SAMPLE_FORMAT_PCM_S16LE:
            raise ValueError("audio core only accepts PCM S16LE chunks")
        if chunk.channels != 1:
            raise ValueError(f"audio core only accepts mono chunks, got {chunk.channels} channels")
        if chunk.sample_rate_hz != 16000:
            raise ValueError(f"audio core requires 16000 Hz audio, got {chunk.sample_rate_hz}")
        if len(chunk.pcm) == 0:
            raise ValueError("audio chunk payload is empty")
        if len(chunk.pcm) % 2 != 0:
            raise ValueError("PCM S16LE payload size must be aligned to 2 bytes")


def _proto_state(state: VadState) -> int:
    if state == VadState.SPEECH:
        return audio_core_pb2.VAD_STATE_SPEECH
    if state == VadState.MAYBE_SILENCE:
        return audio_core_pb2.VAD_STATE_MAYBE_SILENCE
    return audio_core_pb2.VAD_STATE_SILENCE


def serve(addr: str, vad_backend: str) -> None:
    service = AudioCoreService(vad_backend)
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    audio_core_pb2_grpc.add_AudioCoreServicer_to_server(service, server)
    server.add_insecure_port(addr)
    server.start()
    LOGGER.info("audio core listening on %s with %s VAD", addr, service._vad.name)
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        LOGGER.info("stopping audio core")
        server.stop(grace=1).wait()


def main() -> None:
    parser = argparse.ArgumentParser(description="SpeakerFocus Python audio core")
    parser.add_argument("--addr", default="127.0.0.1:50051", help="gRPC listen address")
    parser.add_argument(
        "--vad-backend",
        default="silero",
        choices=["silero", "webrtc", "energy"],
        help="VAD backend to use",
    )
    args = parser.parse_args()

    logging.basicConfig(level=logging.INFO, format="%(levelname)s %(name)s: %(message)s")
    serve(args.addr, args.vad_backend)


if __name__ == "__main__":
    main()
