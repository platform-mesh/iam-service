# Contributing to Platform Mesh IAM Service

We want to make contributing to this project as easy and transparent as possible.

## Project Overview

This is the Platform Mesh IAM (Identity and Access Management) service, a Go-based microservice that provides a GraphQL API for user management and authorization. The service has been refactored to use OpenFGA as the primary backend for authorization data and KCP for multi-cluster resource management, eliminating the need for a traditional database. It integrates with Keycloak for identity management.

## Development Setup

### Prerequisites
1. Go 1.25.1+ (check [go.mod](go.mod) for exact version)
2. Platform Mesh installation (OpenFGA and KCP)
3. Task runner (optional but recommended)

### Environment Setup
1. Copy `.env.sample` to `.env`
2. Configure required services:
   - OpenFGA server (OPENFGA_GRPC_ADDR)
   - Keycloak (if using IDM features)
   - Kubernetes context (KUBECONFIG)

## Development Commands

### Building and Running
```bash
# Run locally (requires .env file from .env.sample)
task run
# OR
go run ./main.go serve

# Build the project
task build
# OR
go build ./...
```

### Testing
```bash
# Run all tests with coverage
task test

# Run unit tests only
task unittest
# OR
go test ./...

# Run specific package tests (e.g., middleware/kcp)
go test -v ./pkg/middleware/kcp

# Check test coverage (requires 95% total, 80% file/package)
task cover

# Generate detailed coverage reports
go test ./pkg/middleware/kcp -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Code Quality and Generation
```bash
# Format code
task fmt

# Run linting
task lint

# Generate code (GraphQL, mocks, etc.)
task generate

# Generate mocks only
task mockery

# Full validation pipeline (format, lint, build, coverage)
task validate
```

## Architecture

### High-Level Structure
- **Entry Point**: `main.go` → `cmd/` → `cmd/server.go`
- **Transport Layer**: GraphQL API via gqlgen
- **Service Layer**: Business logic in `pkg/service/` and `pkg/resolver/`
- **Integration Layer**: OpenFGA (gRPC client), Keycloak for identity management, KCP for multi-cluster management
- **Data Backend**: OpenFGA for authorization data, KCP for resource management (no traditional database)

### Key Components

#### GraphQL API (`graph/schema.graphql`, `pkg/resolver/`)
- User management queries and mutations
- Role-based access control operations
- Schema-first approach using gqlgen
- Resolvers in `pkg/resolver/schema.resolvers.go`

#### OpenFGA Integration (`pkg/fga/`)
- Fine-grained authorization service
- gRPC client for OpenFGA server
- Authorization middleware and store helpers

#### Identity Management (`pkg/service/idm/keycloak/`)
- Keycloak integration for user identity
- Token validation and user synchronization
- Multi-tenant support

#### KCP Integration (`pkg/middleware/kcp/`)
- Kubernetes Control Plane integration
- Multi-cluster resource management
- API export provider setup
- JWT token validation and tenant-based authorization

## Development Patterns

### Code Generation
- GraphQL code generation via gqlgen (run `task generate`)
- Mock generation via mockery (configured in `.mockery.yaml`)
- Generated files should not be manually edited

### Authorization Flow
1. JWT token validation through middleware
2. Tenant context extraction
3. OpenFGA authorization checks
4. Role-based permission evaluation

### Multi-tenancy
- Tenant information extracted from JWT tokens
- Tenant-scoped authorization through OpenFGA
- KCP-based multi-cluster resource management

## Common Development Tasks

### Adding New GraphQL Operations
1. Update `graph/schema.graphql`
2. Run `task generate` to update generated code
3. Implement resolver in `pkg/resolver/schema.resolvers.go`
4. Add service layer logic if needed
5. Write tests and update mocks

### Working with OpenFGA Integration
1. Update FGA service layer in `pkg/service/fga/`
2. Modify store helpers in `internal/pkg/fga/`
3. Add new authorization middleware if needed
4. Generate/update mocks with `task mockery`

## Pull Requests

You are welcome to contribute with your pull requests. These steps explain the contribution process:

1. Fork the repository and create your branch from `main`.
2. [Add tests](#testing) for your code.
3. If you've changed APIs, update the documentation.
4. Make sure the tests pass. Our GitHub Actions pipeline runs unit and e2e tests for your PR.
5. Sign the Developer Certificate of Origin (DCO).

## Testing Strategy

- Unit tests for all packages (`*_test.go`)
- High coverage requirements (95% total, 80% per file/package)
- Mock interfaces for external dependencies
- Integration testing with OpenFGA and Keycloak

> **NOTE:** You should always add tests when adding code to our repository.

## Technology Stack
- **Language**: Go 1.25.1
- **GraphQL**: gqlgen for schema-first GraphQL API
- **Authorization**: OpenFGA for fine-grained access control (gRPC client)
- **Identity**: Keycloak integration for user management
- **Multi-cluster**: KCP (Kubernetes Control Plane) for resource management
- **Build**: Task (Taskfile.yaml)
- **Testing**: Standard Go testing with testify
- **Logging**: zerolog for structured logging

## Current Architecture Status

This repository is currently on the `feat/service-redesign` branch undergoing a major refactoring:

**What was removed:**
- PostgreSQL database and GORM models
- gRPC API endpoints
- Contract tests and database packages
- Traditional database-driven user/role management

**What remains:**
- GraphQL API only (no gRPC)
- OpenFGA integration for authorization backend
- KCP integration for multi-cluster resource management
- Keycloak integration for identity management

**Development Context:**
- The service now uses OpenFGA as the primary data store for authorization relationships
- KCP handles multi-cluster resource coordination instead of database operations
- All data persistence is handled through external services (OpenFGA, KCP, Keycloak)
- Focus is on IAM operations through GraphQL resolvers that coordinate between these services

## Issues
We use GitHub issues to track bugs. Please ensure your description is
clear and includes sufficient instructions to reproduce the issue.

## License
By contributing to Platform Mesh, you agree that your contributions will be licensed
under its [Apache-2.0 license](LICENSE).
