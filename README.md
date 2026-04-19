<p align="center"><img src="https://raw.githubusercontent.com/ccvass/swarmex/main/docs/assets/logo.svg" alt="Swarmex" width="400"></p>

[![Test, Build & Deploy](https://github.com/ccvass/swarmex-api/actions/workflows/publish.yml/badge.svg)](https://github.com/ccvass/swarmex-api/actions/workflows/publish.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

# Swarmex API

Custom resource API server with persistent storage for Docker Swarm.

Part of [Swarmex](https://github.com/ccvass/swarmex) — enterprise-grade orchestration for Docker Swarm.

## What It Does

Provides a RESTful API for managing custom resources in the Swarmex ecosystem. Resources are stored in bbolt (embedded key-value store) and survive container restarts, enabling controllers to persist state and configuration.

## Labels

This is an infrastructure controller — no service labels required.

## How It Works

1. Exposes a REST API on port 8080 for custom resource management.
2. Stores all resources in bbolt with namespace/kind/name keys.
3. Supports full CRUD operations with JSON payloads.
4. Persists data to a mounted volume for durability across restarts.

## Quick Start

```bash
# Deploy the API server
docker service create \
  --name swarmex-api \
  --mount type=volume,src=swarmex-api-data,dst=/data \
  -p 8080:8080 \
  ghcr.io/ccvass/swarmex-api:latest

# Create a resource
curl -X POST http://localhost:8080/api/v1/resources \
  -H "Content-Type: application/json" \
  -d '{"kind": "Config", "name": "my-config", "spec": {"key": "value"}}'

# List resources
curl http://localhost:8080/api/v1/resources
```

## Verified

Full CRUD operations verified. Resources survive container restart with bbolt persistence.

## License

Apache-2.0
