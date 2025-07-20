# 🚀 CI/CD Setup for XDS Control Plane Operator

Quick setup guide for automated Docker builds and releases.

## 📋 Prerequisites

1. **GitHub Repository** - Your repository on GitHub
2. **DockerHub Account** - Account on [DockerHub](https://hub.docker.com/)
3. **Repository Access** - Admin access to configure secrets

## ⚙️ Setup Steps

### 1. Create DockerHub Access Token

1. Go to [DockerHub](https://hub.docker.com/) → **Account Settings** → **Security**
2. Click **New Access Token**
3. Name: `GitHub Actions - xds-cp-operator`
4. Permissions: **Read, Write, Delete**
5. Copy the generated token ⚠️ **Save it safely!**

### 2. Configure GitHub Secrets

1. Go to your GitHub repository
2. Navigate to **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret** and add:

| Secret Name | Value | Description |
|-------------|-------|-------------|
| `DOCKERHUB_USERNAME` | `okassov` | Your DockerHub username |
| `DOCKERHUB_TOKEN` | `dckr_pat_xxxx...` | Token from step 1 |

### 3. Verify Workflows

The following workflows are already configured:

- ✅ **`.github/workflows/docker-build-push.yml`** - Builds and pushes Docker images
- ✅ **`.github/workflows/release.yml`** - Creates GitHub releases

## 🎯 What Happens Next

### Automatic Docker Builds

- **On push to main**: Builds `okassov/xds-cp-operator:main` and `:latest`
- **On version tags**: Builds versioned images (e.g., `v1.0.0`, `v1.0`, `v1`)
- **Multi-platform**: Supports `linux/amd64` and `linux/arm64`
- **Security scanning**: Trivy scans for vulnerabilities

### Automatic Releases

When you push a version tag:

```bash
git tag v1.0.1
git push origin v1.0.1
```

Automatically creates:
- 🐳 Docker images on DockerHub
- 📦 GitHub release with changelog
- 📄 Kubernetes manifests for installation
- 🔍 Security scan results

## 🔍 Monitoring

### View Build Status

- **GitHub Actions**: Repository → **Actions** tab
- **README Badges**: Status badges show build status
- **DockerHub**: Check image uploads at hub.docker.com

### Troubleshooting

**Common issues:**

1. **"Authentication failed"**
   - Check `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` secrets
   - Verify token permissions

2. **"Repository not found"**
   - Update `IMAGE_NAME` in workflow files
   - Ensure DockerHub repository exists

3. **"Tests failed"**
   - All tests must pass before Docker build
   - Check Go 1.21 compatibility

## 🎉 Success!

Once configured, your CI/CD pipeline will:

- ✅ **Automatically build** Docker images on every commit
- ✅ **Run tests** and security scans
- ✅ **Create releases** with proper versioning
- ✅ **Publish to DockerHub** with multi-platform support
- ✅ **Generate manifests** for easy Kubernetes deployment

Your users can now install with:

```bash
kubectl apply -f https://github.com/okassov/xds-cp-operator/releases/latest/download/xds-cp-operator-crds.yaml
kubectl apply -f https://github.com/okassov/xds-cp-operator/releases/latest/download/xds-cp-operator.yaml
```

## 🔄 Next Release

For your next release:

1. Update `CHANGELOG.md` with new features
2. Create and push a new tag: `git tag v1.0.1 && git push origin v1.0.1`
3. CI/CD handles the rest automatically! 🚀 