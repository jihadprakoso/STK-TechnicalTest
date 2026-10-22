package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jihadprakoso/menu-tree/backend/internal/model"
	"gorm.io/gorm"
)

func TestBuildTree(t *testing.T) {
	parentID := "root"
	childID := "child"
	menus := []model.Menu{
		{ID: "root", Name: "Root", Position: 0},
		{ID: "child", Name: "Child", ParentID: &parentID, Depth: 1, Position: 0},
		{ID: "grandchild", Name: "Grandchild", ParentID: &childID, Depth: 2, Position: 0},
		{ID: "another-root", Name: "Another root", Position: 1},
	}

	tree := buildTree(menus)

	if len(tree) != 2 {
		t.Fatalf("expected 2 root menus, got %d", len(tree))
	}
	if tree[0].ID != "root" || tree[1].ID != "another-root" {
		t.Fatalf("unexpected root order: got %q and %q", tree[0].ID, tree[1].ID)
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].ID != "child" {
		t.Fatalf("expected root to have child, got %#v", tree[0].Children)
	}
	if len(tree[0].Children[0].Children) != 1 || tree[0].Children[0].Children[0].ID != "grandchild" {
		t.Fatalf("expected child to have grandchild, got %#v", tree[0].Children[0].Children)
	}
	if tree[1].Children == nil || len(tree[1].Children) != 0 {
		t.Fatalf("expected another root to have an empty children slice, got %#v", tree[1].Children)
	}
}

