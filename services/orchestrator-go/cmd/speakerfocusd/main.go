package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/audio"
	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/audioio/discard"
	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/audioio/wav"
	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/observability"
	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/pipeline"
)

func main() {
	var inputPath string
	var outputPath string
	var chunkMS int
	var metricsAddr string
	var metricsPath string
	var realtime bool
	var inputLoops int
	var discardOutput bool

	flag.StringVar(&inputPath, "input", "", "input WAV file")
	flag.StringVar(&outputPath, "output", "", "output WAV file")
	flag.IntVar(&chunkMS, "chunk-ms", 20, "chunk duration in milliseconds")
	flag.StringVar(&metricsAddr, "metrics-addr", "127.0.0.1:2112", "metrics listen address; set empty to disable")
	flag.StringVar(&metricsPath, "metrics-path", "/metrics", "metrics HTTP path")
	flag.BoolVar(&realtime, "realtime", false, "pace WAV chunks at their audio duration")
	flag.IntVar(&inputLoops, "input-loops", 1, "number of times to read the input WAV; set 0 to loop forever")
	flag.BoolVar(&discardOutput, "discard-output", false, "discard output chunks instead of writing a WAV file")
	flag.Parse()

	if inputPath == "" || (!discardOutput && outputPath == "") {
		fmt.Fprintln(os.Stderr, "usage: speakerfocusd -input input.wav -output output.wav [-chunk-ms 20]")
		os.Exit(2)
	}
	if chunkMS <= 0 {
		fmt.Fprintln(os.Stderr, "chunk-ms must be greater than zero")
		os.Exit(2)
	}
	if inputLoops < 0 {
		fmt.Fprintln(os.Stderr, "input-loops must be greater than or equal to zero")
		os.Exit(2)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	registry := observability.NewRegistry()
	metrics := observability.NewMetrics(registry)
	metricsServer, err := observability.StartMetricsServer(ctx, metricsAddr, metricsPath, registry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start metrics server: %v\n", err)
		os.Exit(1)
	}
	if metricsServer != nil {
		fmt.Fprintf(os.Stderr, "metrics listening on %s%s\n", metricsServer.Addr(), metricsServer.Path())
	}

	var sink audio.Sink
	if discardOutput {
		sink = discard.Sink{}
	} else {
		sink = &wav.WavFileSink{
			Path: outputPath,
		}
	}

	runner := pipeline.Runner{
		Source: &wav.LoopingFileSource{
			Path:          inputPath,
			ChunkDuration: time.Duration(chunkMS) * time.Millisecond,
			Loops:         inputLoops,
		},
		Sink:     sink,
		Recorder: metrics,
		Realtime: realtime,
	}

	if err := runner.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "speakerfocusd: %v\n", err)
		os.Exit(1)
	}
}
