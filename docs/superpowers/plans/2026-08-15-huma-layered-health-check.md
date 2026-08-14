# Huma Layered Health Check Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make health-check a complete Huma, Controller, Service, Manager, and
Probe example with reusable response adapters and generated gomock tests.

**Architecture:** The router explicitly composes local probe, manager, service,
and controller. Common response construction is independent of Gin and Huma;
thin adapters bind it to each transport. Mocks represent only consumed
interfaces, keeping each unit test deterministic.

**Tech Stack:** Go 1.25, Gin, Huma v2.39.1, go.uber.org/mock v0.6.0, httptest.

## Global Constraints

- Preserve the successful GET /health-check JSON contract.
- Keep Huma docs paths empty and CreateHooks nil.
- Do not add a real external health dependency.
- Use constructors; do not use a health-check singleton.
- Commit generated mock source and use go generate without a global executable.

---

### Task 1: Shared response model and transport adapters

**Files:**
- Create: common/response/response.go
- Test: common/response/response_test.go
- Create: common/humax/response.go
- Test: common/humax/response_test.go
- Modify: common/ginx/response.go
- Test: common/ginx/response_test.go

**Interfaces:**
- Produces: response.Envelope[T], response.New, humax.Output[T], and
  humax.InternalServerError.

- [ ] **Step 1: Write failing response tests**

~~~go
func TestNew_SetsSuccessEnvelope(t *testing.T) {
    envelope := response.New("v1", []string{"success"})
    if envelope.Code != 0 || !reflect.DeepEqual(envelope.Data, []string{"success"}) {
        t.Fatal("unexpected success envelope")
    }
    if _, err := time.Parse(time.DateTime, envelope.CurrentTime); err != nil {
        t.Fatalf("invalid current_time: %v", err)
    }
}
~~~

- [ ] **Step 2: Run response tests and verify they fail because the package is absent**

Run: go test ./common/response ./common/humax
Expected: FAIL with package or symbol not found.

- [ ] **Step 3: Implement the smallest shared envelope and Huma output adapter**

~~~go
func New[T any](version string, data T) *Envelope[T]
func NewError[T any](version string, data T, code int, message, trace string) *Envelope[T]
func Success[T any](version string, data T) *Output[T]
func InternalServerError(version string, cause error) *Error
~~~

- [ ] **Step 4: Verify the response tests pass**

Run: go test ./common/response ./common/humax
Expected: PASS.

- [ ] **Step 5: Write a Gin writer regression test, verify its existing output first, then delegate Gin response creation to response.New and response.NewError**

Run: go test ./common/ginx
Expected before and after the mechanical refactor: PASS with data serialized as a
one-element array. This is a contract-preserving refactor, so the output test is
a baseline guard rather than an artificial red test.

- [ ] **Step 6: Commit**

~~~text
git add common/response common/humax common/ginx
git commit -m "feat: add shared API response adapters"
~~~

### Task 2: Dependency contracts and generated mocks

**Files:**
- Create: server/manager/health_check.go
- Modify: server/service/health_check.go
- Modify: server/controller/health_check.go
- Create: server/mocks/generate.go
- Create: server/mocks/mock_controller.go
- Create: server/mocks/mock_service.go
- Create: server/mocks/mock_manager.go
- Modify: go.mod
- Modify: go.sum

**Interfaces:**
- Produces: HealthCheckController.Register, HealthCheckService.Check,
  HealthCheckManager.Check, and HealthProbe.Probe.

- [ ] **Step 1: Define the four narrow interfaces and their GoDoc comments**

~~~go
func (c *Controller) Register(api huma.API)
func (s *Service) Check(ctx context.Context) error
func (m *Manager) Check(ctx context.Context) error
func (p *LocalProbe) Probe(ctx context.Context) error
~~~

- [ ] **Step 2: Add go.uber.org/mock v0.6.0 and three package-mode generation directives**

~~~go
//go:generate go run go.uber.org/mock/mockgen -destination=mock_service.go -package=mocks github.com/jasonlabz/generate-example-project/server/service HealthCheckService
~~~

- [ ] **Step 3: Generate mocks and compile all packages**

Run: go generate ./server/mocks && go test ./...
Expected: generated mocks compile and existing route test is updated in later tasks.

- [ ] **Step 4: Commit**

~~~text
git add go.mod go.sum server/manager/health_check.go server/service/health_check.go server/controller/health_check.go server/mocks
git commit -m "test: add generated health-check mocks"
~~~

### Task 3: Manager and Service behavior

**Files:**
- Create: server/manager/health_check/health_check_impl.go
- Test: server/manager/health_check/health_check_impl_test.go
- Modify: server/service/health_check/health_check_impl.go
- Test: server/service/health_check/health_check_impl_test.go

