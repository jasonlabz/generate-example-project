package dao

import (
	"context"
	"testing"

	"github.com/jasonlabz/sqlite"
	"gorm.io/gorm"

	"github.com/jasonlabz/generate-example-project/internal/dal/db/model"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestUserDaoCRUD(t *testing.T) {
	ctx := context.Background()
	d := NewUserDao(newTestDB(t))

	u := &model.User{Name: "alice", Email: "alice@example.com"}
	if err := d.Insert(ctx, u); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if u.ID == 0 {
		t.Fatal("expect auto increment id")
	}

	got, err := d.SelectByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if got.Name != "alice" {
		t.Fatalf("expect name alice, got %s", got.Name)
	}

	if err := d.UpdateName(ctx, u.ID, "bob"); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err = d.SelectByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("select after update: %v", err)
	}
	if got.Name != "bob" {
		t.Fatalf("expect name bob, got %s", got.Name)
	}

	if err := d.DeleteByID(ctx, u.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err = d.SelectByID(ctx, u.ID); err == nil {
		t.Fatal("expect not found after delete")
	}
}
