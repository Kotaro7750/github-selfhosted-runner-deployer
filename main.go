package main

import (
	"context"
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

func getEnvOrExit(key string) string {
	value := os.Getenv(key)
	if value == "" {
		slog.Error("Required environment variable not set", "variable", key)
		os.Exit(1)
	}
	return value
}

func main() {
	// Set up structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Validate required environment variables
	githubOwner := getEnvOrExit("GITHUB_OWNER")
	githubRepo := getEnvOrExit("GITHUB_REPO")
	githubToken := getEnvOrExit("GITHUB_TOKEN")

	slog.Info("GitHub configuration loaded", "owner", githubOwner, "repo", githubRepo)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	wg := sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())

	go func(ctx context.Context) {
		for {
			createRunners(context.TODO(), &wg, githubOwner, githubRepo, githubToken)

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

func createRunners(ctx context.Context, wg *sync.WaitGroup, githubOwner, githubRepo, githubToken string) {
	createCount := 2 - len(runners)

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

		runner := NewRunner(id, githubOwner, githubRepo, githubToken)
		runners[id] = runner

		go runner.Run(ctx, wg)
	}
}
