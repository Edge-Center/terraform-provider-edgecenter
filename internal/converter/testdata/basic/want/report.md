# v1 to v2 conversion report

## Converted

- data.edgecenter_instance.web -> data.edgecenter_instanceV2.web (data.tf:1)
- edgecenter_loadbalancer.lb -> edgecenter_loadbalancerv2.lb (lb.tf:1)
- data.edgecenter_loadbalancer.lb -> data.edgecenter_loadbalancerv2.lb (lb.tf:26)
- edgecenter_instance.web -> edgecenter_instanceV2.web (main.tf:38)
- edgecenter_reseller_images.rimgs -> edgecenter_reseller_imagesV2.rimgs (reseller.tf:1)
- data.edgecenter_reseller_images.cur -> data.edgecenter_reseller_imagesV2.cur (reseller.tf:15)

## Extracted resources

- edgecenter_loadbalancer.lb nested block -> edgecenter_lblistener.lb (lb.tf)

## State migration

- v2-migrate.tf: 3 removed block(s), 4 import block(s)

## Manual attention required (TODO markers in config)

- data.tf:8 data.edgecenter_instanceV2.web: reference interface: interface outputs are exposed as the interfaces list in the V2 data source
- lb.tf:6 edgecenter_loadbalancerv2.lb: vip_network_id: vip_network_id is create time only and ForceNew, V2 does not read it back after import, keep it commented or the plan would replace the load balancer
- lb.tf:7 edgecenter_loadbalancerv2.lb: vip_subnet_id: vip_subnet_id is create time only and ForceNew, V2 does not read it back after import, keep it commented or the plan would replace the load balancer
- lb.tf:13 edgecenter_lblistener.lb: insert_x_forwarded: insert_x_forwarded is create time only and is not read back on import, re-adding it right after import would plan a listener replacement
- main.tf:54 edgecenter_instanceV2.web: delete_on_termination: delete_on_termination is not supported in V2, volume deletion is controlled by the edgecenter_volume resource
- main.tf:62 edgecenter_instanceV2.web: security_groups: interface security_groups are gone in V2, manage them with the edgecenter_instance_port_security resource
- main.tf:69 edgecenter_instanceV2.web: port_security_disabled: port_security_disabled is gone in V2, manage it with the edgecenter_instance_port_security resource
- main.tf:84 edgecenter_instanceV2.web: reference interface: interface outputs moved to the interfaces set in V2, sets are not index addressable

## Mechanical changes

- main.tf:53 edgecenter_instanceV2.web: data_volumes.boot_index removed: data_volumes have no boot_index in V2
- main.tf:59 edgecenter_instanceV2.web: interfaces.order removed: order was replaced by is_default in V2
- main.tf:66 edgecenter_instanceV2.web: is_default = true set: exactly one interface must have is_default = true, the converter marked the first one, if the first plan wants to replace interfaces move is_default to the interface terraform reports as default
- main.tf:68 edgecenter_instanceV2.web: interfaces.order removed: order was replaced by is_default in V2
- reseller.tf:1 edgecenter_reseller_imagesV2.rimgs: entity_type = "reseller" added
- reseller.tf:15 data.edgecenter_reseller_imagesV2.cur: entity_type = "reseller" added

## Next steps

1. Review the rewritten manifests and every TODO(v2migrate) marker.
2. Make sure the provider version in required_providers supports the V2 resources, then run terraform init -upgrade.
3. Run terraform plan -out=v2-migrate.tfplan and check it only imports and forgets resources, no destroy and no create.
4. Run terraform apply v2-migrate.tfplan.
5. Delete v2-migrate.tf and run terraform plan again, it must show no changes.
6. If the plan wants to replace an instance interface, move is_default = true to the interface terraform reports as default.
