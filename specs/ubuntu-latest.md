# Ubuntu Actions Runner Image Analysis

**Last Updated**: 2026-01-24  
**Source**: [https://github.com/actions/runner-images/blob/ubuntu24/20260119.4/images/ubuntu/Ubuntu2404-Readme.md](https://github.com/actions/runner-images/blob/ubuntu24/20260119.4/images/ubuntu/Ubuntu2404-Readme.md)  
**Ubuntu Version**: 24.04.3 LTS  
**Image Version**: 20260119.4.1  
**Runner Version**: 2.331.0

## Overview

This document provides an analysis of the default GitHub Actions Ubuntu runner image (`ubuntu-latest`, currently Ubuntu 24.04) and guidance for creating Docker images that mimic its environment.

The GitHub Actions `ubuntu-latest` runner provides a comprehensive development environment with pre-installed tools, language runtimes, build systems, databases, and DevOps utilities. As of January 2026, `ubuntu-latest` uses Ubuntu 24.04.3 LTS.

## Included Software Summary

The runner image includes:
- **Operating System**: Ubuntu 24.04.3 LTS (Kernel ~6.8.x)
- **Language Runtimes**: Node.js (multiple versions), Python (multiple versions), Ruby, Go, Java, PHP, Rust, .NET
- **Container Tools**: Docker, containerd, Docker Compose, Buildx
- **CI/CD Tools**: GitHub CLI, Azure CLI, AWS CLI, Google Cloud SDK, Terraform
- **Databases**: PostgreSQL, MySQL, MongoDB, Redis, MS SQL
- **Build Tools**: GCC, CMake, Make, Maven, Gradle, Ant
- **Testing Tools**: Selenium, Playwright, Cypress
- **Version Control**: Git, SVN, Mercurial

## Operating System

- **Distribution**: Ubuntu 24.04.3 LTS
- **Kernel**: Linux 6.8.x (varies by release)
- **Architecture**: x86_64
- **Runner Version**: 2.331.0
- **Image Provisioner**: Hosted Compute Agent (Version: 20260115.477)

## Language Runtimes

### Node.js

The runner includes multiple Node.js versions managed via nvm:

- **Available Versions**: 18.x, 20.x, 22.x (LTS versions)
- **Default Version**: 22.x (latest LTS)
- **Package Managers**:
  - npm (bundled with Node.js)
  - yarn 1.x
  - pnpm (latest stable)

**Installation Path**: `/opt/hostedtoolcache/node/`

### Python

Multiple Python versions are pre-installed:

- **Available Versions**: 3.9, 3.10, 3.11, 3.12, 3.13
- **Default Version**: 3.12
- **Package Managers**:
  - pip (latest)
  - pipenv
  - poetry
  - virtualenv

**Installation Path**: `/opt/hostedtoolcache/Python/`

### Ruby

Multiple Ruby versions via rbenv:

- **Available Versions**: 3.1, 3.2, 3.3
- **Default Version**: 3.3.x
- **Package Manager**: gem, bundler

### Go

Multiple Go versions:

- **Available Versions**: 1.21, 1.22, 1.23
- **Default Version**: 1.23.x
- **Installation Path**: `/opt/hostedtoolcache/go/`

### Java

Multiple Java distributions (Temurin, Microsoft, Zulu):

- **Available Versions**: 11, 17, 21
- **Default Version**: 21 (LTS)
- **Build Tools**: Maven, Gradle, Ant

### PHP

Multiple PHP versions:

- **Available Versions**: 8.1, 8.2, 8.3
- **Default Version**: 8.3
- **Package Manager**: Composer

### .NET

Multiple .NET SDK versions:

- **Available Versions**: 6.0, 7.0, 8.0, 9.0
- **Default Version**: 9.0
- **Installation Path**: `/usr/share/dotnet/`

### Rust

- **Version**: Latest stable (rustc, cargo)
- **Installation Path**: `/usr/share/rust/.cargo/`

### Other Languages

- **PowerShell**: 7.x
- **Kotlin**: Latest stable
- **Swift**: Latest stable (for Linux)

## Container Tools

### Docker

- **Docker Engine**: 27.x
- **Docker Compose**: v2.30.x
- **containerd**: 1.7.x
- **runc**: Latest stable
- **Docker Buildx**: v0.17.x

**Configuration**:
- Docker daemon runs during workflow execution
- BuildKit enabled by default
- Multi-platform builds supported via buildx

### Kubernetes Tools

- **kubectl**: Latest stable (1.31.x)
- **Helm**: 3.x
- **Kind**: Latest stable (Kubernetes in Docker)
- **Minikube**: Latest stable
- **Skaffold**: Latest stable

## Build Tools

### System Compilers

- **GCC**: 13.x
- **G++**: 13.x  
- **Clang**: 18.x
- **CMake**: 3.30.x
- **Make**: 4.x
- **Autoconf/Automake**: Latest stable

### Java Build Tools

- **Apache Maven**: 3.9.x
- **Gradle**: 8.x
- **Apache Ant**: 1.10.x

### JavaScript Build Tools

- **Webpack**: Available via npm
- **Vite**: Available via npm
- **esbuild**: Available via npm
- **Rollup**: Available via npm

### .NET Build Tools

- **MSBuild**: Bundled with .NET SDK
- **NuGet**: Latest stable

## Databases & Services

### PostgreSQL

- **Version**: 16.x
- **Service Status**: Available but not running by default
- **Start Command**: `sudo systemctl start postgresql`
- **Default Port**: 5432

### MySQL

- **Version**: 8.0.x
- **Service Status**: Available but not running by default
- **Start Command**: `sudo systemctl start mysql`
- **Default Port**: 3306

### MongoDB

- **Version**: 8.x
- **Service Status**: Available but not running by default
- **Start Command**: `sudo systemctl start mongod`
- **Default Port**: 27017

### Redis

- **Version**: 7.x
- **Service Status**: Available but not running by default
- **Start Command**: `sudo systemctl start redis-server`
- **Default Port**: 6379

### MS SQL Server

- **Version**: 2022
- **Service Status**: Available but not running by default
- **Start Command**: `sudo systemctl start mssql-server`
- **Default Port**: 1433

## CI/CD Tools

### GitHub CLI

- **Version**: 2.x (gh)
- **Pre-authenticated**: No (requires GITHUB_TOKEN in workflows)

### Azure CLI

- **Version**: 2.x (az)
- **Extensions**: Common extensions pre-installed

### AWS CLI

- **Version**: 2.x (aws)
- **Configuration**: Requires credentials setup

### Google Cloud SDK

- **Version**: Latest stable (gcloud)
- **Components**: Core, bq, gsutil

### Terraform

- **Version**: 1.x
- **Installation Path**: `/usr/bin/terraform`

### Ansible

- **Version**: Latest stable (ansible-core)
- **Installation**: Via apt

### Other DevOps Tools

- **Packer**: Latest stable
- **Pulumi**: Latest stable
- **kubectl**: 1.31.x
- **Helm**: 3.x

## Testing Tools

### Selenium

- **Selenium Server**: Latest stable
- **Browser Drivers**: ChromeDriver, GeckoDriver (Firefox)

### Playwright

- **Version**: Latest stable
- **Browsers**: Chromium, Firefox, WebKit (all pre-installed)
- **Installation**: Available via npm/pip

### Cypress

- **Version**: Latest stable
- **Installation**: Available via npm
- **Browsers**: Chrome, Firefox, Edge

### Browser Testing

**Installed Browsers**:
- **Google Chrome**: Latest stable
- **Mozilla Firefox**: Latest stable  
- **Microsoft Edge**: Latest stable
- **Chromium**: Latest snapshot

## Environment Variables

Key environment variables set in the runner:

```bash
# Path configuration
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin

# GitHub Actions variables
GITHUB_WORKSPACE=/home/runner/work/<repo>/<repo>
RUNNER_TEMP=/home/runner/work/_temp
RUNNER_TOOL_CACHE=/opt/hostedtoolcache

# CI environment markers
CI=true
GITHUB_ACTIONS=true

# Language-specific paths
GOROOT=/opt/hostedtoolcache/go/<version>/x64
JAVA_HOME=/usr/lib/jvm/temurin-<version>-jdk-amd64
DOTNET_ROOT=/usr/share/dotnet

# Container tools
DOCKER_BUILDKIT=1
BUILDKIT_PROGRESS=plain
```

## System Packages

Common system packages pre-installed:

```bash
# Development essentials
build-essential
git
curl
wget
vim
nano
jq
yq

# Networking
net-tools
iputils-ping
dnsutils
netcat
telnet

# Archive tools
zip
unzip
tar
gzip
bzip2
xz-utils

# Package managers
apt-transport-https
ca-certificates
gnupg
lsb-release

# Security
openssh-client
gnupg2
pass

# Utilities
tree
htop
tmux
screen
```

## Creating a Docker Image Mimic

To create a Docker image that mimics the GitHub Actions Ubuntu runner environment, follow these patterns:

### Base Image

Start with Ubuntu 24.04:

```dockerfile
FROM ubuntu:24.04

# Set environment to prevent interactive prompts
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=UTC

# Update system packages
RUN apt-get update && apt-get upgrade -y
```

### System Setup

Install essential packages and tools:

```dockerfile
# Install build essentials and common tools
RUN apt-get install -y \
    build-essential \
    git \
    curl \
    wget \
    ca-certificates \
    gnupg \
    lsb-release \
    software-properties-common \
    apt-transport-https \
    vim \
    nano \
    jq \
    zip \
    unzip \
    && rm -rf /var/lib/apt/lists/*
```

### Language Runtimes

#### Node.js Installation

```dockerfile
# Install Node.js via NodeSource
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && \
    apt-get install -y nodejs && \
    npm install -g npm@latest yarn pnpm && \
    rm -rf /var/lib/apt/lists/*
```

#### Python Installation

```dockerfile
# Install Python versions
RUN apt-get update && apt-get install -y \
    python3.12 \
    python3.12-dev \
    python3.12-venv \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

# Install Python package managers
RUN pip3 install --upgrade pip setuptools wheel && \
    pip3 install pipenv poetry virtualenv
```

#### Go Installation

```dockerfile
# Install Go
ARG GO_VERSION=1.23.6
RUN wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz && \
    rm go${GO_VERSION}.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOROOT="/usr/local/go"
```

#### Java Installation

```dockerfile
# Install OpenJDK via apt
RUN apt-get update && apt-get install -y \
    openjdk-21-jdk \
    maven \
    gradle \
    && rm -rf /var/lib/apt/lists/*

ENV JAVA_HOME=/usr/lib/jvm/java-21-openjdk-amd64
```

### Container Tools Installation

#### Docker Installation

```dockerfile
# Install Docker
RUN curl -fsSL https://get.docker.com | sh

# Install Docker Compose
RUN curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" \
    -o /usr/local/bin/docker-compose && \
    chmod +x /usr/local/bin/docker-compose

# Install Docker Buildx
RUN mkdir -p ~/.docker/cli-plugins && \
    curl -L "https://github.com/docker/buildx/releases/latest/download/buildx-v0.17.0.linux-amd64" \
    -o ~/.docker/cli-plugins/docker-buildx && \
    chmod +x ~/.docker/cli-plugins/docker-buildx
```

### CI/CD Tools Installation

#### GitHub CLI

```dockerfile
# Install GitHub CLI
RUN mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | \
    tee /etc/apt/keyrings/githubcli-archive-keyring.gpg > /dev/null && \
    chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | \
    tee /etc/apt/sources.list.d/github-cli.list > /dev/null && \
    apt-get update && \
    apt-get install -y gh && \
    rm -rf /var/lib/apt/lists/*
```

#### Cloud CLIs

```dockerfile
# Install Azure CLI
RUN curl -sL https://aka.ms/InstallAzureCLIDeb | bash

# Install AWS CLI
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && \
    unzip awscliv2.zip && \
    ./aws/install && \
    rm -rf aws awscliv2.zip

# Install Google Cloud SDK
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | \
    tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | \
    gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg && \
    apt-get update && apt-get install -y google-cloud-sdk && \
    rm -rf /var/lib/apt/lists/*
```

### Database Installation

```dockerfile
# Install PostgreSQL client and server
RUN apt-get update && apt-get install -y \
    postgresql-16 \
    postgresql-client-16 \
    && rm -rf /var/lib/apt/lists/*

# Install MySQL
RUN apt-get update && apt-get install -y \
    mysql-server \
    mysql-client \
    && rm -rf /var/lib/apt/lists/*

# Install Redis
RUN apt-get update && apt-get install -y redis-server && \
    rm -rf /var/lib/apt/lists/*
```

### Environment Configuration

```dockerfile
# Set up environment variables to match GitHub Actions
ENV CI=true
ENV DEBIAN_FRONTEND=noninteractive
ENV PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin"
ENV DOCKER_BUILDKIT=1

# Create runner directories
RUN mkdir -p /home/runner/work /home/runner/work/_temp && \
    chmod -R 777 /home/runner
```

### Complete Dockerfile Example

Here's a minimal but comprehensive Dockerfile:

```dockerfile
FROM ubuntu:24.04

# Prevent interactive prompts
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=UTC

# Update and install essential packages
RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y \
    build-essential \
    git \
    curl \
    wget \
    ca-certificates \
    gnupg \
    lsb-release \
    software-properties-common \
    apt-transport-https \
    vim \
    nano \
    jq \
    zip \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js 22.x
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && \
    apt-get install -y nodejs && \
    npm install -g npm@latest yarn pnpm && \
    rm -rf /var/lib/apt/lists/*

# Install Python 3.12
RUN apt-get update && apt-get install -y \
    python3.12 \
    python3.12-dev \
    python3.12-venv \
    python3-pip \
    && rm -rf /var/lib/apt/lists/* && \
    pip3 install --upgrade pip setuptools wheel pipenv poetry virtualenv

# Install Go 1.23
ARG GO_VERSION=1.23.6
RUN wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz && \
    rm go${GO_VERSION}.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOROOT="/usr/local/go"

# Install Java 21
RUN apt-get update && apt-get install -y \
    openjdk-21-jdk \
    maven \
    && rm -rf /var/lib/apt/lists/*
ENV JAVA_HOME=/usr/lib/jvm/java-21-openjdk-amd64

# Install Docker
RUN curl -fsSL https://get.docker.com | sh

# Install Docker Compose
RUN curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" \
    -o /usr/local/bin/docker-compose && \
    chmod +x /usr/local/bin/docker-compose

# Install GitHub CLI
RUN mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | \
    tee /etc/apt/keyrings/githubcli-archive-keyring.gpg > /dev/null && \
    chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | \
    tee /etc/apt/sources.list.d/github-cli.list > /dev/null && \
    apt-get update && \
    apt-get install -y gh && \
    rm -rf /var/lib/apt/lists/*

# Set environment variables
ENV CI=true
ENV DOCKER_BUILDKIT=1
ENV PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

# Create runner directories
RUN mkdir -p /home/runner/work /home/runner/work/_temp && \
    chmod -R 777 /home/runner

WORKDIR /home/runner/work

CMD ["/bin/bash"]
```

### Building and Using the Image

```bash
# Build the image
docker build -t github-runner-mimic:ubuntu24.04 .

# Run a container
docker run -it --rm github-runner-mimic:ubuntu24.04

# Run with Docker-in-Docker (requires privileged mode)
docker run -it --rm --privileged \
    -v /var/run/docker.sock:/var/run/docker.sock \
    github-runner-mimic:ubuntu24.04
```

## Key Differences from Real Runner

Note aspects that cannot be perfectly replicated:

### 1. GitHub Actions Context

The real runner provides GitHub Actions-specific environment variables and context that won't be available:

- `GITHUB_WORKSPACE`, `GITHUB_REPOSITORY`, `GITHUB_SHA`
- `GITHUB_TOKEN` and authentication
- Runner context (`runner.os`, `runner.arch`, etc.)
- Workflow context (`github.event_name`, `github.ref`, etc.)

### 2. Pre-cached Dependencies

The real runner has pre-cached dependencies for faster builds:

- npm packages cache
- pip packages cache
- Maven/Gradle caches
- Docker layer cache

Your Docker image will start fresh each time.

### 3. Tool Version Management

The real runner uses specific tools for version management:

- **setup-node**, **setup-python**, **setup-go** actions for runtime switching
- Multiple runtime versions available simultaneously
- hostedtoolcache directory structure

Your Docker image will typically have single versions unless you implement similar tooling.

### 4. Service Configuration

Services in the real runner:

- May have specific configurations
- Are managed by systemd
- Can be started/stopped within workflows

In Docker, you'll need to handle service lifecycle differently (potentially using supervisord or running services in separate containers).

### 5. File System Layout

The real runner uses specific directory structures:

- `/home/runner/work/<repo>/<repo>` for workspace
- `/opt/hostedtoolcache` for language runtimes
- `/home/runner/work/_temp` for temporary files

You can mimic these, but some tools may have hardcoded paths.

### 6. Performance Characteristics

- The real runner uses cloud infrastructure with specific CPU/memory allocations
- Network connectivity may differ
- Disk I/O performance will vary

## Maintenance and Updates

### Keeping Your Image Current

The GitHub Actions runner image is updated regularly:

1. **Monitor the official repository**: [actions/runner-images](https://github.com/actions/runner-images)
2. **Check release notes**: [ubuntu24 releases](https://github.com/actions/runner-images/releases?q=ubuntu24)
3. **Update versions**: Modify your Dockerfile when new versions are released
4. **Rebuild regularly**: Set up automated builds to stay current

### Version Pinning

For reproducibility, pin versions in your Dockerfile:

```dockerfile
# Pin specific versions
ARG NODE_VERSION=22.13.1
ARG GO_VERSION=1.23.6
ARG PYTHON_VERSION=3.12.8

# Use these variables in your installation commands
```

### Testing Your Image

Compare your Docker image against the real runner:

```bash
# In your Docker container
node --version
python3 --version
go version
docker --version
gh --version

# Compare with GitHub Actions output
```

## Usage Examples

### Running Tests Locally

```bash
# Clone your repository
git clone https://github.com/your-org/your-repo.git
cd your-repo

# Run tests in the mimicked environment
docker run --rm \
    -v $(pwd):/home/runner/work/repo/repo \
    -w /home/runner/work/repo/repo \
    github-runner-mimic:ubuntu24.04 \
    npm test
```

### Debugging Workflow Issues

```bash
# Run an interactive session
docker run -it --rm \
    -v $(pwd):/home/runner/work/repo/repo \
    -w /home/runner/work/repo/repo \
    github-runner-mimic:ubuntu24.04 \
    /bin/bash

# Inside container, manually run workflow steps
```

### CI Pipeline Integration

Use your Docker image in other CI systems:

```yaml
# GitLab CI example
test:
  image: github-runner-mimic:ubuntu24.04
  script:
    - npm install
    - npm test

# CircleCI example
jobs:
  test:
    docker:
      - image: github-runner-mimic:ubuntu24.04
    steps:
      - checkout
      - run: npm install
      - run: npm test
```

## References

- **Runner Image Repository**: https://github.com/actions/runner-images
- **Documentation Source**: https://github.com/actions/runner-images/blob/ubuntu24/20260119.4/images/ubuntu/Ubuntu2404-Readme.md
- **Image Releases**: https://github.com/actions/runner-images/releases?q=ubuntu24
- **Ubuntu Documentation**: https://ubuntu.com/server/docs
- **Docker Documentation**: https://docs.docker.com/
- **GitHub Actions Documentation**: https://docs.github.com/en/actions

## Additional Resources

### Tool-Specific Documentation

- **Node.js**: https://nodejs.org/
- **Python**: https://www.python.org/
- **Go**: https://go.dev/
- **Docker**: https://www.docker.com/
- **GitHub CLI**: https://cli.github.com/

### Community Resources

- **Runner Images Discussions**: https://github.com/actions/runner-images/discussions
- **Stack Overflow**: Search for "github-actions ubuntu-latest"
- **GitHub Community Forum**: https://github.community/

---

*This document is automatically generated by the Ubuntu Actions Image Analyzer workflow.*

*Last analyzed: 2026-01-24 from runner version 2.331.0 on Ubuntu 24.04.3 LTS*
