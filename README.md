> [!WARNING]
> This Repository is under development and not ready for productive use. It is in an alpha stage. That means APIs and concepts may change on short notice including breaking changes or complete removal of apis.

<!-- trigger -->

# Platform Mesh - iam-service
![Build Status](https://github.com/platform-mesh/iam-service/actions/workflows/pipeline.yml/badge.svg)

## Local dev

To run the application locally, create `.env` config file from `.env.sample` and run:
```shell
go run ./main.go serve
```

## Description

The Platform Mesh IAM service is a Go-based microservice that provides a GraphQL API for user management and authorization. The service uses:

- **OpenFGA** for fine-grained authorization backend
- **KCP** for multi-cluster resource management
- **Keycloak** for identity management

## Features
- GraphQL API for user and role management
- Multi-tenant authorization through OpenFGA
- Multi-cluster resource coordination via KCP
- Keycloak integration for identity provider support

## Architecture

This service has been refactored to eliminate traditional database dependencies, instead using:
- OpenFGA as the authorization data backend
- KCP for Kubernetes resource management
- Keycloak for user identity management

## Quick Start

### Prerequisites
1. Go 1.25.1+
2. Platform Mesh installation (OpenFGA and KCP)
3. Task runner (optional)

### Development Setup
1. Copy `.env.sample` to `.env` and configure services
2. Run the service: `go run ./main.go serve`
3. Load test data: `go run main.go dataload --schema=$SCHEMA_PATH --file=$DATA_PATH --tenants=tenant1,tenant2`

For detailed development information, architecture details, and contribution guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).

## Releasing

The release is performed automatically through a GitHub Actions Workflow.
All the released versions will be available through access to GitHub (as any other Golang Module).

## Requirements

The iam-service requires a installation of go. Checkout the [go.mod](go.mod) for the required go version and dependencies.

## Contributing

Please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) file in this repository for instructions on how to contribute to Platform Mesh.

## Code of Conduct

Please refer to the [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) file in this repository information on the expected Code of Conduct for contributing to Platform Mesh.

## Licensing

Copyright 2024 SAP SE or an SAP affiliate company and Platform Mesh contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/platform-mesh/iam-service).
