# services/

Service packages of the EdgeCenter Terraform provider.

Each subpackage (`cdn/`, `dns/`, `edgemon/`, `protection/`, `reseller/`, `storage/`) exposes its own
resources and data sources through the `Resources()` and `DataSources()` methods and is
registered in `edgecenter/provider/provider.go`.

Everything that has not been split out yet lives in `edgecenter.LegacyService`.
