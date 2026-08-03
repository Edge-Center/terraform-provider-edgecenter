# integrationtest

`integrationtest` is the only place for integration-test-related code in this repository.

## Where this layer sits

| Layer | Directory | What it exercises | Speed | Needs |
|-------|-----------|-------------------|-------|-------|
| Acceptance (E2E) | `edgecenter/test/` | The whole resource through the Terraform CLI against the real API | minutes | credentials, Vault, terraform binary |
| Resource-level | `edgecenter/integrationtest/` | The whole resource (`CreateContext`, `ReadContext`, `UpdateContext`, `DeleteContext`) against a mocked SDK | milliseconds | nothing |
| Unit | co-located `*_test.go` | A single function in isolation | microseconds | nothing |

Tests in this directory call the real resource functions but talk to a testify mock
instead of the SDK client. They verify that a resource builds the right SDK requests,
handles SDK responses and errors correctly, and manages Terraform state correctly. They
do not cover HTTP serialization or transport.

They are gated behind the `integration` build tag and run in CI as a separate job.
Unlike acceptance tests they need no network, no Terraform CLI and no credentials.

Instead of the stock `resource.Test` harness the directory carries its own runner in
`support/`: `case.go` (the `ResourceCase` shape), `runner.go` (drives the CRUD functions
directly), `state.go` (builds `terraform.InstanceState` from a Go map, no HCL),
`diag.go` (diagnostics assertions) and one mock package per domain.

## Structure

```
integrationtest/
├── support/          # Generic foundation helpers (package support)
│   ├── case.go       # ResourceCase[T], CheckFunc, Operation, Meta
│   ├── runner.go     # RunResourceCases, DispatchCase, RunCase*
│   ├── state.go      # NewState, ApplyConfig, NewResourceDataFromState
│   ├── diag.go       # RequireNoErrorDiags, RequireHasErrorDiags, RequireOnlyErrorDiags, RequireErrorDiagContains
│   ├── sets.go       # StringSet, IntSet, List
│   ├── cloud/        # Cloud-specific helpers (package cloud)
│   │   ├── config.go # WithProjectRegion, WithName, Merge
│   │   └── mock/     # Generated testify mocks + MockedCloud (package cloudmock)
│   │       ├── client.go       # MockedCloud, NewMockedCloud (strict), NewDefaultMockedCloud
│   │       ├── NetworksService.go  (generated)
│   │       ├── TasksService.go     (generated)
│   │       ├── ProjectsService.go  (generated)
│   │       ├── RegionsService.go   (generated)
│   │       ├── VolumesService.go   (generated)
│   │       └── generate.go         # go:generate entry point
│   ├── edgemon/      # RMON-specific helpers (package edgemon)
│   │   ├── config.go # WithName, WithReceiver, Merge
│   │   └── mock/     # Hand-written testify mocks + MockedRMON (package edgemonmock)
│   │       ├── client.go   # MockedRMON, NewMockedRMON, clientShim (implements rmon.ClientService)
│   │       └── services.go # ChannelService, StatusPageService, CheckGroupService, generic CheckService[Req,Resp]
│   ├── cdn/          # CDN-specific helpers
│   │   └── mock/     # Generated testify mocks + MockedCDN (package cdnmock)
│   │       ├── client.go        # MockedCDN, NewMockedCDN, clientShim (implements cdn.ClientService)
│   │       ├── generate.go      # go:generate entry point (mockery, one line per SDK interface)
│   │       ├── ResourceService.go, RulesService.go, OriginGroupService.go, LECertService.go,
│   │       ├── ShieldingService.go, SSLCertService.go        (generated)
│   │       └── ResourceStatisticsService.go, ResourceToolsService.go  (generated)
│   ├── dns/          # DNS-specific helpers
│   │   └── mock/     # Generated testify mock + MockedDNS (package dnsmock)
│   │       ├── client.go            # MockedDNS, NewMockedDNS, NewUnconfiguredDNS
│   │       ├── generate.go          # go:generate entry point
│   │       └── DNSClientService.go  (generated)
│   ├── storage/      # Storage-specific helpers
│   │   └── mock/     # Generated testify mocks + MockedStorage (package storagemock)
│   │       ├── client.go                 # MockedStorage, NewMockedStorage, clientShim
│   │       ├── generate.go               # go:generate entry point
│   │       ├── StorageLocationService.go (generated)
│   │       ├── StorageS3Service.go       (generated)
│   │       └── StorageBucketService.go   (generated)
│   └── protection/   # Protection-specific helpers
│       └── mock/     # Generated testify mocks + MockedProtection (package protectionmock)
│           ├── client.go            # MockedProtection, NewMockedProtection
│           ├── generate.go          # go:generate entry point
│           ├── ResourcesService.go, AliasesService.go, OriginsService.go,
│           ├── HeadersService.go, BlacklistsService.go, WhitelistsService.go,
│           └── ServicesService.go   (generated)
├── cloud/            # Cloud resource integration tests
│   ├── network_test.go
│   └── ...
├── edgemon/          # RMON (edgemon) resource integration tests
│   ├── channel_test.go
│   └── ...
├── cdn/              # CDN resource and data source integration tests
│   ├── resource_test.go
│   └── ...
├── dns/              # DNS resource and data source integration tests
│   ├── zone_test.go
│   └── ...
├── storage/          # Storage resource and data source integration tests
│   ├── s3_test.go
│   └── ...
└── protection/       # Protection resource integration tests
    ├── resource_test.go
    └── ...
```

