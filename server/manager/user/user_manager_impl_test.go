package user

import (
	"context"
	"sync"
	"testing"
)

func TestManager_PreseedsDemoUsers(t *testing.T) {
	manager := NewManager()

	record, exists, err := manager.FindByName(context.Background(), "alice")
	if err != nil {
		t.Fatalf("findByName: %v", err)
	}
	if !exists {
		t.Fatal("alice must be preseeded for local debugging")
	}
	if record.Email != "alice@example.com" {
		t.Fatalf("email = %q, want alice@example.com", record.Email)
	}
}

func TestManager_Insert_AssignsIncreasingIDs(t *testing.T) {
	manager := NewManager()

	first, err := manager.Insert(context.Background(), Record{Name: "carol", Email: "carol@example.com"})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	second, err := manager.Insert(context.Background(), Record{Name: "dave", Email: "dave@example.com"})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if first.ID == 0 || second.ID <= first.ID {
		t.Fatalf("ids = %d, %d, want increasing non-zero ids", first.ID, second.ID)
	}
}

func TestManager_Search_PaginatesAndReportsTotal(t *testing.T) {
	manager := NewManager()

	// 预置 2 条，再插入 1 条，共 3 条；取第二页、每页 2 条应只剩 1 条。
	if _, err := manager.Insert(context.Background(), Record{Name: "carol", Email: "carol@example.com"}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	page, total, err := manager.Search(context.Background(), "", 2, 2)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(page) != 1 {
		t.Fatalf("page length = %d, want 1", len(page))
	}
}

func TestManager_Search_ZeroLimitReturnsEverything(t *testing.T) {
	manager := NewManager()

	page, total, err := manager.Search(context.Background(), "", 0, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if int(total) != len(page) || total == 0 {
		t.Fatalf("total = %d, page length = %d, want all records", total, len(page))
	}
}

func TestManager_Search_FiltersByKeyword(t *testing.T) {
	manager := NewManager()

	page, total, err := manager.Search(context.Background(), "bob", 0, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total != 1 || len(page) != 1 || page[0].Name != "bob" {
		t.Fatalf("page = %#v, total = %d, want only bob", page, total)
	}
}

func TestManager_Delete_ReportsWhetherRecordExisted(t *testing.T) {
	manager := NewManager()

	deleted, err := manager.Delete(context.Background(), 9999)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleted {
		t.Fatal("delete of missing record must report false")
	}

	deleted, err = manager.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !deleted {
		t.Fatal("delete of existing record must report true")
	}
}

// TestManager_ConcurrentAccessIsSafe 在 -race 下验证并发读写安全：
// manager 是共享依赖，任何一次漏锁都会在 make test 里暴露成数据竞争。
func TestManager_ConcurrentAccessIsSafe(t *testing.T) {
	manager := NewManager()

	var waitGroup sync.WaitGroup
	for i := 0; i < 8; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			if _, err := manager.Insert(context.Background(), Record{Name: "concurrent"}); err != nil {
				t.Errorf("insert: %v", err)
			}
			if _, _, err := manager.Search(context.Background(), "", 0, 10); err != nil {
				t.Errorf("search: %v", err)
			}
		}()
	}
	waitGroup.Wait()
}
