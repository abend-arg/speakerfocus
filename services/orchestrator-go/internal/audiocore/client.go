package audiocore

import (
	"context"
	"fmt"
	"time"

	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/audio"
	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/audiocorepb"
	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/policy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	Addr string

	conn   *grpc.ClientConn
	stream audiocorepb.AudioCore_DetectVoiceClient
	format audio.Format
}

func (c *Client) Open(ctx context.Context, format audio.Format) error {
	if c.Addr == "" {
		return fmt.Errorf("audio core address is required")
	}
	if err := validateFormat(format); err != nil {
		return err
	}

	conn, err := grpc.NewClient(c.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("create audio core client: %w", err)
	}
	c.conn = conn
	c.format = format

	client := audiocorepb.NewAudioCoreClient(conn)
	healthCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	health, err := client.Health(healthCtx, &audiocorepb.HealthRequest{})
	if err != nil {
		_ = c.Close()
		return fmt.Errorf("audio core health check: %w", err)
	}
	if health.GetStatus() != "ok" {
		_ = c.Close()
		return fmt.Errorf("audio core health status %q", health.GetStatus())
	}

	stream, err := client.DetectVoice(ctx)
	if err != nil {
		_ = c.Close()
		return fmt.Errorf("open VAD stream: %w", err)
	}
	c.stream = stream

	return nil
}

func (c *Client) DetectVoice(ctx context.Context, chunk audio.Chunk) (policy.VoiceDecision, error) {
	if err := ctx.Err(); err != nil {
		return policy.VoiceDecision{}, err
	}
	if c.stream == nil {
		return policy.VoiceDecision{}, fmt.Errorf("VAD stream is not open")
	}

	req := &audiocorepb.AudioChunk{
		SequenceNumber: chunk.FrameIndex,
		SampleRateHz:   c.format.SampleRate,
		Channels:       uint32(c.format.Channels),
		SampleFormat:   audiocorepb.SampleFormat_SAMPLE_FORMAT_PCM_S16LE,
		DurationMs:     uint32(chunk.Duration / time.Millisecond),
		Pcm:            chunk.Data,
	}
	if err := c.stream.Send(req); err != nil {
		return policy.VoiceDecision{}, fmt.Errorf("send VAD chunk: %w", err)
	}

	resp, err := c.stream.Recv()
	if err != nil {
		return policy.VoiceDecision{}, fmt.Errorf("receive VAD decision: %w", err)
	}
	if resp.GetSequenceNumber() != chunk.FrameIndex {
		return policy.VoiceDecision{}, fmt.Errorf("VAD sequence mismatch: got %d want %d", resp.GetSequenceNumber(), chunk.FrameIndex)
	}

	vad := resp.GetVad()
	return policy.VoiceDecision{
		SpeechProbability: vad.GetSpeechProbability(),
		IsSpeech:          vad.GetIsSpeech(),
		State:             voiceStateFromProto(vad.GetState()),
	}, nil
}

func (c *Client) Close() error {
	var err error
	if c.stream != nil {
		err = c.stream.CloseSend()
		c.stream = nil
	}
	if c.conn != nil {
		if closeErr := c.conn.Close(); err == nil {
			err = closeErr
		}
		c.conn = nil
	}
	return err
}

func voiceStateFromProto(state audiocorepb.VadState) policy.VoiceState {
	switch state {
	case audiocorepb.VadState_VAD_STATE_SPEECH:
		return policy.VoiceStateSpeech
	case audiocorepb.VadState_VAD_STATE_MAYBE_SILENCE:
		return policy.VoiceStateMaybeSilence
	default:
		return policy.VoiceStateSilence
	}
}

func validateFormat(format audio.Format) error {
	if err := format.Validate(); err != nil {
		return err
	}
	if format.SampleFormat != audio.SampleFormatS16LE {
		return fmt.Errorf("audio core requires %s format, got %s", audio.SampleFormatS16LE, format.SampleFormat)
	}
	if format.Channels != 1 {
		return fmt.Errorf("audio core requires mono audio, got %d channels", format.Channels)
	}
	if format.SampleRate != 16000 {
		return fmt.Errorf("audio core requires 16000 Hz audio, got %d", format.SampleRate)
	}
	return nil
}