## How to write a cloud resource integration test

### 1. Generate mocks (if new SDK interfaces are needed)

Edit `support/cloud/mock/generate.go` and add the interface name:

```go
//go:generate go run github.com/vektra/mockery/v2 --name=NewInterface --srcpkg=... --output=. --outpkg=cloudmock --testonly=false --with-expecter=false --log-level=error
```

Then run:

```bash
go generate ./edgecenter/integrationtest/support/cloud/mock/...
```

### 2. Create a test file

```go
//go:build integration

package edgecenter_test

import (
    "testing"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
    edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
    "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
    "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
    "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
    "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
    "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud"
    cloudmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud/mock"
)
```

### 3. Build a case factory

Each test case is built by a factory function that:
1. Creates `mc := cloudmock.NewMockedCloud(projectID, regionID)`
2. Adds explicit resolution expectations such as `cloudmock.ExpectProjectResolutionTimes(...)`
   or `cloudmock.ExpectRegionResolutionTimes(...)`
3. If exact resolution counts are not important, uses permissive helpers
   `cloudmock.AllowProjectResolution(...)` or `cloudmock.AllowRegionResolution(...)`
4. Sets testify expectations on `mc.Tasks`, `mc.Networks`, etc.
5. Returns `support.ResourceCase[*cloudmock.MockedCloud]`
6. Uses `cloud.Merge(cloud.WithProjectRegion(...), cloud.WithName(...))` for config

`MockedCloud` implements `support.MetaProvider`, so `RunResourceCases`
automatically passes `mc.Config` as Terraform `meta`. The fake and meta stay
bound to the same fixture object without an extra `MetaFunc`.

Create/Read operations call `InitCloudClient` which resolves project/region
via `Projects.List` and/or `Regions.List` depending on which fields the test
config uses. Prefer explicit counts with `cloudmock.ExpectProjectResolutionTimes(...)`
and `cloudmock.ExpectRegionResolutionTimes(...)`. Use permissive
`Allow*Resolution(...)` helpers only when resolution is incidental to the
behavior under test.

Mock expectations are verified automatically via `t.Cleanup` - no explicit
`AssertExpectations` call needed:

```go
Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
    support.RequireNoErrorDiags(t, diags)
    require.Nil(t, state) // after delete, State() returns nil
},
```

### 4. Run

```bash
go test -tags=integration -v -count=1 ./edgecenter/integrationtest/cloud/...
```

## Patterns & conventions

- **Always use `project_id` and `region_id`** in test configs (via `cloud.WithProjectRegion`).
  Avoid `project_name`/`region_name` - they trigger additional API resolution logic.
- **Mock `Tasks` for every async resource** - nearly all cloud resources use
  `utilV2.WaitAndGetTaskInfo` or `ExecuteAndExtractTaskResult`, which call
  `client.Tasks.Get` internally.
- **Keep test-only code inside `integrationtest/`** - production packages under `edgecenter/`
  must not import test infrastructure.
- **Use `//go:build integration`** build tag to isolate integration tests from acceptance tests.
- **One factory function per case** - creates an isolated `MockedCloud` per case,
  avoiding shared mutable state between subtests.
