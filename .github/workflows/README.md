# GitHub Actions Configuration

This directory contains GitHub Actions workflows for the XDS Control Plane Operator project.

## 🚀 Workflows

### 1. `docker-build-push.yml` - Build and Push Docker Images
- **Triggers**: Push to main, tags starting with `v*`, pull requests
- **Features**:
  - Multi-platform builds (linux/amd64, linux/arm64)
  - Automatic tagging based on git tags
  - Caching for faster builds
  - Security scanning with Trivy
  - Code quality checks with staticcheck

### 2. `release.yml` - Automated Releases
- **Triggers**: Push tags starting with `v*`
- **Features**:
  - Creates GitHub releases automatically
  - Generates release notes from CHANGELOG.md
  - Builds and uploads Kubernetes manifests
  - Includes Docker image installation instructions

## 🔧 Setup Requirements

### DockerHub Configuration

To enable Docker image publishing, you need to configure the following secrets in your GitHub repository:

1. Go to your repository on GitHub
2. Navigate to **Settings** → **Secrets and variables** → **Actions**
3. Add the following repository secrets:

| Secret Name | Description | Example Value |
|-------------|-------------|---------------|
| `DOCKERHUB_USERNAME` | Your DockerHub username | `okassov` |
| `DOCKERHUB_TOKEN` | DockerHub access token | `dckr_pat_xxxxxxxxxxxx` |

### Creating DockerHub Access Token

1. Log in to [DockerHub](https://hub.docker.com/)
2. Go to **Account Settings** → **Security**
3. Click **New Access Token**
4. Give it a descriptive name (e.g., "GitHub Actions - xds-cp-operator")
5. Select **Read, Write, Delete** permissions
6. Copy the generated token and add it as `DOCKERHUB_TOKEN` secret

## 📋 Workflow Behavior

### Docker Image Tags

The workflows automatically create the following Docker image tags:

| Trigger | Tag | Example |
|---------|-----|---------|
| Push to main | `main` | `okassov/xds-cp-operator:main` |
| Version tag | Version number | `okassov/xds-cp-operator:v1.0.0` |
| Version tag | Major.minor | `okassov/xds-cp-operator:v1.0` |
| Version tag | Major | `okassov/xds-cp-operator:v1` |
| Main branch | `latest` | `okassov/xds-cp-operator:latest` |

### Release Process

1. **Create a tag**: `git tag v1.0.1 && git push origin v1.0.1`
2. **Automatic actions**:
   - Docker image is built and pushed to DockerHub
   - GitHub release is created with release notes
   - Kubernetes manifests are attached to the release
   - Security scan is performed

## 🔍 Monitoring

### Workflow Status

You can monitor workflow status in several ways:

1. **GitHub Repository**: Go to the **Actions** tab
2. **README Badges**: Status badges are displayed in the main README
3. **Email Notifications**: GitHub sends emails on workflow failures

### Troubleshooting

#### Docker Push Failures
- Verify `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` secrets are correctly set
- Check that the DockerHub access token has sufficient permissions
- Ensure the repository name matches your DockerHub namespace

#### Test Failures
- Go builds are tested with Go 1.21
- All tests must pass before Docker image is built
- Static analysis checks must pass

#### Security Scan Issues
- Trivy scans Docker images for vulnerabilities
- Results are uploaded to GitHub Security tab
- High-severity vulnerabilities may require base image updates

## 🛠️ Local Testing

You can test the Docker build process locally:

```bash
# Build multi-platform image locally
docker buildx create --use
docker buildx build --platform linux/amd64,linux/arm64 -t xds-cp-operator:test .

# Run security scan locally
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd):/workspace aquasec/trivy:latest image xds-cp-operator:test
```

## 📝 Customization

### Changing Docker Repository

To use a different DockerHub repository:

1. Update the `IMAGE_NAME` environment variable in both workflow files
2. Update the DockerHub secrets accordingly
3. Update documentation and examples

### Adding Additional Platforms

To build for additional platforms (e.g., `linux/arm/v7`):

1. Edit the `platforms` field in `docker-build-push.yml`
2. Ensure your base image supports the target platform

### Custom Release Notes

The release workflow automatically extracts release notes from `CHANGELOG.md`. To customize:

1. Update the changelog parsing logic in `release.yml`
2. Modify the release body template
3. Add additional release assets if needed

## 🎯 Best Practices

1. **Always test locally** before pushing tags
2. **Update CHANGELOG.md** before creating releases
3. **Use semantic versioning** for tags (v1.2.3)
4. **Monitor security scan results** regularly
5. **Keep access tokens secure** and rotate them periodically 