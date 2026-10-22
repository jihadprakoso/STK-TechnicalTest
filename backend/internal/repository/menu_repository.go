package repository

import (
	"context"

	"github.com/jihadprakoso/menu-tree/backend/internal/model"
	"gorm.io/gorm"
)

// MenuRepository = daftar fungsi yang tersedia untuk akses data menu
type MenuRepository interface {
	FindAll(ctx context.Context) ([]model.Menu, error)
	FindByID(ctx context.Context, id string) (*model.Menu, error)
	Create(ctx context.Context, m *model.Menu) error
	Update(ctx context.Context, m *model.Menu) error
	Delete(ctx context.Context, id string) error
	NextPosition(ctx context.Context, parentID *string) (int, error)
	Move(ctx context.Context, id string, parentID *string, position, depth int) error
	Reorder(ctx context.Context, parentID *string, menuIDs []string) error
}

type menuRepository struct {
	db *gorm.DB
}

func NewMenuRepository(db *gorm.DB) MenuRepository {
	return &menuRepository{db: db}
}

// Ambil semua menu, urut berdasarkan position
func (r *menuRepository) FindAll(ctx context.Context) ([]model.Menu, error) {
	var menus []model.Menu
	err := r.db.WithContext(ctx).Order("position ASC, created_at ASC").Find(&menus).Error
	return menus, err
}

// Ambil satu menu. Jika tidak ada, mengembalikan gorm.ErrRecordNotFound
func (r *menuRepository) FindByID(ctx context.Context, id string) (*model.Menu, error) {
	var m model.Menu
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *menuRepository) Create(ctx context.Context, m *model.Menu) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *menuRepository) Update(ctx context.Context, m *model.Menu) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// Anak-anak ikut terhapus otomatis oleh ON DELETE CASCADE di database
func (r *menuRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Menu{}, "id = ?", id).Error
}

// Hitung posisi berikutnya di antara saudara (menu dengan parent yang sama)
func (r *menuRepository) NextPosition(ctx context.Context, parentID *string) (int, error) {
	q := r.db.WithContext(ctx).Model(&model.Menu{})
	if parentID == nil {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", *parentID)
	}

	var next int
	err := q.Select("COALESCE(MAX(position) + 1, 0)").Scan(&next).Error
	return next, err
}

func (r *menuRepository) Move(ctx context.Context, id string, parentID *string, position, depth int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var menus []model.Menu
		if err := tx.Order("position ASC, id ASC").Find(&menus).Error; err != nil {
			return err
		}

		var moving *model.Menu
		for i := range menus {
			if menus[i].ID == id {
				moving = &menus[i]
				break
			}
		}
		if moving == nil {
			return gorm.ErrRecordNotFound
		}

		source := siblingMenus(menus, moving.ParentID, id)
		destination := siblingMenus(menus, parentID, id)
		if position < 0 || position > len(destination) {
			return gorm.ErrInvalidData
		}
		destination = insertMenu(destination, *moving, position)

		delta := depth - moving.Depth
		descendantIDs := descendantsOf(menus, id)
		if err := tx.Model(&model.Menu{}).Where("id = ?", id).Updates(map[string]any{
			"parent_id": parentID,
			"position":  position,
			"depth":     depth,
		}).Error; err != nil {
			return err
		}
		for _, descendantID := range descendantIDs {
			if err := tx.Model(&model.Menu{}).
				Where("id = ?", descendantID).
				Update("depth", gorm.Expr("depth + ?", delta)).Error; err != nil {
				return err
			}
		}

		if sameParent(moving.ParentID, parentID) {
			return updateSiblingPositions(tx, destination)
		}
		if err := updateSiblingPositions(tx, source); err != nil {
			return err
		}
		return updateSiblingPositions(tx, destination)
	})
}

func (r *menuRepository) Reorder(ctx context.Context, parentID *string, menuIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var siblings []model.Menu
		query := tx.Model(&model.Menu{})
		if parentID == nil {
			query = query.Where("parent_id IS NULL")
		} else {
			query = query.Where("parent_id = ?", *parentID)
		}
		if err := query.Order("position ASC, id ASC").Find(&siblings).Error; err != nil {
			return err
		}
		if len(siblings) != len(menuIDs) {
			return gorm.ErrInvalidData
		}

		byID := make(map[string]model.Menu, len(siblings))
		for _, sibling := range siblings {
			byID[sibling.ID] = sibling
		}
		ordered := make([]model.Menu, 0, len(menuIDs))
		seen := make(map[string]struct{}, len(menuIDs))
		for _, menuID := range menuIDs {
			menu, ok := byID[menuID]
			if !ok {
				return gorm.ErrInvalidData
			}
			if _, duplicate := seen[menuID]; duplicate {
				return gorm.ErrInvalidData
			}
			seen[menuID] = struct{}{}
			ordered = append(ordered, menu)
		}
		return updateSiblingPositions(tx, ordered)
	})
}

func siblingMenus(menus []model.Menu, parentID *string, excludeID string) []model.Menu {
	siblings := make([]model.Menu, 0)
	for _, menu := range menus {
		if menu.ID != excludeID && sameParent(menu.ParentID, parentID) {
			siblings = append(siblings, menu)
		}
	}
	return siblings
}

func insertMenu(menus []model.Menu, menu model.Menu, position int) []model.Menu {
	menus = append(menus, model.Menu{})
	copy(menus[position+1:], menus[position:])
	menus[position] = menu
	return menus
}

func descendantsOf(menus []model.Menu, id string) []string {
	children := make(map[string][]string)
	for _, menu := range menus {
		if menu.ParentID != nil {
			children[*menu.ParentID] = append(children[*menu.ParentID], menu.ID)
		}
	}

	var descendants []string
	queue := append([]string(nil), children[id]...)
	visited := map[string]struct{}{id: {}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if _, seen := visited[current]; seen {
			continue
		}
		visited[current] = struct{}{}
		descendants = append(descendants, current)
		queue = append(queue, children[current]...)
	}
	return descendants
}

func updateSiblingPositions(tx *gorm.DB, siblings []model.Menu) error {
	for position, sibling := range siblings {
		if err := tx.Model(&model.Menu{}).
			Where("id = ?", sibling.ID).
			Update("position", position).Error; err != nil {
			return err
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
