package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/google/go-github/v74/github"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type Runner struct {
	Id          string
	owner       string
	repo        string
	githubToken string
	errCh       chan RunnerExitInfo
	stopCh      chan struct{}
}

func NewRunner(id string, owner, repo, githubToken string) Runner {
	return Runner{
		Id:          id,
		owner:       owner,
		repo:        repo,
		githubToken: githubToken,
		errCh:       make(chan RunnerExitInfo, 0),
		stopCh:      make(chan struct{}, 0),
	}
}

type RunnerExitInfo struct {
	RunnerId string
	Err      error
}

func (r *Runner) constructExitInfo(err error) RunnerExitInfo {
	return RunnerExitInfo{
		RunnerId: r.Id,
		Err:      err,
	}
}

func (r *Runner) logger() *slog.Logger {
	return slog.With("owner", r.owner, "repository", r.repo, "runner_name", r.runnerName())
}

func (r *Runner) runnerName() string {
	return fmt.Sprintf("runner-%s", r.Id)
}

func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer close(r.errCh)
	wg.Add(1)
	defer wg.Done()
	r.logger().Info("Starting runner")

	ghClient := github.NewClient(nil).WithAuthToken(r.githubToken)

	token, _, err := ghClient.Actions.CreateRegistrationToken(ctx, r.owner, r.repo)
	if err != nil {
		r.errCh <- r.constructExitInfo(fmt.Errorf("Error creating registration token err: %w", err))
		return
	}

	dockerClient, err := client.NewClientWithOpts(client.WithHost("unix:///var/run/docker.sock"), client.WithVersion("1.41"))
	if err != nil {
		r.errCh <- r.constructExitInfo(fmt.Errorf("Error creating Docker client err: %w", err))
		return
	}

	containerConfig := &container.Config{
		Image:      "ghcr.io/actions/actions-runner",
		Entrypoint: []string{"sh", "-c", fmt.Sprintf("./config.sh --url https://github.com/%s/%s --name %s --token %s --unattended --ephemeral; ./run.sh", r.owner, r.repo, r.runnerName(), *token.Token)},
	}
	hostConfig := &container.HostConfig{
		AutoRemove: true,
	}
	networkingConfig := &network.NetworkingConfig{}

	response, err := dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, &v1.Platform{}, r.runnerName())
	if err != nil {
		r.errCh <- r.constructExitInfo(fmt.Errorf("Error creating container err: %w", err))
		return
	}

	containerId := response.ID

	if err := dockerClient.ContainerStart(ctx, containerId, container.StartOptions{}); err != nil {
		r.errCh <- r.constructExitInfo(fmt.Errorf("Error starting container err: %w", err))
		return
	}

	waitCh, containerErrCh := dockerClient.ContainerWait(ctx, containerId, container.WaitConditionNotRunning)

	select {
	case err := <-containerErrCh:
		r.errCh <- r.constructExitInfo(fmt.Errorf("Error waiting for container err: %w", err))

	case waitResponse := <-waitCh:
		if waitResponse.Error != nil {
			r.errCh <- r.constructExitInfo(fmt.Errorf("Container exited with error: %s", waitResponse.Error.Message))
		} else {
			r.errCh <- r.constructExitInfo(nil)
		}

	case <-r.stopCh:
		err := r.stop(ctx, containerId)
		if err != nil {
			r.logger().Error("Error stopping runner", "error", err)
		}
	}
}

func (r *Runner) stop(ctx context.Context, containerId string) error {
	r.logger().Info("Stopping runner")

	ghClient := github.NewClient(nil).WithAuthToken(r.githubToken)

	dockerClient, err := client.NewClientWithOpts(client.WithHost("unix:///var/run/docker.sock"), client.WithVersion("1.41"))
	if err != nil {
		return fmt.Errorf("Error creating Docker client err: %w", err)
	}

	if err := dockerClient.ContainerRemove(ctx, containerId, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("Error removing container err: %w", err)
	}

	runnerName := r.runnerName()

	result, _, err := ghClient.Actions.ListRunners(ctx, r.owner, r.repo, &github.ListRunnersOptions{
		Name: &runnerName,
	})
	if err != nil {
		return fmt.Errorf("Error listing runners err: %w", err)
	}

	switch len(result.Runners) {
	case 0:
		r.logger().Info("Runner not found on GitHub, assuming already removed")
	case 1:
	default:
		r.logger().Warn("Multiple runners found with the same name, unexpected state")
	}

	for _, runner := range result.Runners {
		if runner.ID != nil {
			_, err := ghClient.Actions.RemoveRunner(ctx, r.owner, r.repo, runner.GetID())
			if err != nil {
				return err
			}
		}
	}

	r.logger().Info("Runner stopped successfully")
	return nil
}

func (r *Runner) SendTerminate() {
	r.logger().Info("Sending terminate signal")
	go func() {
		r.stopCh <- struct{}{}
	}()
}