- **Default to `NewMockedCloud`** - then add explicit project/region resolution
  expectations with `ExpectProjectResolutionTimes` / `ExpectRegionResolutionTimes`.
  Use `AllowProjectResolution` / `AllowRegionResolution` only when exact
  resolution call counts add noise.
- **No explicit `AssertExpectations` needed** - `MockedCloud.MockCleanup` is registered
  automatically via `t.Cleanup` by `RunResourceCases` and runs even if `Check` fails.
- **RunCaseRead only executes ReadContext** - state verification belongs in Check.
- **`RequireNoErrorDiags`** checks only that no `diag.Error` exists; warning-level
  diagnostics are ignored. For a completely clean happy-path (zero diagnostics of
  any severity) use **`RequireNoDiags`** instead.

## edgemon (RMON)

`support/edgemon/` follows the same layout as `support/cloud/`. The difference is
in the mock: the RMON SDK exposes `rmon.ClientService` (an interface returning one
service per resource), so `MockedRMON` wires hand-written testify mocks instead of
generated ones. `MockedRMON` implements `MetaProvider` (`TestMeta` -> `*edgecenter.Config`)
and `MockCleanuper`, exactly like `MockedCloud`. The six check kinds share one generic
mock `CheckService[Req, Resp]` because their SDK service is the generic
`checks.Service[Req, Resp]`. Tests live in `integrationtest/edgemon/` and set
expectations directly on `mc.Channel`, `mc.CheckHTTP`, `mc.StatusPage`, etc.

## CDN

`support/cdn/` follows `support/cloud/`: the eight SDK interfaces are mocked with
mockery (`support/cdn/mock/generate.go`), and `client.go` wires them into a
`MockedCDN` plus a `clientShim` that implements `cdn.ClientService`. Regenerate with:

```bash
go generate ./edgecenter/integrationtest/support/cdn/mock/...
```

`MockedCDN` implements `MetaProvider` (`TestMeta` -> `*edgecenter.Config`) and
`MockCleanuper`, so `RunResourceCases` binds meta and verifies expectations
automatically. Tests live in `integrationtest/cdn/` and set expectations on
`mc.Resources`, `mc.Rules`, `mc.OriginGroups`, `mc.LECerts`, `mc.Shielding`,
`mc.SSLCerts`, `mc.Tools`.

The nested `options` blocks (37 on the CDN resource, 34 on the rule) are not covered
one by one here. They are covered wholesale by a co-located white-box test,
`services/cdn/options_test.go`: it fills **every** option and **every** field of every
option via reflection, pushes the struct through `d.Set` -> `d.Get`, converts it back,
and compares. Any mapper that drops or mangles a field fails there, by option name.

Data sources are covered too: they are plain `*schema.Resource` values fetched from
`provider.Provider().DataSourcesMap` and driven with `support.OpRead`. Because
`ResourceData.State()` returns nil for an empty ID, a data source case must set a
non-empty placeholder `CurrentID` so the config materializes into state; the read
then assigns the real ID.

## DNS

DNS differs from CDN and edgemon in one way: the DNS SDK exports a concrete
`*dnssdk.Client` and no client interface. The seam is therefore declared at the
consumer, in `edgecenter/config.go`:

```go
type DNSClientService interface { CreateZone(...); Zone(...); RRSet(...); ... }
```

`Config.DNSClient` holds that interface, `*dnssdk.Client` satisfies it (there is a
compile-time assertion next to the declaration), and `support/dns/mock` holds a single
mockery-generated mock of it. Regenerate with:

```bash
go generate ./edgecenter/integrationtest/support/dns/mock/...
```

Whenever a DNS resource starts calling a new SDK method, add it to `DNSClientService`
and re-run the generator.

`MockedDNS` implements `MetaProvider` and `MockCleanuper` like the others, so
`RunResourceCases` binds meta and verifies expectations automatically. Because there is
one client rather than one service per resource, expectations go on `mc.Client`:

```go
mc := dnsmock.NewMockedDNS()
mc.Client.On("Zone", mock.Anything, "example.com").Return(dnssdk.Zone{Name: "example.com"}, nil)
```

