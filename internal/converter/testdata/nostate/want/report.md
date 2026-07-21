# v1 to v2 conversion report

## Converted

- edgecenter_instance.solo -> edgecenter_instanceV2.solo (main.tf:1)

## State migration

- v2-migrate.tf: 1 removed block(s), 1 import block(s)
- some import ids could not be resolved from state and contain <placeholders>, fill them before applying

## Warnings

- : no state file given, import ids are placeholders, run with -state or fill them manually
- main.tf:1 edgecenter_instance.solo: no state entry found, the import id in v2-migrate.tf is a placeholder, fill it manually

## Mechanical changes

- main.tf:13 edgecenter_instanceV2.solo: is_default = true set: exactly one interface must have is_default = true, the converter marked the first one, if the first plan wants to replace interfaces move is_default to the interface terraform reports as default

## Next steps

1. Review the rewritten manifests and every TODO(v2migrate) marker.
2. Make sure the provider version in required_providers supports the V2 resources, then run terraform init -upgrade.
3. Run terraform plan -out=v2-migrate.tfplan and check it only imports and forgets resources, no destroy and no create.
4. Run terraform apply v2-migrate.tfplan.
5. Delete v2-migrate.tf and run terraform plan again, it must show no changes.
6. If the plan wants to replace an instance interface, move is_default = true to the interface terraform reports as default.
