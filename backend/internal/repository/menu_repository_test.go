package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jihadprakoso/menu-tree/backend/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openTestRepository(t *testing.T) (*gorm.DB, MenuRepository) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a dedicated PostgreSQL test database")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open PostgreSQL test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get PostgreSQL connection: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close PostgreSQL test database: %v", err)
		}
	})

	if err := db.Exec(`
		CREATE TEMP TABLE menus (
			id UUID PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			parent_id UUID REFERENCES menus(id) ON DELETE CASCADE,
			depth INT NOT NULL DEFAULT 0,
			position INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		) ON COMMIT PRESERVE ROWS
	`).Error; err != nil {
		t.Fatalf("create temporary menus table: %v", err)
	}
	return db, NewMenuRepository(db)
}

func insertTestMenu(t *testing.T, db *gorm.DB, id, name string, parentID *string, depth, position int) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO menus (id, name, parent_id, depth, position) VALUES (?, ?, ?, ?, ?)",
		id, name, parentID, depth, position,
	).Error; err != nil {
		t.Fatalf("insert test menu: %v", err)
	}
}

func TestMenuRepositoryCRUDAndNextPosition(t *testing.T) {
	db, repo := openTestRepository(t)
	ctx := context.Background()
	rootID := "9465f973-2aa8-473d-b391-9b3027f2a2bb"
	childID := "2917550a-6342-405a-b4eb-cf3870d6ae7d"
	insertTestMenu(t, db, rootID, "Root", nil, 0, 0)

	next, err := repo.NextPosition(ctx, nil)
	if err != nil || next != 1 {
		t.Fatalf("NextPosition(root) = %d, %v; want 1, nil", next, err)
	}
	if err := repo.Create(ctx, &model.Menu{ID: childID, Name: "Child", ParentID: &rootID, Depth: 1, Position: 0}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	menus, err := repo.FindAll(ctx)
	if err != nil || len(menus) != 2 {
		t.Fatalf("FindAll() returned %d menus, error = %v; want 2, nil", len(menus), err)
	}
	child, err := repo.FindByID(ctx, childID)
	if err != nil || child.Name != "Child" {
		t.Fatalf("FindByID() = %#v, %v; want child menu", child, err)
	}

	child.Name = "Updated child"
	if err := repo.Update(ctx, child); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := repo.Delete(ctx, childID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.FindByID(ctx, childID); err == nil {
		t.Fatal("FindByID() after Delete() unexpectedly found menu")
	}
}

func TestMenuRepositoryMoveAndReorder(t *testing.T) {
	db, repo := openTestRepository(t)
	ctx := context.Background()
	rootA := "9465f973-2aa8-473d-b391-9b3027f2a2bb"
	branch := "2917550a-6342-405a-b4eb-cf3870d6ae7d"
	moving := "81057828-f358-42fd-8bb8-95ebc98cd79d"
	descendant := "7600fb13-d953-40ba-ab3a-ed2ab7695661"
	rootB := "ae5185a0-bf43-4990-9fe3-1a5ecdeced01"
	destinationChild := "00000000-0000-4000-8000-000000000001"

	insertTestMenu(t, db, rootA, "Root A", nil, 0, 0)
	insertTestMenu(t, db, branch, "Branch", &rootA, 1, 0)
	insertTestMenu(t, db, moving, "Moving", &branch, 2, 0)
	insertTestMenu(t, db, descendant, "Descendant", &moving, 3, 0)
	insertTestMenu(t, db, rootB, "Root B", nil, 0, 1)
	insertTestMenu(t, db, destinationChild, "Destination child", &rootB, 1, 0)

	if err := repo.Move(ctx, moving, &rootB, 0, 1); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	moved, err := repo.FindByID(ctx, moving)
	if err != nil {
		t.Fatalf("FindByID(moving) error = %v", err)
	}
	if moved.ParentID == nil || *moved.ParentID != rootB || moved.Position != 0 || moved.Depth != 1 {
		t.Errorf("moved menu = %#v; want parent root B, position 0, depth 1", moved)
	}
	child, err := repo.FindByID(ctx, destinationChild)
	if err != nil || child.Position != 1 {
		t.Errorf("destination sibling = %#v, %v; want position 1", child, err)
	}
	grandchild, err := repo.FindByID(ctx, descendant)
	if err != nil || grandchild.Depth != 2 {
		t.Errorf("moved descendant = %#v, %v; want depth 2", grandchild, err)
	}
	oldSibling, err := repo.FindByID(ctx, branch)
	if err != nil || oldSibling.Position != 0 {
		t.Errorf("source sibling = %#v, %v; want compacted position 0", oldSibling, err)
	}

	if err := repo.Reorder(ctx, &rootB, []string{destinationChild, moving}); err != nil {
		t.Fatalf("Reorder() error = %v", err)
	}
	moved, err = repo.FindByID(ctx, moving)
	if err != nil || moved.Position != 1 {
		t.Errorf("reordered menu = %#v, %v; want position 1", moved, err)
	}
	child, err = repo.FindByID(ctx, destinationChild)
	if err != nil || child.Position != 0 {
		t.Errorf("reordered sibling = %#v, %v; want position 0", child, err)
	}
}
