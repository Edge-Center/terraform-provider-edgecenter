# v1 to v2 conversion report

## Converted

- edgecenter_loadbalancer.farm -> edgecenter_loadbalancerv2.farm (lb.tf:1)
- edgecenter_instance.workers -> edgecenter_instanceV2.workers (main.tf:9)
- edgecenter_instance.min -> edgecenter_instanceV2.min (main.tf:33)

## State migration

- v2-migrate.tf: 3 removed block(s), 5 import block(s)
- some import ids could not be resolved from state and contain <placeholders>, fill them before applying

## Manual attention required (TODO markers in config)

- lb.tf:8 edgecenter_loadbalancerv2.farm: listener: the nested listener moved to a standalone edgecenter_lblistener resource, cannot extract automatically when the parent uses count or for_each, create the edgecenter_lblistener resource manually
- main.tf:16 edgecenter_instanceV2.workers: volume: boot_index must be a literal number to classify this volume, move the block to boot_volumes or data_volumes manually
- main.tf:23 edgecenter_instanceV2.workers: type: any_subnet is not supported in V2, use subnet with an explicit subnet_id
- main.tf:27 edgecenter_instanceV2.workers: metadata: deprecated metadata blocks are gone in V2, move the entries into the metadata map attribute
- main.tf:44 edgecenter_instanceV2.min: type is missing: type is required in V2, set subnet, external or reserved_fixed_ip

## Warnings

- module.net.edgecenter_instance.inmodule: v1 resource lives in a child module, not migrated
- edgecenter_instance.ghost: v1 resource exists in state but not in the configuration, not migrated
- main.tf:33 edgecenter_instance.min: no state entry found, the import id in v2-migrate.tf is a placeholder, fill it manually

## Mechanical changes

- main.tf:22 edgecenter_instanceV2.workers: is_default = true set: exactly one interface must have is_default = true, the converter marked the first one, if the first plan wants to replace interfaces move is_default to the interface terraform reports as default
- main.tf:39 edgecenter_instanceV2.min: boot_index = 0 added
- main.tf:44 edgecenter_instanceV2.min: is_default = true set: exactly one interface must have is_default = true, the converter marked the first one, if the first plan wants to replace interfaces move is_default to the interface terraform reports as default

## Next steps

1. Review the rewritten manifests and every TODO(v2migrate) marker.
2. Make sure the provider version in required_providers supports the V2 resources, then run terraform init -upgrade.
3. Run terraform plan -out=v2-migrate.tfplan and check it only imports and forgets resources, no destroy and no create.
4. Run terraform apply v2-migrate.tfplan.
5. Delete v2-migrate.tf and run terraform plan again, it must show no changes.
6. If the plan wants to replace an instance interface, move is_default = true to the interface terraform reports as default.
