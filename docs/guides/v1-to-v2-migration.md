---
page_title: "Migrating v1 resources to V2 with v2migrate"
subcategory: ""
description: |-
  How to convert manifests and state from deprecated v1 resources to their V2 versions without destroying infrastructure.
---

# Migrating v1 resources to V2

Some resources have a second version and the first one is deprecated:

| v1 | V2 |
|----|----|
| `edgecenter_instance` | `edgecenter_instanceV2` |
| `edgecenter_loadbalancer` | `edgecenter_loadbalancerv2` |
| `edgecenter_reseller_images` | `edgecenter_reseller_imagesV2` |

Renaming a resource type in a manifest by hand makes terraform plan a destroy and a create.
The `v2migrate` tool converts a project in place without recreating anything:

- rewrites `.tf` manifests to the V2 schema, keeping formatting and comments;
- rewrites references (`edgecenter_instance.web.id`, `data.edgecenter_instance...`) across all files;
- extracts the nested load balancer `listener` block into a standalone `edgecenter_lblistener` resource;
- generates `v2-migrate.tf` with `removed` (with `lifecycle { destroy = false }`) and `import` blocks that move the existing objects under the new resource types;
- everything that cannot be converted mechanically is commented out with a `TODO(v2migrate)` marker and listed in the report instead of being guessed.

## Install

```shell
go install github.com/Edge-Center/terraform-provider-edgecenter/cmd/v2migrate@latest
```

Or build from a repository clone:

```shell
make build_v2migrate
```

The binary lands in `bin/v2migrate`.

## Usage

```shell
v2migrate -dir ./my-project
```

Flags:

- `-dir` - directory with the terraform configuration (default `.`);
- `-state` - path to `terraform.tfstate`; defaults to `<dir>/terraform.tfstate` when present. For remote backends run `terraform state pull > terraform.tfstate` first. Without a state the import ids are generated as `<placeholders>`;
- `-migrations` - path of the generated migration file (default `<dir>/v2-migrate.tf`);
- `-report` - also write the report to a file;
- `-dry-run` - print the report without writing anything.

## Migration flow

1. Run `v2migrate -dir <project>` and read the report; resolve every `TODO(v2migrate)` marker it lists.
2. Make sure `required_providers` allows a provider version with the V2 resources, then run `terraform init -upgrade`.
3. `terraform plan -out=v2-migrate.tfplan` - the plan must only import and forget resources: no destroy, no create.
4. `terraform apply v2-migrate.tfplan`.
5. Delete `v2-migrate.tf` and run `terraform plan` - it must show `No changes`.

Requires terraform >= 1.7 (`removed` blocks with `destroy = false`).

## What to expect per resource

`edgecenter_instance`: `volume` blocks are split into `boot_volumes` (`boot_index = 0` or absent) and `data_volumes`, `interface` becomes `interfaces`, `metadata_map` becomes `metadata`, `userdata` becomes `user_data`. Exactly one interface gets `is_default = true` (the one with the lowest `order`, or the first). Interface `security_groups`, `port_security_disabled` and floating ip fields are commented out: in V2 they are managed by the `edgecenter_instance_port_security` and `edgecenter_floatingip` resources. If the first plan after import wants to replace an interface, move `is_default = true` to the interface terraform reports as default. Fields that V2 does not read back (`user_data`, `keypair_name`, `password`, `configuration`) show a harmless in-place update in the migration plan.

`edgecenter_loadbalancer`: the nested `listener` becomes a standalone `edgecenter_lblistener` resource with its own import. `vip_network_id` and `vip_subnet_id` are commented out: they are create time only, V2 never reads them back after import, and re-adding them would plan a load balancer replacement.

`edgecenter_reseller_images`: `reseller_id` becomes `entity_id` plus `entity_type = "reseller"`. Import uses the `<entity_type>:<entity_id>` format.

## Limitations

- Child modules are not converted automatically: run the tool in each module directory and adjust the `removed`/`import` addresses with the `module.<name>` prefix by hand.
- `*.tf.json` files are not converted.
- Resources that exist only in state (not in the configuration) are reported but not migrated.

## Adding a new v1/V2 pair

The engine is generic. A pair is described declaratively by one YAML file in `internal/converter/rules/`; adding a file is enough, no converter code changes. The test suite validates every rule file against the real provider schemas (`TestRulesMatchProviderSchemas`), so a typo in a rule fails CI.