func TestBuildTreeEmpty(t *testing.T) {
	tree := buildTree(nil)
	if len(tree) != 0 {
		t.Fatalf("expected no roots for empty input, got %d", len(tree))
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "trims surrounding spaces", input: "  Menu  ", want: "Menu"},
		{name: "rejects empty", input: "", wantErr: ErrValidation},
		{name: "rejects whitespace", input: "   ", wantErr: ErrValidation},
		{name: "accepts 100 characters", input: strings.Repeat("a", 100), want: strings.Repeat("a", 100)},
		{name: "rejects more than 100 characters", input: strings.Repeat("a", 101), wantErr: ErrValidation},
		{name: "counts unicode characters", input: strings.Repeat("한", 100), want: strings.Repeat("한", 100)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateName(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validateName() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("validateName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateMenu(t *testing.T) {
	repo := &fakeMenuRepository{
		menus: []model.Menu{
			{ID: "existing-root", Name: "Existing root", Position: 0},
			{ID: "parent", Name: "Parent", Depth: 1, Position: 0},
		},
	}
	svc := NewMenuService(repo)
	ctx := context.Background()

	root, err := svc.Create(ctx, "  New root  ", nil)
	if err != nil {
		t.Fatalf("Create() root error = %v", err)
	}
	if root.Name != "New root" || root.Depth != 0 || root.Position != 1 || root.ParentID != nil {
		t.Errorf("Create() root = %#v", root)
	}

	parentID := "parent"
	child, err := svc.Create(ctx, "Child", &parentID)
	if err != nil {
		t.Fatalf("Create() child error = %v", err)
	}
	if child.Depth != 2 || child.Position != 0 || child.ParentID == nil || *child.ParentID != parentID {
		t.Errorf("Create() child = %#v", child)
	}
}

func TestCreateMenuRejectsMissingParent(t *testing.T) {
	svc := NewMenuService(&fakeMenuRepository{})

	_, err := svc.Create(context.Background(), "Child", stringPointer("missing"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Create() error = %v, want %v", err, ErrNotFound)
	}
}

func TestUpdateMenu(t *testing.T) {
	repo := &fakeMenuRepository{
		menus: []model.Menu{{ID: "menu-1", Name: "Old name", Depth: 2}},
	}
	svc := NewMenuService(repo)

	updated, err := svc.Update(context.Background(), "menu-1", "  New name  ")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "New name" || updated.Depth != 2 {
		t.Errorf("Update() menu = %#v", updated)
	}
	if repo.updateCalls != 1 || repo.menus[0].Name != "New name" {
		t.Errorf("expected repository to save updated menu, calls=%d menu=%#v", repo.updateCalls, repo.menus[0])
	}
}

func TestUpdateMenuRejectsInvalidName(t *testing.T) {
	repo := &fakeMenuRepository{
		menus: []model.Menu{{ID: "menu-1", Name: "Existing name"}},
	}
	svc := NewMenuService(repo)

	_, err := svc.Update(context.Background(), "menu-1", " ")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want %v", err, ErrValidation)
	}
	if repo.updateCalls != 0 {
		t.Errorf("expected repository not to be updated, got %d calls", repo.updateCalls)
	}
}

func TestUpdateMenuNotFound(t *testing.T) {
	repo := &fakeMenuRepository{}
	svc := NewMenuService(repo)

	_, err := svc.Update(context.Background(), "missing", "Name")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, ErrNotFound)
	}
	if repo.updateCalls != 0 {
		t.Errorf("expected repository not to be updated, got %d calls", repo.updateCalls)
	}
}

func TestDeleteMenu(t *testing.T) {
	repo := &fakeMenuRepository{
		menus: []model.Menu{{ID: "menu-1", Name: "Menu"}},
	}
	svc := NewMenuService(repo)

	if err := svc.Delete(context.Background(), "menu-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if repo.deleteCalls != 1 || repo.deletedID != "menu-1" {
		t.Errorf("expected repository to delete menu-1, calls=%d id=%q", repo.deleteCalls, repo.deletedID)
	}
}

func TestDeleteMenuNotFound(t *testing.T) {
	repo := &fakeMenuRepository{}
	svc := NewMenuService(repo)

	err := svc.Delete(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete() error = %v, want %v", err, ErrNotFound)
	}
	if repo.deleteCalls != 0 {
		t.Errorf("expected repository not to be called, got %d calls", repo.deleteCalls)
	}
}

func TestMoveMenuUpdatesParentDepthAndPosition(t *testing.T) {
	parentID := "parent"
	repo := &fakeMenuRepository{
		menus: []model.Menu{
			{ID: "root-a", Depth: 0},
			{ID: "parent", ParentID: stringPointer("root-a"), Depth: 1},
			{ID: "moving", ParentID: &parentID, Depth: 2, Position: 1},
			{ID: "descendant", ParentID: stringPointer("moving"), Depth: 3},
			{ID: "root-b", Depth: 0},
		},
	}
	svc := NewMenuService(repo)

	moved, err := svc.Move(context.Background(), "moving", stringPointer("root-b"), 0)
	if err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	if moved.ParentID == nil || *moved.ParentID != "root-b" || moved.Depth != 1 || moved.Position != 0 {
		t.Errorf("Move() menu = %#v", moved)
	}
}

func TestMoveMenuRejectsCycleAndInvalidPosition(t *testing.T) {
	parentID := "root"
	repo := &fakeMenuRepository{
		menus: []model.Menu{
			{ID: "root", Depth: 0},
			{ID: "child", ParentID: &parentID, Depth: 1},
			{ID: "grandchild", ParentID: stringPointer("child"), Depth: 2},
		},
	}
	svc := NewMenuService(repo)

	if _, err := svc.Move(context.Background(), "root", stringPointer("grandchild"), 0); !errors.Is(err, ErrMoveCycle) {
		t.Errorf("Move() cycle error = %v, want %v", err, ErrMoveCycle)
	}
	if _, err := svc.Move(context.Background(), "child", nil, 2); !errors.Is(err, ErrInvalidPosition) {
		t.Errorf("Move() position error = %v, want %v", err, ErrInvalidPosition)
	}
}

func TestReorderMenus(t *testing.T) {
	repo := &fakeMenuRepository{
		menus: []model.Menu{
			{ID: "first", Position: 0},
			{ID: "second", Position: 1},
		},
	}
	svc := NewMenuService(repo)

	if err := svc.Reorder(context.Background(), nil, []string{"second", "first"}); err != nil {
		t.Fatalf("Reorder() error = %v", err)
	}
	if repo.menus[0].Position != 1 || repo.menus[1].Position != 0 {
		t.Errorf("unexpected reordered positions: %#v", repo.menus)
	}
}

func TestReorderMenusRequiresEverySiblingOnce(t *testing.T) {
	repo := &fakeMenuRepository{
		menus: []model.Menu{
			{ID: "first", Position: 0},
			{ID: "second", Position: 1},
		},
	}
	svc := NewMenuService(repo)

	for _, ids := range [][]string{{"first"}, {"first", "first"}, {"first", "missing"}} {
		if err := svc.Reorder(context.Background(), nil, ids); !errors.Is(err, ErrInvalidReorder) {
			t.Errorf("Reorder(%v) error = %v, want %v", ids, err, ErrInvalidReorder)
		}
	}
}

func stringPointer(value string) *string {
	return &value
}

type fakeMenuRepository struct {
	menus       []model.Menu
	updateCalls int
	deleteCalls int
	deletedID   string
}

func (r *fakeMenuRepository) FindAll(context.Context) ([]model.Menu, error) {
	return r.menus, nil
}

func (r *fakeMenuRepository) FindByID(_ context.Context, id string) (*model.Menu, error) {
	for i := range r.menus {
		if r.menus[i].ID == id {
			return &r.menus[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeMenuRepository) Create(_ context.Context, menu *model.Menu) error {
	menu.ID = "created-menu"
	r.menus = append(r.menus, *menu)
	return nil
}

func (r *fakeMenuRepository) Update(_ context.Context, menu *model.Menu) error {
	r.updateCalls++
	for i := range r.menus {
		if r.menus[i].ID == menu.ID {
			r.menus[i] = *menu
			return nil
		}
	}
	return nil
}

func (r *fakeMenuRepository) Delete(_ context.Context, id string) error {
	r.deleteCalls++
	r.deletedID = id
	return nil
}

func (r *fakeMenuRepository) NextPosition(_ context.Context, parentID *string) (int, error) {
	next := 0
	for _, menu := range r.menus {
		if sameParent(menu.ParentID, parentID) && menu.Position >= next {
			next = menu.Position + 1
		}
	}
	return next, nil
}

func (r *fakeMenuRepository) Move(_ context.Context, id string, parentID *string, position, depth int) error {
	for i := range r.menus {
		if r.menus[i].ID == id {
			r.menus[i].ParentID = parentID
			r.menus[i].Position = position
			r.menus[i].Depth = depth
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (r *fakeMenuRepository) Reorder(_ context.Context, parentID *string, menuIDs []string) error {
	for position, id := range menuIDs {
		for i := range r.menus {
			if r.menus[i].ID == id && sameParent(r.menus[i].ParentID, parentID) {
				r.menus[i].Position = position
				break
			}
		}
	}
	return nil
}

func sameParent(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
