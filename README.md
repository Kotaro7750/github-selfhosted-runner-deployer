# GitHub Self-hosted Runner Deployer

An application that automatically deploys and manages self-hosted runners for GitHub Actions. It runs as a Docker container and dynamically creates and removes runners for specified GitHub repositories.

## Features

- **Dynamic scaling**: Automatically recovers when runners exit
- **Ephemeral runners**: Runs runners in ephemeral mode, automatically removing them after job completion

## Configuration

The application uses a YAML configuration file to define GitHub settings and runner groups.

### Configuration File Format

Create a YAML configuration file with the following structure:

```yaml
defaultGithubOwner: "your-github-username"
defaultGithubRepository: "your-repository-name"
defaultGithubToken: "your-github-token"

runnerGroups:
  - name: group-1
    count: 3
    # Optional: Override default settings for this group
    # githubOwner: "specific-owner"
    # githubRepository: "specific-repo"
    # githubToken: "specific-token"
  - name: group-2
    count: 1
```

### Configuration Fields

| Field | Description | Required | Example |
|-------|-------------|----------|---------|
| `defaultGithubOwner` | Default GitHub repository owner name | Yes* | `Kotaro7750` |
| `defaultGithubRepository` | Default GitHub repository name | Yes* | `my-repo` |
| `defaultGithubToken` | Default GitHub Personal Access Token | Yes* | `ghp_xxxxxxxxxxxx` |
| `runnerGroups[].name` | Runner group name (alphanumeric, hyphens, underscores only) | Yes | `production-runners` |
| `runnerGroups[].count` | Number of runners in this group | Yes | `3` |
| `runnerGroups[].githubOwner` | Override GitHub owner for this group | No | `different-owner` |
| `runnerGroups[].githubRepository` | Override GitHub repository for this group | No | `different-repo` |
| `runnerGroups[].githubToken` | Override GitHub token for this group | No | `ghp_yyyyyyyy` |

*Required either as default values or specified individually for each runner group.

### GitHub Token Permissions

The GitHub token requires the following permissions:
- `Administration` - Read & Write
- `Metadata` - Read-only

## Usage

1. Create a configuration file (e.g., `config.yaml`) with your GitHub settings and runner groups:

```yaml
defaultGithubOwner: "your-github-username"
defaultGithubRepository: "your-repository-name"
defaultGithubToken: "your-github-token"

runners:
  - name: production-runners
    count: 2
  - name: test-runners
    count: 1
    githubRepository: "test-repo"
```

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
