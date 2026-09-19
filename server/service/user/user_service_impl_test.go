package user

import (
	"context"
	"errors"
	"testing"

	potatoErrors "github.com/jasonlabz/potato/errors"
	"go.uber.org/mock/gomock"

	"github.com/jasonlabz/generate-example-project/common/apperr"
	manager_mocks "github.com/jasonlabz/generate-example-project/mocks/server/manager/user"
	"github.com/jasonlabz/generate-example-project/server/manager/user"
)

// assertCatalogCode 校验 service 抛出的错误仍携带 apperr 目录中的业务 code。
// 这是全链路最关键的一环：错误码一旦在用 fmt.Errorf 包装时丢失，
// humax 就会把它当成未知故障降级成 500。
func assertCatalogCode(t *testing.T, err error, want apperr.Spec) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target potatoErrors.IError
	if !errors.As(err, &target) {
		t.Fatalf("err type %T does not carry a catalog code; wrap errors with %%w", err)
	}
	if target.Code() != want.Code() {
		t.Fatalf("code = %d, want %d", target.Code(), want.Code())
	}
}

func TestService_Get_ReturnsNotFoundWhenMissing(t *testing.T) {
	managerMock := manager_mocks.NewMockManager(gomock.NewController(t))
	managerMock.EXPECT().FindByID(gomock.Any(), int64(7)).Return(user.Record{}, false, nil)

	_, err := NewService(managerMock).Get(context.Background(), 7)
	assertCatalogCode(t, err, apperr.NotFound)
}

func TestService_Get_WrapsTechnicalFailure(t *testing.T) {
	managerMock := manager_mocks.NewMockManager(gomock.NewController(t))
	managerMock.EXPECT().FindByID(gomock.Any(), int64(7)).
		Return(user.Record{}, false, errors.New("connection reset"))

	_, err := NewService(managerMock).Get(context.Background(), 7)
	if err == nil {
		t.Fatal("expected error")
	}
	// 技术故障不应伪装成业务错误码，交由 humax 映射为 500。
	var target potatoErrors.IError
	if errors.As(err, &target) {
		t.Fatalf("technical failure must not carry a business code, got %d", target.Code())
	}
}

func TestService_Create_ReturnsConflictOnDuplicateName(t *testing.T) {
	managerMock := manager_mocks.NewMockManager(gomock.NewController(t))
	managerMock.EXPECT().FindByName(gomock.Any(), "alice").Return(user.Record{ID: 1}, true, nil)

	_, err := NewService(managerMock).Create(context.Background(), "alice", "alice@example.com")
	assertCatalogCode(t, err, apperr.Conflict)
}

func TestService_Create_InsertsWhenNameIsAvailable(t *testing.T) {
	managerMock := manager_mocks.NewMockManager(gomock.NewController(t))
	managerMock.EXPECT().FindByName(gomock.Any(), "carol").Return(user.Record{}, false, nil)
	managerMock.EXPECT().Insert(gomock.Any(), user.Record{Name: "carol", Email: "carol@example.com"}).
		Return(user.Record{ID: 3, Name: "carol", Email: "carol@example.com"}, nil)

	item, err := NewService(managerMock).Create(context.Background(), "carol", "carol@example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if item.ID != 3 || item.Name != "carol" {
		t.Fatalf("item = %#v, want created user", item)
	}
}

func TestService_Delete_ReturnsNotFoundWhenMissing(t *testing.T) {
	managerMock := manager_mocks.NewMockManager(gomock.NewController(t))
	managerMock.EXPECT().Delete(gomock.Any(), int64(7)).Return(false, nil)

	assertCatalogCode(t, NewService(managerMock).Delete(context.Background(), 7), apperr.NotFound)
}

func TestService_Export_ReturnsBusinessRuleWhenEmpty(t *testing.T) {
	managerMock := manager_mocks.NewMockManager(gomock.NewController(t))
	managerMock.EXPECT().Search(gomock.Any(), "", int64(0), int64(0)).Return(nil, int64(0), nil)

	_, err := NewService(managerMock).Export(context.Background())
	assertCatalogCode(t, err, apperr.BusinessRule)
}

func TestService_List_ConvertsRecordsToDomainModel(t *testing.T) {
	managerMock := manager_mocks.NewMockManager(gomock.NewController(t))
	managerMock.EXPECT().Search(gomock.Any(), "ali", int64(0), int64(20)).
		Return([]user.Record{{ID: 1, Name: "alice", Email: "alice@example.com"}}, int64(1), nil)

	items, total, err := NewService(managerMock).List(context.Background(), "ali", 0, 20)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Name != "alice" {
		t.Fatalf("items = %#v, total = %d, want one alice", items, total)
	}
}
