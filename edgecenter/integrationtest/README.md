# integrationtest

`integrationtest` is the only place for unit-test-related code in this repository.

## Structure

```
integrationtest/
├── support/          # Generic foundation helpers (package support)
│   ├── case.go       # ResourceCase[T], CheckFunc, Operation, Meta
│   ├── runner.go     # RunResourceCases, DispatchCase, RunCase*
│   ├── state.go      # NewState, ApplyConfig, NewResourceDataFromState
│   ├── diag.go       # RequireNoErrorDiags, RequireHasErrorDiags, RequireOnlyErrorDiags, RequireErrorDiagContains
│   ├── sets.go       # StringSet, IntSet, List
│   └── cloud/        # Cloud-specific helpers (package cloud)
│       ├── config.go # WithProjectRegion, WithName, Merge
│       └── mock/     # Generated testify mocks + MockedCloud (package cloudmock)
│           ├── client.go       # MockedCloud, NewMockedCloud (strict), NewDefaultMockedCloud
│           ├── NetworksService.go  (generated)
│           ├── TasksService.go     (generated)
│           ├── ProjectsService.go  (generated)
│           ├── RegionsService.go   (generated)
│           ├── VolumesService.go   (generated)
│           └── generate.go         # go:generate entry point
├── cloud/            # Cloud resource unit tests
│   ├── network_test.go
│   └── …
├── cdn/              # Future: CDN resource unit tests
└── dns/              # Future: DNS resource unit tests
```

## How to write a cloud resource unit test

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
//go:build unit

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

Mock expectations are verified automatically via `t.Cleanup` — no explicit
`AssertExpectations` call needed:

```go
Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
    support.RequireNoErrorDiags(t, diags)
    require.Nil(t, state) // after delete, State() returns nil
},
```

### 4. Run

```bash
go test -tags=unit -v -count=1 ./edgecenter/integrationtest/cloud/...
```

## Patterns & conventions

- **Always use `project_id` and `region_id`** in test configs (via `cloud.WithProjectRegion`).
  Avoid `project_name`/`region_name` — they trigger additional API resolution logic.
- **Mock `Tasks` for every async resource** — nearly all cloud resources use
  `utilV2.WaitAndGetTaskInfo` or `ExecuteAndExtractTaskResult`, which call
  `client.Tasks.Get` internally.
- **Keep test-only code inside `integrationtest/`** — production packages under `edgecenter/`
  must not import test infrastructure.
- **Use `//go:build unit`** build tag to isolate unit tests from acceptance tests.
- **One factory function per case** — creates an isolated `MockedCloud` per case,
  avoiding shared mutable state between subtests.
- **Default to `NewMockedCloud`** — then add explicit project/region resolution
  expectations with `ExpectProjectResolutionTimes` / `ExpectRegionResolutionTimes`.
  Use `AllowProjectResolution` / `AllowRegionResolution` only when exact
  resolution call counts add noise.
- **No explicit `AssertExpectations` needed** — `MockedCloud.MockCleanup` is registered
  automatically via `t.Cleanup` by `RunResourceCases` and runs even if `Check` fails.
- **RunCaseRead only executes ReadContext** — state verification belongs in Check.
- **`RequireNoErrorDiags`** checks only that no `diag.Error` exists; warning-level
  diagnostics are ignored. For a completely clean happy-path (zero diagnostics of
  any severity) use **`RequireNoDiags`** instead.

## Extending to CDN/DNS

The same pattern applies:
1. Add `support/cdn/` and `support/dns/` directories with domain-specific helpers.
2. Generate mocks from the corresponding SDK packages.
3. Place tests in `integrationtest/cdn/` and `integrationtest/dns/`.



Резюме: тестинговая архитектура проекта
Три слоя тестов
Слой	Папка	Что тестирует	Скорость	Зависимости
Acceptance (E2E)	edgecenter/test/	Resource целиком через Terraform CLI + реальное API	Минуты	Креды, Vault, Terraform бинарник
Resource-level (моки SDK)	edgecenter/integrationtest/	Resource целиком (CreateContext/ReadContext/DeleteContext) с замоканным SDK (testify mocks) — проверяет интеграцию resource ↔ SDK-контракт	Миллисекунды	Нет (все замокано)
Чистые unit (если нужны)	нет пока	Отдельная функция (например, flattenNetwork) изолированно	Наносекунды	Нет
Что в edgecenter/integrationtest/
Это resource-level тесты. Они вызывают реальные функции ресурсов (resource.CreateContext, resource.ReadContext, resource.DeleteContext), но SDK-клиент замокан (testify mock).

То, что они «ходят в SDK и гоняют моки» — это и есть их суть. Они не тестируют HTTP-сериализацию или транспорт, а проверяют, что resource:

Правильно формирует запросы в SDK
Правильно обрабатывает ответы/ошибки SDK
Правильно управляет Terraform state
«Свой движок»
В edgecenter/integrationtest/support/ лежит кастомный фреймворк вместо стандартного resource.Test:

case.go — своя ResourceCase структура
runner.go — свой раннер, сам дёргает CRUD-функции напрямую
state.go — сборка terraform.InstanceState из Go-map (без HCL)
diag.go — свои ассерты на диагностики
support/cloud/mock/ — сгенерированные testify mocks на SDK-интерфейсы
Терминология (как договорились)
То, что в папке integrationtest — можно называть как угодно:

"unit tests" (по папке и тегу //go:build unit)
"integration tests" (по факту — проверка связки resource ↔ SDK)
"resource-level tests" (нейтрально)
В CI они гоняются тегом unit. От acceptance отличаются отсутствием сети, Terraform CLI, и кредов.

Что писать сейчас
Для каждого нового/изменяемого ресурса:

Resource-level тест в integrationtest/ — покрыть happy path (create/read/update/delete) + ключевые ошибки (API error, task error)
Acceptance тест в test/ — один happy path на реальном API для регрессии