# GitHub Self-hosted Runner Deployer

An application that automatically deploys and manages self-hosted runners for GitHub Actions. It runs as a Docker container and dynamically creates and removes runners for specified GitHub repositories.

## Features

- **Dynamic scaling**: Automatically recovers when runners exit
- **Ephemeral runners**: Runs runners in ephemeral mode, automatically removing them after job completion

## Configuration

The application uses a YAML configuration file to define GitHub settings and runner groups.
Some settings can also be overridden using environment variables.

### Configuration File Format

Create a YAML configuration file.
Sample configuration is provided in [`sample_config.yaml`](./sample_config.yaml).

### Configuration Fields

| Field | Description | Required | Environment Variable | Example |
|-------|-------------|----------|---------------------|---------|
| `defaultGithubOwner` | Default GitHub repository owner name | Yes* | `DEFAULT_GITHUB_OWNER` | `Kotaro7750` |
| `defaultGithubRepository` | Default GitHub repository name | Yes* | `DEFAULT_GITHUB_REPOSITORY` | `my-repo` |
| `defaultGithubToken` | Default GitHub Personal Access Token | Yes* | `DEFAULT_GITHUB_TOKEN` | `ghp_xxxxxxxxxxxx` |
| `defaultLabels` | Default labels for all runner groups | No | `DEFAULT_LABELS` | `["linux", "x64"]` or `linux,x64` for environment variable |
| `defaultNoDefaultLabels` | Disable default labels for all runner groups | No | `DEFAULT_NO_DEFAULT_LABELS` | `false` |
| `defaultImage` | Default container image for all runner groups | No | `DEFAULT_IMAGE` | `ghcr.io/actions/actions-runner:latest` |
| `defaultEnvVars` | Default environment variables for all runner groups (YAML map). Merged with environment variable; environment variable takes precedence | No | `DEFAULT_ENV_VARS` (JSON string) | `{"NODE_ENV": "production", "LOG_LEVEL": "info"}` |
| `runnerGroups[].name` | Runner group name (alphanumeric, hyphens, underscores only) | Yes | - | `production-runners` |
| `runnerGroups[].count` | Number of runners in this group | Yes | - | `3` |
| `runnerGroups[].githubOwner` | Override GitHub owner for this group | No | - | `different-owner` |
| `runnerGroups[].githubRepository` | Override GitHub repository for this group | No | - | `different-repo` |
| `runnerGroups[].githubToken` | Override GitHub token for this group | No | - | `ghp_yyyyyyyy` |
| `runnerGroups[].labels` | Override labels for this group | No | - | `["gpu", "high-memory"]` |
| `runnerGroups[].noDefaultLabels` | Disable default labels for this group | No | - | `true` |
| `runnerGroups[].image` | Override container image for this group | No | - | `ghcr.io/actions/actions-runner:2.327.1` |
| `runnerGroups[].envVars` | Environment variables for this runner group. Overrides defaultEnvVars for matching keys | No | - | `{"GPU_ENABLED": "true", "CUDA_VERSION": "11.8"}` |

*Required either as default values or specified individually for each runner group.
Environment variables take precedence over configuration file values.

### GitHub Token Permissions

The GitHub token requires the following permissions:
- `Administration` - Read & Write
- `Metadata` - Read-only

## Usage

1. Create a configuration file (e.g., `config.yaml`) based on the sample provided.

2. Create a `docker-compose.yml` file:

```yaml
services:
  runner-deployer:
    build: .
    command: ["-config", "/app/config.yaml"]
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./config.yaml:/app/config.yaml:ro
    restart: unless-stopped
```

3. Start the application:

```bash
docker-compose up -d
```
