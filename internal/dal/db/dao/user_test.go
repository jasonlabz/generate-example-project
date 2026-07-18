package dao

import (
	"context"
	"errors"
	"testing"

	"github.com/jasonlabz/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/jasonlabz/generate-example-project/internal/dal/db/model"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
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
	if _, err = d.SelectByID(ctx, u.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expect gorm.ErrRecordNotFound after delete, got %v", err)
	}
}
