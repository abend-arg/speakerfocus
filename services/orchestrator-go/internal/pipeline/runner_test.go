package pipeline_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/audio"
	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/audioio/wav"
	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/pipeline"
)

func TestRunnerCopiesWavInputToOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.wav")
	outputPath := filepath.Join(dir, "output.wav")

	format := audio.Format{
		SampleRate:     8000,
		Channels:       1,
		SampleFormat:   audio.SampleFormatS16LE,
		BytesPerSample: 2,
	}
	pcm := make([]byte, 8000)
	for i := range pcm {
		pcm[i] = byte(i % 251)
	}

	writeTestWav(t, inputPath, format, pcm)

	runner := pipeline.Runner{
		Source: &wav.WavFileSource{
			Path:          inputPath,
			ChunkDuration: 20 * time.Millisecond,
		},
		Sink: &wav.WavFileSink{
			Path: outputPath,
		},
	}

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run pipeline: %v", err)
	}

	gotFormat, gotPCM := readTestWav(t, outputPath)
	if gotFormat != format {
		t.Fatalf("format mismatch: got %#v want %#v", gotFormat, format)
	}
	if !bytes.Equal(gotPCM, pcm) {
		t.Fatalf("PCM mismatch: got %d bytes want %d bytes", len(gotPCM), len(pcm))
	}
}

func TestRunnerCopiesLoopedWavInputToOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.wav")
	outputPath := filepath.Join(dir, "output.wav")

	format := audio.Format{
		SampleRate:     8000,
		Channels:       1,
		SampleFormat:   audio.SampleFormatS16LE,
		BytesPerSample: 2,
	}
	pcm := make([]byte, 1600)
	for i := range pcm {
		pcm[i] = byte(i % 251)
	}

	writeTestWav(t, inputPath, format, pcm)

	runner := pipeline.Runner{
		Source: &wav.LoopingFileSource{
			Path:          inputPath,
			ChunkDuration: 20 * time.Millisecond,
			Loops:         3,
		},
		Sink: &wav.WavFileSink{
			Path: outputPath,
		},
	}

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run pipeline: %v", err)
	}

	wantPCM := append(append(append([]byte(nil), pcm...), pcm...), pcm...)
	gotFormat, gotPCM := readTestWav(t, outputPath)
	if gotFormat != format {
		t.Fatalf("format mismatch: got %#v want %#v", gotFormat, format)
	}
	if !bytes.Equal(gotPCM, wantPCM) {
		t.Fatalf("PCM mismatch: got %d bytes want %d bytes", len(gotPCM), len(wantPCM))
	}
}

func writeTestWav(t *testing.T, path string, format audio.Format, pcm []byte) {
	t.Helper()

	sink := &wav.WavFileSink{Path: path}
	if err := sink.Open(context.Background(), format); err != nil {
		t.Fatalf("open test sink: %v", err)
	}
	if err := sink.WriteChunk(context.Background(), audio.Chunk{Data: pcm}); err != nil {
		t.Fatalf("write test sink: %v", err)
	}
	if err := sink.Close(); err != nil {
		t.Fatalf("close test sink: %v", err)
	}
}

func readTestWav(t *testing.T, path string) (audio.Format, []byte) {
	t.Helper()

	source := &wav.WavFileSource{
		Path:          path,
		ChunkDuration: 20 * time.Millisecond,
	}
	format, err := source.Open(context.Background())
	if err != nil {
		t.Fatalf("open test source: %v", err)
	}
	defer source.Close()

	var pcm []byte
	for {
		chunk, err := source.ReadChunk(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read test source: %v", err)
		}
		pcm = append(pcm, chunk.Data...)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat output: %v", err)
	}

	return format, pcm
}
