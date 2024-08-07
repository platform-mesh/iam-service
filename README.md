> [!WARNING]
> This Repository is under construction and not yet ready for public consumption. Please check back later for updates.


# openMFP - iam-service
![Build Status](https://github.com/openmfp/iam-service/actions/workflows/pipeline.yml/badge.svg)

## Description

The openMFP iam-service exposes a graphql and a grpc API. The Graphql API is primarily used by user management UIs, while the GRPC API is used to authorize write calls into openfga.

## Features
- backend for frontend API's to manage user data
- write GRPC API to validate write requests into the FGA schema

## Architecture overview
`iam-service`has 2 base layers
- DB 
- Core layer, which is responsible for business logic. It is being called by consumer(or Transport layer) and interacts with all other layers(Hooks, etc.)
Core layers also is responsible for the proper error handling and logging.

## No-Op
If there is no actual implementation for an interface, you can find a no-op implementation in the `./pkg/interfaces/no-op` package. 

## Packages

### graph

This package contains the GraphQL models and resolvers as reusable code. The `graph/openmfp.graphql` file contains the schema.

## Getting started

TBD

## Releasing

The release is performed automatically through a GitHub Actions Workflow.
All the released versions will be available through access to GitHub (as any other Golang Module).

## Requirements

The iam-service requires a installation of go. Checkout the [go.mod](go.mod) for the required go version and dependencies.

## Contributing

Please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) file in this repository for instructions on how to contribute to openMFP.

## Code of Conduct

Please refer to the [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) file in this repository informations on the expected Code of Conduct for contributing to openMFP.

## Licensing

Copyright 2024 SAP SE or an SAP affiliate company and openMFP contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/openmfp/account-operator).
