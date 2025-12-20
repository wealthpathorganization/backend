# CI/CD Build Configuration Guide

This guide provides configuration examples for fixing Go build cache permission issues in various CI/CD platforms.

## Problem

When building Go applications in CI/CD agents, you may encounter:

```
go: open /Users/.../Library/Caches/go-build/...: operation not permitted
```

This happens because Go tries to write to default cache directories that aren't accessible in restricted environments.

## Solution

Set `GOCACHE` and `GOMODCACHE` environment variables to writable directories (typically `/tmp`).

---

## GitHub Actions

### Method 1: Using Environment Variables (Recommended)

```yaml
name: Build

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    
    env:
      GOCACHE: /tmp/go-cache
      GOMODCACHE: /tmp/go-mod
      GOTMPDIR: /tmp/go-build
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      
      - name: Create cache directories
        run: mkdir -p /tmp/go-cache /tmp/go-mod /tmp/go-build
      
      - name: Build
        run: make build
```

### Method 2: Using Make Target

```yaml
- name: Build
  run: make build-ci
  working-directory: ./backend
```

---

## GitLab CI

```yaml
variables:
  GOCACHE: /tmp/go-cache
  GOMODCACHE: /tmp/go-mod
  GOTMPDIR: /tmp/go-build

build:
  image: golang:1.24-alpine
  
  before_script:
    - mkdir -p /tmp/go-cache /tmp/go-mod /tmp/go-build
  
  script:
    - cd backend
    - make build
  
  artifacts:
    paths:
      - backend/bin/api
    expire_in: 1 week
```

---

## Jenkins

```groovy
pipeline {
    agent {
        docker {
            image 'golang:1.24-alpine'
        }
    }
    
    environment {
        GOCACHE = '/tmp/go-cache'
        GOMODCACHE = '/tmp/go-mod'
        GOTMPDIR = '/tmp/go-build'
    }
    
    stages {
        stage('Prepare') {
            steps {
                sh 'mkdir -p /tmp/go-cache /tmp/go-mod /tmp/go-build'
            }
        }
        
        stage('Build') {
            steps {
                dir('backend') {
                    sh 'make build'
                }
            }
        }
    }
}
```

---

## CircleCI

```yaml
version: 2.1

jobs:
  build:
    docker:
      - image: cimg/go:1.24
    
    environment:
      GOCACHE: /tmp/go-cache
      GOMODCACHE: /tmp/go-mod
      GOTMPDIR: /tmp/go-build
    
    steps:
      - checkout
      
      - run:
          name: Create cache directories
          command: mkdir -p /tmp/go-cache /tmp/go-mod /tmp/go-build
      
      - run:
          name: Build
          command: |
            cd backend
            make build
      
      - store_artifacts:
          path: backend/bin/api

workflows:
  build-workflow:
    jobs:
      - build
```

---

## Travis CI

```yaml
language: go

go:
  - "1.24"

env:
  global:
    - GOCACHE=/tmp/go-cache
    - GOMODCACHE=/tmp/go-mod
    - GOTMPDIR=/tmp/go-build

before_install:
  - mkdir -p /tmp/go-cache /tmp/go-mod /tmp/go-build

script:
  - cd backend
  - make build
```

---

## Azure Pipelines

```yaml
trigger:
  - main

pool:
  vmImage: 'ubuntu-latest'

variables:
  GOCACHE: '/tmp/go-cache'
  GOMODCACHE: '/tmp/go-mod'
  GOTMPDIR: '/tmp/go-build'

steps:
  - task: GoTool@0
    inputs:
      version: '1.24'
  
  - script: |
      mkdir -p /tmp/go-cache /tmp/go-mod /tmp/go-build
    displayName: 'Create cache directories'
  
  - script: |
      cd backend
      make build
    displayName: 'Build application'
  
  - task: PublishBuildArtifacts@1
    inputs:
      PathtoPublish: 'backend/bin/api'
      ArtifactName: 'api-binary'
```

---

## Bitbucket Pipelines

```yaml
image: golang:1.24

pipelines:
  default:
    - step:
        name: Build
        caches:
          - go
        script:
          - export GOCACHE=/tmp/go-cache
          - export GOMODCACHE=/tmp/go-mod
          - export GOTMPDIR=/tmp/go-build
          - mkdir -p /tmp/go-cache /tmp/go-mod /tmp/go-build
          - cd backend
          - make build
        artifacts:
          - backend/bin/api
```

---

## Docker Build

The Dockerfile is already configured correctly:

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Set Go cache directories to writable locations
ENV GOCACHE=/tmp/go-cache
ENV GOMODCACHE=/tmp/go-mod

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api ./cmd/api
```

---

## Local Development

For local development, you generally don't need these environment variables since your user has proper cache permissions. However, if you want to use them:

```bash
# Option 1: Set for current session
export GOCACHE=/tmp/go-cache
export GOMODCACHE=/tmp/go-mod
make build

# Option 2: Use the setup script
source backend/setup-build-env.sh
make build

# Option 3: Use the CI target
make build-ci
```

---

## Testing the Fix

After implementing the fix, verify it works:

```bash
# Clean previous builds
make clean

# Try building
make build

# Check cache directories were created
ls -la /tmp/ | grep go-
```

You should see:
```
drwxr-xr-x  ... go-cache
drwxr-xr-x  ... go-mod
```

---

## Troubleshooting

### Still Getting Permission Errors?

1. Verify the environment variables are set:
   ```bash
   echo $GOCACHE
   echo $GOMODCACHE
   ```

2. Check if `/tmp` is writable:
   ```bash
   touch /tmp/test && rm /tmp/test
   ```

3. Try using a different directory:
   ```bash
   export GOCACHE=$HOME/.cache/go-build
   export GOMODCACHE=$HOME/go/pkg/mod
   ```

### Cache Taking Too Much Space?

Clean up cache directories periodically:

```bash
# Clean Go cache
go clean -cache -modcache -testcache

# Or manually remove
rm -rf /tmp/go-cache /tmp/go-mod
```

---

## Additional Resources

- [Go Build Cache Documentation](https://pkg.go.dev/cmd/go#hdr-Build_and_test_caching)
- [Go Environment Variables](https://pkg.go.dev/cmd/go#hdr-Environment_variables)

