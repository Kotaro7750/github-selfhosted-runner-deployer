# GitHub Self-hosted Runner Deployer

An application that automatically deploys and manages self-hosted runners for GitHub Actions. It runs as a Docker container and dynamically creates and removes runners for specified GitHub repositories.

## Features

- **Dynamic scaling**: Automatically recovers when runners exit
- **Ephemeral runners**: Runs runners in ephemeral mode, automatically removing them after job completion

## Required Environment Variables

Before running the application, set the following environment variables:

| Environment Variable | Description | Example |
|---------------------|-------------|---------|
| `GITHUB_OWNER` | GitHub repository owner name | `octocat` |
| `GITHUB_REPO` | GitHub repository name | `my-repo` |
| `GITHUB_TOKEN` | GitHub Personal Access Token | `ghp_xxxxxxxxxxxx` |

### GitHub Token Permissions

The GitHub token requires the following permissions:
- `Administration` - Read & Write
- `Metadata` - Read-only

## Usage

Example `docker-compose.yml`:

```yaml
services:
  runner-deployer:
    build: .
    environment:
      - GITHUB_OWNER=your-github-username
      - GITHUB_REPO=your-repository-name
      - GITHUB_TOKEN=your-github-token
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    restart: unless-stopped
```

```bash
# After creating docker-compose.yml, run the following command
docker-compose up -d
```
