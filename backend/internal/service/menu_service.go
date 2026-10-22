package service

import (
	"context"
	"errors"
	"strings"

	"github.com/jihadprakoso/menu-tree/backend/internal/model"
	"github.com/jihadprakoso/menu-tree/backend/internal/repository"
	"gorm.io/gorm"
)

// Error yang dikenali oleh handler untuk menentukan kode HTTP
var (
	ErrNotFound        = errors.New("menu not found")
	ErrValidation      = errors.New("name is required and must be at most 100 characters")
	ErrMoveCycle       = errors.New("cannot move a menu into its own subtree")
	ErrInvalidPosition = errors.New("position is out of range")
	ErrInvalidReorder  = errors.New("menu_ids must include every sibling exactly once")
)

type MenuService struct {
	repo repository.MenuRepository
}

func NewMenuService(repo repository.MenuRepository) *MenuService {
	return &MenuService{repo: repo}
}

// Validasi nama: tidak boleh kosong, maksimal 100 karakter (dihitung per huruf, aman untuk teks Korea)
func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 100 {
		return "", ErrValidation
	}
	return name, nil
}

// Ambil seluruh data lalu susun menjadi tree
func (s *MenuService) GetTree(ctx context.Context) ([]*model.MenuNode, error) {
	menus, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	return buildTree(menus), nil
}

// Satu query untuk semua data, lalu tree disusun di memori (O(n)).
// Ini menghindari masalah N+1 query.
func buildTree(menus []model.Menu) []*model.MenuNode {
	nodes := make(map[string]*model.MenuNode, len(menus))
	for _, m := range menus {
		nodes[m.ID] = &model.MenuNode{Menu: m, Children: []*model.MenuNode{}}
	}

	roots := []*model.MenuNode{}
	for _, m := range menus {
		node := nodes[m.ID]
		if m.ParentID == nil {
			roots = append(roots, node)
		} else if parent, ok := nodes[*m.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}
	return roots
}

func (s *MenuService) GetByID(ctx context.Context, id string) (*model.Menu, error) {
	m, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return m, err
}

func (s *MenuService) Create(ctx context.Context, name string, parentID *string) (*model.Menu, error) {
	name, err := validateName(name)
	if err != nil {
		return nil, err
	}

	// Depth = depth parent + 1. Root = 0.
	depth := 0
	if parentID != nil {
		parent, err := s.GetByID(ctx, *parentID)
		if err != nil {
			return nil, err // ErrNotFound jika parent tidak ada
		}
		depth = parent.Depth + 1
	}

	pos, err := s.repo.NextPosition(ctx, parentID)
	if err != nil {
		return nil, err
	}

	m := &model.Menu{Name: name, ParentID: parentID, Depth: depth, Position: pos}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *MenuService) Update(ctx context.Context, id string, name string) (*model.Menu, error) {
	name, err := validateName(name)
	if err != nil {
		return nil, err
	}

	m, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	m.Name = name
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *MenuService) Delete(ctx context.Context, id string) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *MenuService) Move(ctx context.Context, id string, parentID *string, position int) (*model.Menu, error) {
	menus, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	byID := make(map[string]model.Menu, len(menus))
	var moving *model.Menu
	for i := range menus {
		byID[menus[i].ID] = menus[i]
		if menus[i].ID == id {
			moving = &menus[i]
		}
	}
	if moving == nil {
		return nil, ErrNotFound
	}

	depth := 0
	if parentID != nil {
		parent, ok := byID[*parentID]
		if !ok {
			return nil, ErrNotFound
		}
		depth = parent.Depth + 1

		visited := make(map[string]struct{})
		currentID := parent.ID
		for {
			if currentID == id {
				return nil, ErrMoveCycle
			}
			if _, seen := visited[currentID]; seen {
				return nil, ErrMoveCycle
			}
			visited[currentID] = struct{}{}
			current, exists := byID[currentID]
			if !exists || current.ParentID == nil {
				break
			}
			currentID = *current.ParentID
		}
	}

	destination := siblingMenusForService(menus, parentID, id)
	if position < 0 || position > len(destination) {
		return nil, ErrInvalidPosition
	}

	if err := s.repo.Move(ctx, id, parentID, position, depth); err != nil {
		return nil, err
	}
	moving.ParentID = parentID
	moving.Position = position
	moving.Depth = depth
	return moving, nil
}

func (s *MenuService) Reorder(ctx context.Context, parentID *string, menuIDs []string) error {
	menus, err := s.repo.FindAll(ctx)
	if err != nil {
		return err
	}
	if parentID != nil {
		if _, ok := findMenu(menus, *parentID); !ok {
			return ErrNotFound
		}
	}

	siblings := siblingMenusForService(menus, parentID, "")
	if len(siblings) != len(menuIDs) {
		return ErrInvalidReorder
	}
	expected := make(map[string]struct{}, len(siblings))
	for _, sibling := range siblings {
		expected[sibling.ID] = struct{}{}
	}
	for _, menuID := range menuIDs {
		if _, ok := expected[menuID]; !ok {
			return ErrInvalidReorder
		}
		delete(expected, menuID)
	}
	if len(expected) != 0 {
		return ErrInvalidReorder
	}
	return s.repo.Reorder(ctx, parentID, menuIDs)
}

func siblingMenusForService(menus []model.Menu, parentID *string, excludeID string) []model.Menu {
	siblings := make([]model.Menu, 0)
	for _, menu := range menus {
		if menu.ID != excludeID && sameParentForService(menu.ParentID, parentID) {
			siblings = append(siblings, menu)
		}
	}
	return siblings
}

func findMenu(menus []model.Menu, id string) (model.Menu, bool) {
	for _, menu := range menus {
		if menu.ID == id {
			return menu, true
		}
	}
	return model.Menu{}, false
}

func sameParentForService(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