**Interfaces:**
- Consumes: manager.HealthProbe and manager.HealthCheckManager.
- Produces: NewLocalProbe, NewManager, and NewService.

- [ ] **Step 1: Write failing Manager tests with MockHealthProbe**

~~~go
mockProbe.EXPECT().Probe(gomock.Any()).Return(probeErr)
err := health_check.NewManager(mockProbe).Check(context.Background())
if !errors.Is(err, probeErr) {
    t.Fatalf("got %v, want wrapped probe error", err)
}
~~~

- [ ] **Step 2: Run Manager tests and verify they fail because NewManager is absent**

Run: go test ./server/manager/health_check
Expected: FAIL with NewManager undefined.

- [ ] **Step 3: Implement LocalProbe and Manager with wrapped probe errors**

~~~go
func NewLocalProbe() manager.HealthProbe
func NewManager(probe manager.HealthProbe) manager.HealthCheckManager
func (m *Manager) Check(ctx context.Context) error {
    if err := m.probe.Probe(ctx); err != nil {
        return fmt.Errorf("probe health: %w", err)
    }
    return nil
}
~~~

- [ ] **Step 4: Run Manager tests and verify they pass**

Run: go test ./server/manager/health_check
Expected: PASS.

- [ ] **Step 5: Write failing Service tests with MockHealthCheckManager**

~~~go
mockManager.EXPECT().Check(gomock.Any()).Return(managerErr)
err := health_check.NewService(mockManager).Check(context.Background())
if !errors.Is(err, managerErr) {
    t.Fatalf("got %v, want wrapped manager error", err)
}
~~~

- [ ] **Step 6: Run Service tests, implement delegation with error context, and run them green**

Run: go test ./server/service/health_check
Expected before implementation: FAIL with NewService undefined or old DoCheck mismatch.
Expected after implementation: PASS.

- [ ] **Step 7: Commit**

~~~text
git add server/manager/health_check server/service/health_check
git commit -m "feat: layer health-check service and manager"
~~~

### Task 4: Huma controller and router composition

**Files:**
- Modify: server/controller/health_check.go
- Test: server/controller/health_check_test.go
- Modify: server/router/router.go
- Modify: server/router/router_test.go

**Interfaces:**
- Consumes: service.HealthCheckService and controller.HealthCheckController.
- Produces: NewHealthCheckController and a router-local composition function.

- [ ] **Step 1: Write a failing controller HTTP test with MockHealthCheckService**

~~~go
mockService.EXPECT().Check(gomock.Any()).Return(nil)
request := httptest.NewRequest(http.MethodGet, "/health-check", nil)
response := httptest.NewRecorder()
router.ServeHTTP(response, request)
if response.Code != http.StatusOK {
    t.Fatalf("got %d, want 200", response.Code)
}
~~~

- [ ] **Step 2: Run controller test and verify it fails because NewHealthCheckController is absent**

Run: go test ./server/controller
Expected: FAIL with NewHealthCheckController undefined.

- [ ] **Step 3: Implement the typed Huma controller**

~~~go
func NewHealthCheckController(service service.HealthCheckService) HealthCheckController
func (c *Controller) Register(api huma.API)
~~~

The success handler calls Check and returns humax.Success("v1",
[]string{"success"}); it returns humax.InternalServerError("v1", err) on
failure.

- [ ] **Step 4: Run controller tests and verify success envelope plus error envelope**

Run: go test ./server/controller
Expected: PASS.

- [ ] **Step 5: Write a failing router delegation test with MockHealthCheckController**

~~~go
mockController.EXPECT().Register(gomock.Any())
registerRootAPI(api, mockController)
~~~

- [ ] **Step 6: Wire the production graph in router, preserving the docs configuration, then run router tests green**

Run: go test ./server/router
Expected: root route JSON contract, no Link header or $schema, debug documentation,
and controller registration delegation all pass.

- [ ] **Step 7: Commit**

~~~text
git add server/controller server/router
git commit -m "feat: wire layered Huma health-check"
~~~

### Task 5: Final generation and verification

**Files:**
- Modify: generated mock files only if go generate changes them.

- [ ] **Step 1: Regenerate and inspect generated files**

Run: go generate ./... && git diff --check
Expected: no regeneration drift and no whitespace errors.

- [ ] **Step 2: Run full validation**

Run: go test ./... && go test -race ./... && go vet ./... && go build ./...
Expected: all commands exit 0.

- [ ] **Step 3: Review the specification checklist against the diff**

Verify explicit constructors, no singleton, response contract, mock coverage, and
no Knife4go or Swaggo changes.

- [ ] **Step 4: Commit**

~~~text
git add go.mod go.sum common server
git commit -m "test: verify layered health-check example"
~~~
