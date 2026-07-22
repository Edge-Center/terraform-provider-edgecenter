# v2migrate

Converts a terraform project from deprecated v1 resources to their V2
counterparts without destroying and recreating infrastructure:

| v1 (deprecated) | V2 |
|----|----|
| `edgecenter_instance` | `edgecenter_instanceV2` |
| `edgecenter_loadbalancer` | `edgecenter_loadbalancerv2` + `edgecenter_lblistener` |
| `edgecenter_reseller_images` | `edgecenter_reseller_imagesV2` |

The tool rewrites `.tf` manifests to the V2 schema (keeping formatting and
comments), rewrites references across the project, and generates a
`v2-migrate.tf` file with `removed` + `import` blocks that move the existing
cloud objects under the new resource types. Everything that cannot be converted
mechanically is commented out with a `TODO(v2migrate)` marker and listed in the
report instead of being guessed.

## Install

```shell
go install github.com/Edge-Center/terraform-provider-edgecenter/cmd/v2migrate@latest
```

Or from a repository clone:

```shell
make build_v2migrate
# the binary lands in bin/v2migrate
```

## Quick start

```shell
cd my-project
terraform state pull > terraform.tfstate   # only for remote backends
v2migrate -dir . -dry-run                  # preview
v2migrate -dir .                           # convert
terraform init -upgrade
terraform plan -out=v2-migrate.tfplan      # must be: N to import, 0 to add, 0 to destroy
terraform apply v2-migrate.tfplan
rm v2-migrate.tf
terraform plan                             # must be: No changes
```

Requires terraform >= 1.7 and a provider version with the V2 resources.

Full guide with the TODO reference and limitations:
[docs/guides/v1-to-v2-migration.md](../../docs/guides/v1-to-v2-migration.md)
(rendered on the registry as the "Migrating v1 resources to V2" guide).

## Flags

- `-dir` - directory with the terraform configuration (default `.`);
- `-state` - path to `terraform.tfstate`, defaults to `<dir>/terraform.tfstate` when present;
- `-migrations` - path of the generated migration file (default `<dir>/v2-migrate.tf`);
- `-report` - also write the report to a file;
- `-dry-run` - print the report without writing any files.
