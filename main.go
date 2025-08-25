package main

import (
	"context"
	"flag"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"

	"github.com/google/uuid"
)

var runners = make(map[string]Runner, 0)

func main() {
	// Set up structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.Parse()

	if configPath == "" {
		slog.Error("Configuration file path is required", "usage", "Use -config flag to specify config file path")
		os.Exit(1)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("Configuration loaded", "runner_groups", len(config.RunnerGroups))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	wg := sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())

	go func(ctx context.Context) {
		for {
			createRunners(context.TODO(), &wg, config)

			exitInfoCh := orRunnerCh(slices.Collect(maps.Values(runners)))

			select {
			case exitInfo := <-exitInfoCh:
				if exitInfo.Err != nil {
					slog.Error("Runner error", "runner_id", exitInfo.RunnerId, "error", exitInfo.Err)
				} else {
					slog.Info("Runner exited normally", "runner_id", exitInfo.RunnerId)
				}

				delete(runners, exitInfo.RunnerId)

			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	<-sigCh
	slog.Info("Shutting down...")

	cancel()

	for _, runner := range runners {
		runner.SendTerminate()

		go func() {
			<-runner.errCh
		}()
	}

	wg.Wait()
	slog.Info("All runners have exited. Shutdown complete.")

}

func createRunners(ctx context.Context, wg *sync.WaitGroup, config *Config) {
	if len(config.RunnerGroups) == 0 {
		return
	}

	for _, runnerGroup := range config.RunnerGroups {
		createRunnersForGroup(ctx, wg, &runnerGroup)
	}
}

func createRunnersForGroup(ctx context.Context, wg *sync.WaitGroup, runnerGroupConfig *RunnerGroupConfig) {
	// 1. Check how many runners are already running for this group
	existingCount := 0
	// XXX Simple but inefficient way
	for _, runner := range runners {
		if runner.RunnerGroupConfig != nil && runner.RunnerGroupConfig.Name == runnerGroupConfig.Name {
			existingCount++
		}
	}

	// 2. Determine how many more runners to create
	createCount := runnerGroupConfig.Count - existingCount

	// 3. Create the required number of runners in goroutines
	for i := 0; i < createCount; i++ {
		var id string

	GEN_ID:
		for {
			id = uuid.NewString()

			_, exist := runners[id]
			if !exist {
				break GEN_ID
			}
		}

		runner := NewRunner(id, runnerGroupConfig)
		runners[id] = runner

		go runner.Run(ctx, wg)
	}
}
