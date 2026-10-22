package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jihadprakoso/menu-tree/backend/internal/service"
)

type MenuHandler struct {
	svc *service.MenuService
}

func NewMenuHandler(svc *service.MenuService) *MenuHandler {
	return &MenuHandler{svc: svc}
}

// Daftarkan semua route di bawah /menus
func (h *MenuHandler) Register(r *gin.RouterGroup) {
	g := r.Group("/menus")
	g.GET("", h.list)
	g.GET("/:id", h.get)
	g.POST("", h.create)
	g.PUT("/:id", h.update)
	g.DELETE("/:id", h.delete)
	g.PATCH("/:id/move", h.move)
	g.PATCH("/:id/reorder", h.reorderOne)
	g.PUT("/reorder", h.reorder)
}

// list godoc
// @Summary      Get menu tree
// @Description  Mengembalikan seluruh menu dalam struktur tree
// @Tags         menus
// @Produce      json
// @Success      200  {array}   model.MenuNode
// @Failure      500  {object}  ErrorResponse
// @Router       /menus [get]
func (h *MenuHandler) list(c *gin.Context) {
	tree, err := h.svc.GetTree(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, tree)
}

// Bentuk body request (validasi input lapis pertama)
type CreateMenuRequest struct {
	Name     string  `json:"name" binding:"required,max=100" example:"System Code"`
	ParentID *string `json:"parent_id" binding:"omitempty,uuid" example:"8af03526-7a4b-4c6b-8cfd-90c05d0f0ad1"`
}

type UpdateMenuRequest struct {
	Name string `json:"name" binding:"required,max=100" example:"System Code"`
}

type MoveMenuRequest struct {
	ParentID *string `json:"parent_id" binding:"omitempty,uuid" example:"8af03526-7a4b-4c6b-8cfd-90c05d0f0ad1"`
	Position *int    `json:"position" binding:"required,gte=0" example:"1"`
}

type ReorderMenusRequest struct {
	ParentID *string  `json:"parent_id" binding:"omitempty,uuid" example:"8af03526-7a4b-4c6b-8cfd-90c05d0f0ad1"`
	MenuIDs  []string `json:"menu_ids" binding:"required,dive,uuid" example:"8af03526-7a4b-4c6b-8cfd-90c05d0f0ad1,9465f973-2aa8-473d-b391-9b3027f2a2bb"`
}

type ReorderMenuRequest struct {
	Position *int `json:"position" binding:"required,gte=0" example:"1"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"menu not found"`
}

// get godoc
// @Summary      Get single menu
// @Tags         menus
// @Produce      json
// @Param        id   path      string  true  "Menu ID (UUID)"
// @Success      200  {object}  model.Menu
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /menus/{id} [get]
// GET /api/menus/:id
func (h *MenuHandler) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	m, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}

// POST /api/menus
// create godoc
// @Summary      Create menu
// @Description  Buat menu baru. Kosongkan parent_id untuk menu root
// @Tags         menus
// @Accept       json
// @Produce      json
// @Param        body  body      CreateMenuRequest  true  "Menu baru"
// @Success      201   {object}  model.Menu
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /menus [post]
func (h *MenuHandler) create(c *gin.Context) {
	var req CreateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m, err := h.svc.Create(c.Request.Context(), req.Name, req.ParentID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, m)
}

// update godoc
// @Summary      Update menu name
// @Tags         menus
// @Accept       json
// @Produce      json
// @Param        id    path      string             true  "Menu ID (UUID)"
// @Param        body  body      UpdateMenuRequest  true  "Nama baru"
// @Success      200   {object}  model.Menu
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /menus/{id} [put]
// PUT /api/menus/:id
func (h *MenuHandler) update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req UpdateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m, err := h.svc.Update(c.Request.Context(), id, req.Name)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}

// delete godoc
// @Summary      Delete menu
// @Description  Menghapus menu beserta seluruh anaknya
// @Tags         menus
// @Param        id   path  string  true  "Menu ID (UUID)"
// @Success      204
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /menus/{id} [delete]
// DELETE /api/menus/:id
func (h *MenuHandler) delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// move godoc
// @Summary      Move menu
// @Description  Pindahkan menu ke parent baru. position adalah indeks sibling berbasis 0; parent_id null untuk root.
// @Tags         menus
// @Accept       json
// @Produce      json
// @Param        id    path      string          true  "Menu ID (UUID)"
// @Param        body  body      MoveMenuRequest true  "Parent dan posisi tujuan"
// @Success      200   {object}  model.Menu
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /menus/{id}/move [patch]
func (h *MenuHandler) move(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req MoveMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	menu, err := h.svc.Move(c.Request.Context(), id, req.ParentID, *req.Position)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, menu)
}

// reorder godoc
// @Summary      Reorder sibling menus
// @Description  Atur ulang seluruh sibling dengan urutan ID yang diberikan; semua sibling harus dicantumkan tepat sekali. parent_id null untuk root.
// @Tags         menus
// @Accept       json
// @Produce      json
// @Param        body  body      ReorderMenusRequest true "Parent dan urutan semua sibling"
// @Success      204
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /menus/reorder [put]
func (h *MenuHandler) reorder(c *gin.Context) {
	var req ReorderMenusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.Reorder(c.Request.Context(), req.ParentID, req.MenuIDs); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// reorderOne godoc
// @Summary      Reorder menu within its level
// @Description  Mengubah posisi menu di antara sibling pada parent yang sama.
// @Tags         menus
// @Accept       json
// @Produce      json
// @Param        id    path      string             true  "Menu ID (UUID)"
// @Param        body  body      ReorderMenuRequest true  "Posisi baru berbasis 0"
// @Success      200   {object}  model.Menu
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /menus/{id}/reorder [patch]
func (h *MenuHandler) reorderOne(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req ReorderMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	menu, err := h.svc.GetByID(c.Request.Context(), id)
	if err == nil {
		menu, err = h.svc.Move(c.Request.Context(), id, menu.ParentID, *req.Position)
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, menu)
}

// Pastikan :id di URL berformat UUID yang valid
func parseID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return "", false
	}
	return id, true
}

// Ubah error dari service menjadi kode HTTP
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrMoveCycle),
		errors.Is(err, service.ErrInvalidPosition),
		errors.Is(err, service.ErrInvalidReorder):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
