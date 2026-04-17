package pipeline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/audio"
)

type Recorder interface {
	ObserveChunkDuration(result string, duration time.Duration)
	ObserveStageDuration(stage string, duration time.Duration)
	IncChunk(result string)
	IncStageError(stage string, reason string)
}

type Runner struct {
	Source   audio.Source
	Sink     audio.Sink
	Recorder Recorder
	Realtime bool
}

func (r Runner) Run(ctx context.Context) (err error) {
	if r.Source == nil {
		return fmt.Errorf("source is required")
	}
	if r.Sink == nil {
		return fmt.Errorf("sink is required")
	}

	format, err := r.Source.Open(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, r.Source.Close())
	}()

	if err := r.Sink.Open(ctx, format); err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, r.Sink.Close())
	}()

	for {
		chunkStart := time.Now()

		chunk, err := runStage(ctx, r.Recorder, "source_read", func() (audio.Chunk, error) {
			return r.Source.ReadChunk(ctx)
		})
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			recordChunk(r.Recorder, "error", time.Since(chunkStart))
			return err
		}

		if err := runStageNoValue(ctx, r.Recorder, "sink_write", func() error {
			return r.Sink.WriteChunk(ctx, chunk)
		}); err != nil {
			recordChunk(r.Recorder, "error", time.Since(chunkStart))
			return err
		}

		recordChunk(r.Recorder, "ok", time.Since(chunkStart))

		if r.Realtime {
			if err := sleepUntil(ctx, chunkStart.Add(chunk.Duration)); err != nil {
				return err
			}
		}
	}
}

func runStage[T any](ctx context.Context, recorder Recorder, stage string, fn func() (T, error)) (T, error) {
	start := time.Now()
	value, err := fn()
	duration := time.Since(start)

	if recorder != nil && !errors.Is(err, io.EOF) {
		recorder.ObserveStageDuration(stage, duration)
		if err != nil && ctx.Err() == nil {
			recorder.IncStageError(stage, errorReason(err))
		}
	}

	return value, err
}

func runStageNoValue(ctx context.Context, recorder Recorder, stage string, fn func() error) error {
	_, err := runStage(ctx, recorder, stage, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}

func recordChunk(recorder Recorder, result string, duration time.Duration) {
	if recorder == nil {
		return
	}
	recorder.ObserveChunkDuration(result, duration)
	recorder.IncChunk(result)
}

func errorReason(err error) string {
	if err == nil {
		return "none"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline_exceeded"
	}
	return "error"
}

func sleepUntil(ctx context.Context, deadline time.Time) error {
	duration := time.Until(deadline)
	if duration <= 0 {
		return nil
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