`dnsmock.NewUnconfiguredDNS()` returns a `Config` with a nil `DNSClient`. That is the
state of a provider without `edgecenter_dns_api`, and it is what `checkDNSDependency`
(the wrapper every DNS CRUD function goes through) must reject.

The record mappers are not exercised one by one here. `services/dns/record_mappers_test.go`
covers `fillRRSet`, `listToFailoverMeta`, `failoverMetaToList` and `verifyFailoverMeta`
as co-located white-box tests, the same way `services/cdn/options_test.go` covers the
CDN option blocks.

## Storage

The storage SDK has the same shape problem as DNS and one extra twist. It exports a
concrete `*storageSDK.SDK` and no interface, so the seam again lives in
`edgecenter/config.go`, split by role to mirror the SDK's own composition:

```go
type StorageClientService interface {
    StorageLocationService
    StorageS3Service
    StorageBucketService
}
```

`support/storage/mock` holds one generated mock per role and a `clientShim` that embeds
all three, so expectations land on the role that owns the call:

```go
mc := storagemock.NewMockedStorage()
mc.Locations.On("LocationsList", anyOpts(1)...).Return(locations, nil)
mc.Storages.On("StoragesList", anyOpts(3)...).Return([]models.Storage{st}, nil)
mc.Buckets.On("BucketsList", anyOpts(2)...).Return(list, nil)
```

Regenerate with:

```bash
go generate ./edgecenter/integrationtest/support/storage/mock/...
```

The twist: every storage SDK method takes variadic functional options rather than
values, so testify matches on the option closures themselves. Two helpers in
`storage/common_test.go` handle that. `anyOpts(n)` builds the `mock.Anything` list, and
its `n` is a real assertion - it pins how many options the provider builds for that
call. `appliedOpts[T]` replays the closures onto a zero `T`, which is exactly what the
SDK does, so a test can assert on the resulting request:

```go
mc.Storages.On("CreateStorage", anyOpts(4)...).
    Run(func(args mock.Arguments) { sent = appliedOpts[storages.StorageCreateHTTPParams](args) }).
    Return(&created, nil)
```

Assert on `sent` inside `Check`, not inside `Run` - `Run` has no `*testing.T`.

Schema flags, both name validators and the two id parsers are covered co-located in
`services/storage/schema_test.go`.

## Protection

Protection is the easy case, and the only one that needed no change in
`edgecenter/config.go`. The SDK already exports one interface per service
(`ResourcesService`, `AliasesService`, `OriginsService`, `HeadersService`,
`BlacklistsService`, `WhitelistsService`, `ServicesService`) **and** holds them as
exported fields on `protectionSDK.Client`. So there is nothing to shim: the mock is a
real client with mocks in its fields.

```go
mc.Config = &edgecenter.Config{ProtectionClient: &protectionSDK.Client{
    Resources:  mc.Resources,
    Aliases:    mc.Aliases,
    ...
}}
```

`Config.ProtectionClient` therefore stays `*protection.Client`, unlike `DNSClient` and
`StorageClient`. Regenerate the mocks with:

```bash
go generate ./edgecenter/integrationtest/support/protection/mock/...
```

`MockedProtection` implements `MetaProvider` and `MockCleanuper` like the others.
Expectations go on the service that owns the call - `mc.Resources`, `mc.Origins`,
`mc.Headers`, `mc.Blacklists`, `mc.Whitelists`, `mc.Aliases`.

Two things worth knowing before adding cases here:

- Seven of the eight resources use a composite id `<resource_id>:<child_id>` parsed by
  `edgecenter.ImportStringParserSimple`; `edgecenter_protection_resource_certificate` is
  the exception and uses the protected resource id itself.
- The tests are **characterization tests**: several of them pin behaviour that is wrong
  on purpose, so that a fix shows up as a failing test rather than a silent change. Each
  such case says so in its name (for example
  `read leaves the client attribute empty because the api sends a number into a string
  attribute`). The matching defects are written up in
  `tsks/terraform-provider-edgecenter/bugs/protection/`.

Schema flags and all four validators are covered co-located in
`services/protection/schema_test.go`.

## What to write for a new resource

1. A resource-level test here: the happy path (create, read, update, delete) plus the
   errors that matter (API failure, task failure, malformed id).
2. An acceptance test in `edgecenter/test/`: one happy path against the real API, as a
   regression net.
