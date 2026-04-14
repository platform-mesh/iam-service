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
