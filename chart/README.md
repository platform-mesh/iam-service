# Get started
## Prerequisites

You might need to install the following components:
```bash
istioctl install --set profile=default
helm repo add external-secrets https://charts.external-secrets.io
k apply -f https://raw.githubusercontent.com/external-secrets/external-secrets/v0.10.3/deploy/crds/bundle.yaml
```

## Configuration

Change values.yaml file to suit your needs.

If you want to run in-memory database:
1. Set inMemory to true
2. Set postgresNeedsProvision to false.

If you want to connect to existing postgres database:
1. Set inMemory to false
2. Set postgresNeedsProvision to false.

If you want to provision a new postgres database:
1. Set inMemory to false
2. Set postgresNeedsProvision to true.

If you want to provision a new postgres database but use existing secret for password:
1. Set inMemory to false
2. Set postgresNeedsProvision to true.
3. Set global.auth.postgres.secretName to the name of the existing secret. 
Also make sure that the secret's key corresponds global.auth.postgres.secretKeys.adminPasswordKey. Change it if needed.

## DataLoader

To seed Postgresql and FGA store with the initial data, you can use DataLoader job.

This job does 3 things:
1. Imports FGA schema
2. Loads data to FGA store
3. Loads data to Postgresql

### Prerequisites

1. Postgresql
2. OpenFGA server

### Golang configuration

Dataloader uses the following fields from the `../intenral/pkg/config.Config` struct:

1. `Config.Database` must reflect your postgresql setup.
2. `Config.Openfga` must reflect your FGA server setup.

### Quickstart

Dataloader needs the following params:
1. `schema` - path to the FGA schema file
2. `file` - path to the FGA data
3. `tenants` - list of tenants to load data for

#### Terminal

```bash 
go run main.go dataload --schema=./chart/assets/schema.fga  --file=./chart/assets/data.yaml --tenants=tenant1,tenant2
```

#### Chart

[!Note] Chart is not yet implemented fully.

To use Dataloader in your charts, you need to:
1. Enable DataLoader in values.yaml by setting `features.useDataLoader` to true.
2. Run helm install command including values-dataloader.yaml file:
```bash
helm install my-app ./chart/ -f ./chart/values.yaml -f ./chart/values-dataloader.yaml
```




