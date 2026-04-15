## Repository Description
- `iam-service` provides a GraphQL API for user and role management in Platform Mesh.
- It manages users, roles, and related authorization state through OpenFGA, Keycloak, and KCP-backed resources.
- This is a Go service built around [gqlgen](https://github.com/99designs/gqlgen), [OpenFGA](https://github.com/openfga/openfga), [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime), and [multicluster-runtime](https://github.com/kubernetes-sigs/multicluster-runtime).
- Read the org-wide [AGENTS.md](https://github.com/platform-mesh/.github/blob/main/AGENTS.md) for general conventions.

## Core Principles
- Keep changes small and local. Prefer the narrowest fix that addresses the real problem.
- Verify behavior before finishing. Start with targeted tests, then broader validation if needed.
- Prefer existing `task` targets over ad-hoc shell commands.
- Keep human-facing process details in `CONTRIBUTING.md`.

## Project Structure
- `cmd`: CLI and service startup entrypoints.
- `graph`: GraphQL schema and gqlgen inputs.
- `pkg/resolver`: GraphQL resolver logic.
- `pkg/router`, `pkg/middleware`: HTTP routing and request middleware.
- `pkg/fga`: OpenFGA integration and tuple management.
- `pkg/keycloak`: identity provider integration.
- `pkg/accountinfo`, `pkg/workspace`, `pkg/roles`: domain services.
- `pkg/config`, `pkg/cache`, `pkg/context`: runtime configuration and request context helpers.
- `input/roles.yaml`: role definitions consumed by the service.

## Architecture
This is a GraphQL service with thin transport wiring and most domain behavior split between resolver services, OpenFGA integration, Keycloak integration, and KCP-aware middleware.

### Runtime model
- `cmd/server.go` starts an `mcmanager.Manager` backed by an APIExport provider for `core.platform-mesh.io`; leader election is intentionally disabled.
- The HTTP router is chi-based and serves `/graphql`, `/healthz`, and `/readyz`; the GraphQL playground is exposed only in local mode.
- The manager's local config is also reused to derive a root-cluster KCP client and account/workspace helpers.

### Request flow
- Middleware built in `setupRouter` injects KCP user context before GraphQL execution.
- GraphQL directives are part of the authorization path: the `Authorized` directive uses OpenFGA plus `AccountInfo` and workspace lookups.
- `pkg/resolver/pm.Service` is the main service layer behind resolvers; it composes FGA, Keycloak, pagination, sorting, and user transformation.

### Domain model
- User and role mutations are backed by OpenFGA tuples and role definitions from `input/roles.yaml`, not a relational database.
- Keycloak is the identity source for user lookup/enrichment; OpenFGA is the authorization source.
- KCP workspace and `AccountInfo` lookups provide tenant and workspace context needed to resolve resource-scoped permissions.

### Configuration and tests
- `cmd/server.go` derives the root KCP host by stripping the multicluster path from the manager config; changing that logic affects all cluster lookups.
- GraphQL schema, mocks, and Keycloak client code are generated and must stay in sync with source definitions.

## Commands
- `task fmt` — format Go code.
- `task lint` — run formatting plus golangci-lint.
- `task build` — build the service.
- `task run` — run the service locally using `.env`.
- `task unittest` — run tests and write `cover.out`.
- `task test` — run the standard local test flow.
- `task cover` — enforce coverage using `.testcoverage.yml`.
- `task mockery` — refresh generated mocks when interfaces change.
- `task generate` — regenerate mocks and `go generate` outputs.
- `task generate:keycloak` — regenerate the minimal Keycloak client from `keycloak-minimal.json`.
- `task validate` — run format, lint, build, and coverage checks together.
- `go test ./...` — fast fallback for targeted verification.

## Code Conventions
- Follow existing GraphQL and service-layer patterns before introducing new abstractions.
- Update `graph/schema.graphql` first for GraphQL API changes, then regenerate code.
- Keep resolver code in `pkg/resolver` and integration-specific logic in the corresponding package.
- Add or update `_test.go` files alongside changed behavior.
- Regenerate mocks and generated clients instead of hand-editing generated output.
- Keep logs structured and never log secrets, tokens, or Keycloak client secrets.

## Generated Artifacts
- Run `task generate` after changing GraphQL schema, codegen inputs, or interfaces used by mocks.
- Run `task generate:keycloak` after changing `keycloak-minimal.json` or `oapi-codegen.yaml`.
- Review generated changes carefully before mixing them with manual logic changes.
- Do not hand-edit gqlgen, mockery, or oapi-codegen output.

## Do Not
- Edit gqlgen, mockery, or oapi-codegen output by hand.
- Change `graph/schema.graphql` without regenerating the related code.
- Update `.testcoverage.yml` unless the task explicitly requires it.

## Hard Boundaries
- Do not invent new local workflows when a `task` target already exists.
- Treat auth, tenant-context, and role-management changes as high-risk; verify them explicitly.
- Ask before changing release flow, CI wiring, published image behavior, or Helm integration outside this repo.

## Human-Facing Guidance
- Use `README.md` for local certificate setup, startup arguments, and service context.
- Use `CONTRIBUTING.md` for contribution process, DCO, and broader developer workflow expectations.
