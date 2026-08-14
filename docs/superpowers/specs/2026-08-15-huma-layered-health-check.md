# Huma Layered Health Check Specification

**Status:** Approved on 2026-08-15

## Goal

Turn the Huma health-check endpoint into a small, production-shaped example of
explicit dependency injection, framework-neutral response construction, and
generated gomock test doubles.

## Constraints

- Modify generate-example-project directly on master, as explicitly authorized.
- Do not modify knife4go or reintroduce Swaggo.
- Keep GET /health-check successful JSON fields exactly code, version,
  current_time, and data. Its data remains an array containing success.
- Keep the Huma documentation routes disabled and keep CreateHooks nil so Huma
  does not inject schema links into API responses.
- Use Huma v2.39.1 already selected by the project.
- Use go.uber.org/mock v0.6.0 and commit generated mock source. Generation must
  run without relying on a globally installed mockgen binary.
- Remove the health-check singleton and use explicit constructors.
- Every new exported Go symbol has a GoDoc comment.

## Architecture

The router is the composition root. It creates one local probe, passes it to a
manager, passes that manager to a service, and passes that service to the Huma
controller. Dependencies flow toward lower layers only:

    Router -> HealthCheckController -> HealthCheckService
           -> HealthCheckManager -> HealthProbe

The health-check interfaces are:

~~~go
type HealthCheckController interface {
    Register(api huma.API)
}

type HealthCheckService interface {
    Check(ctx context.Context) error
}

type HealthCheckManager interface {
    Check(ctx context.Context) error
}

type HealthProbe interface {
    Probe(ctx context.Context) error
}
~~~

The local probe is deliberately deterministic and returns nil. A real database,
Redis, or remote service check can later replace it without changing Controller,
Service, or Manager code.

## Response Adaptation

Create common/response as a framework-neutral generic envelope:

~~~go
type Envelope[T any] struct {
    Code        int
    Message     string
    ErrTrace    string
    Version     string
    CurrentTime string
    Data        T
}
~~~

The existing Gin helper keeps its public functions and Response type, including
its established JSON omission behavior. It delegates common field construction
to the shared factory before writing JSON through Gin.

Create common/humax as the Huma-facing adapter:

~~~go
type Output[T any] struct {
    Body *response.Envelope[T]
}
~~~

The Huma controller returns Output[[]string], so Huma can infer an OpenAPI
schema for data rather than receiving an any-typed response. A service failure
is adapted to a status-aware envelope error with the legacy 500 HTTP behavior.
Automatic Huma request-validation errors remain Huma-native: replacing them
would require a process-global Huma hook and would make its generated error
schema inaccurate.

## Testing

Generated mocks live in server/mocks and are produced by go generate using
go.uber.org/mock/mockgen. Tests must demonstrate one boundary at a time:

- common/response and common/humax verify response shape and timestamps.
- common/ginx verifies its existing JSON writer preserves the envelope.
- Manager tests mock HealthProbe for success and failure propagation.
- Service tests mock HealthCheckManager for success and failure propagation.
- Controller tests mock HealthCheckService and exercise Huma through httptest.
- Router tests mock HealthCheckController to prove composition delegates route
  registration, while the existing HTTP route test verifies the complete local
  graph and its OpenAPI document.

## Non-goals

- No actual database, Redis, network, retry, timeout, or health aggregation.
- No migration of unrelated Gin handlers.
- No global error-hook override in Huma.
