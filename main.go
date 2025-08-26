package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var scheduler = NewScheduler()

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
	defer close(sigCh)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	wg := sync.WaitGroup{}
	runnerChangedCh := make(chan struct{})
	defer close(runnerChangedCh)

	ctx, cancel := context.WithCancel(context.Background())

	// main schedule loop
	go func(ctx context.Context) {
		for {
			slog.Info("Scheduling runners")
			createRunners(context.TODO(), &wg, config, runnerChangedCh)

			select {
			case <-ctx.Done():
				return
			case <-runnerChangedCh:
			}
		}
	}(ctx)

	<-sigCh
	slog.Info("Shutting down...")

	cancel()

	for runner := range scheduler.Runners() {
		runner.SendTerminate()

		go func() {
			<-runner.errCh
		}()
	}

	wg.Wait()
	slog.Info("All runners have exited. Shutdown complete.")
}

func createRunners(ctx context.Context, wg *sync.WaitGroup, config *Config, runnerChangedCh chan<- struct{}) {
	if len(config.RunnerGroups) == 0 {
		return
	}

	for _, runnerGroup := range config.RunnerGroups {
		createRunnersForGroup(ctx, wg, &runnerGroup, runnerChangedCh)
	}
}

func createRunnersForGroup(ctx context.Context, wg *sync.WaitGroup, runnerGroupConfig *RunnerGroupConfig, runnerChangedCh chan<- struct{}) {
	// 1. Check how many runners are already running for this group
	existingCount := 0
	// XXX Simple but inefficient way
	for runner := range scheduler.Runners() {
		if runner.RunnerGroupConfig != nil && runner.RunnerGroupConfig.Name == runnerGroupConfig.Name {
			existingCount++
		}
	}

	// 2. Determine how many more runners to create
	createCount := runnerGroupConfig.Count - existingCount

	// 3. Create the required number of runners in goroutines
	for i := 0; i < createCount; i++ {
		launchRunner(ctx, runnerGroupConfig, wg, runnerChangedCh)
	}
}

// Launch a single runner and is responsible for its lifecycle
// When runner exits with error, notify via channel and remove it from the global map
func launchRunner(ctx context.Context, runnerGroupConfig *RunnerGroupConfig, wg *sync.WaitGroup, runnerChangedCh chan<- struct{}) {
	runner := scheduler.NewRunner(runnerGroupConfig)

	go runner.Run(ctx, wg)

	go func() {
		select {
		case exitInfo := <-runner.errCh:
			if exitInfo.Err != nil {
				runner.logger().Error("Runner exited with error", "error", exitInfo.Err)
			} else {
				runner.logger().Info("Runner exited normally")
			}

			runnerChangedCh <- struct{}{}
		case <-ctx.Done():
		}

		scheduler.RemoveRunner(runner.Id)
	}()
}
