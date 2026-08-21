# services/

Service packages of the EdgeCenter Terraform provider.

Each subpackage (`cdn/`, `cloud/`, `dbaas/`, `dns/`, `edgemon/`, `mkaas/`, `protection/`, `reseller/`,
`storage/`) exposes its own resources and data sources through the `Resources()` and `DataSources()`
methods and is registered in `edgecenter/provider/provider.go`.

The cloud domain is split by subdomain: `cloud/platform/`, `cloud/network/`, `cloud/compute/`,
`cloud/lb/`, `cloud/security/` and `cloud/useractions/`. Each registers itself separately.

`edgecenter.LegacyService` no longer holds any resource. The package `edgecenter` keeps the provider
schema, the client configuration and the project/region resolution shared by every cloud subpackage.
