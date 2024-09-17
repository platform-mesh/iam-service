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

