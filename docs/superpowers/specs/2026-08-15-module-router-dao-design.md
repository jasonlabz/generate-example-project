# Module Router, Mockgen, and DAO Design

## Scope

This design covers three boundaries in `generate-example-project`: mock source
discovery, HTTP module composition, and persistence dependencies in Manager.
It preserves the existing health-check HTTP contract and does not add a real
database feature to the example project.

## Mock Generation

`script/go-mockgen.sh` is the single entry point. It scans the project root (or
the comma-separated `-p` roots), skips tests, generated files, vendored code,
`.mocks`, and the selected output root, then invokes the `mockgen` executable
from `PATH`. Destinations mirror the project-relative source path below `-o`.
Only files with an exported top-level interface declaration are passed to
`mockgen`; this avoids creating empty mocks for ordinary Go files.

When `-o` is under `internal/` and the source path also starts with
`internal/`, callers pass `-s internal` so the repeated segment is removed:

```text
internal/dao/user/interface.go
  -> internal/mocks/dao/user/mock_interface.go
```

Cleanup is marker-based and limited to `mock_*.go` files below the output root.
The repository no longer contains or invokes `internal/mockscan` or
`script/internal/mockscan`.

## Router Composition

`server/module.Module` is the transport registration contract. A module Wire
constructs its Controller and returns a descriptor containing optional root,
base, and v1 registration functions. `server/wire.Modules()` is the only
application module registry. Router creates nested `huma.Group` values and
passes the correct `huma.API` scope to each module:

```text
Gin Engine middleware
  -> Huma root registrations
  -> Huma service group
  -> Huma API middleware
  -> Huma version group
  -> module registrations and nested Huma groups
```

Adding a module changes its own Wire and one registry entry; `router.go` does
not gain another Controller or Wire import. Public root operations remain
outside protected API middleware. Authentication, authorization, rate limits,
or module-specific tracing belong in a child Huma Group's `UseMiddleware` and
are applied after the common API chain. Gin groups are retained only for
Knife4go's non-Huma static documentation routes.

## Manager and DAO

DAO interfaces and ORM implementations belong under `dal/db/dao` and
`dal/db/dao/impl`, respectively. Manager owns business orchestration and
depends only on the smallest DAO interfaces needed by its use cases. Service
depends on Manager, never on `*gorm.DB` or a generated DAO aggregate.

The Wire composition order is:

```text
db -> concrete DAO -> Manager -> Service -> Controller -> Module
```

Concrete DAO constructors receive an explicit database handle. Manager owns a
use-case transaction boundary when multiple DAO writes must be atomic; DAO
methods only perform persistence and return database errors. This keeps DAO
tests focused on SQL/ORM behavior, Manager tests focused on business rules,
and Service tests independent of persistence.
