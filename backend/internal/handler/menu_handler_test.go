package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jihadprakoso/menu-tree/backend/internal/model"
	"github.com/jihadprakoso/menu-tree/backend/internal/service"
	"gorm.io/gorm"
)

const testMenuID = "9465f973-2aa8-473d-b391-9b3027f2a2bb"

func TestMenuHandlerCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		body       string
		repo       *handlerMenuRepository
		wantStatus int
		wantError  string
	}{
		{
			name:       "created",
			body:       `{"name":"New menu"}`,
			repo:       &handlerMenuRepository{},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid json",
			body:       `{"name":`,
			repo:       &handlerMenuRepository{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid parent uuid",
			body:       `{"name":"New menu","parent_id":"not-a-uuid"}`,
			repo:       &handlerMenuRepository{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid name",
			body:       `{"name":"   "}`,
			repo:       &handlerMenuRepository{},
			wantStatus: http.StatusBadRequest,
			wantError:  service.ErrValidation.Error(),
		},
		{
			name:       "missing parent",
			body:       `{"name":"New menu","parent_id":"00000000-0000-4000-8000-000000000001"}`,
			repo:       &handlerMenuRepository{},
			wantStatus: http.StatusNotFound,
			wantError:  service.ErrNotFound.Error(),
		},
		{
			name:       "repository failure",
			body:       `{"name":"New menu"}`,
			repo:       &handlerMenuRepository{nextPositionErr: errors.New("database unavailable")},
			wantStatus: http.StatusInternalServerError,
			wantError:  "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/menus", strings.NewReader(tt.body))
			newTestRouter(tt.repo).ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if tt.wantError != "" {
				var response ErrorResponse
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatalf("decode error response: %v", err)
				}
				if response.Error != tt.wantError {
					t.Errorf("error = %q, want %q", response.Error, tt.wantError)
				}
			}
		})
	}
}

func TestMenuHandlerGetAndDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		method     string
		path       string
		repo       *handlerMenuRepository
		wantStatus int
	}{
		{
			name:       "get menu",
			method:     http.MethodGet,
			path:       "/api/menus/" + testMenuID,
			repo:       &handlerMenuRepository{menus: []model.Menu{{ID: testMenuID, Name: "Menu"}}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid id",
			method:     http.MethodGet,
			path:       "/api/menus/not-a-uuid",
			repo:       &handlerMenuRepository{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing menu",
			method:     http.MethodGet,
			path:       "/api/menus/" + testMenuID,
			repo:       &handlerMenuRepository{},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "delete menu",
			method:     http.MethodDelete,
			path:       "/api/menus/" + testMenuID,
			repo:       &handlerMenuRepository{menus: []model.Menu{{ID: testMenuID, Name: "Menu"}}},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "delete missing menu",
			method:     http.MethodDelete,
			path:       "/api/menus/" + testMenuID,
			repo:       &handlerMenuRepository{},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, nil)
			newTestRouter(tt.repo).ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
		})
	}
}

func TestMenuHandlerListRepositoryError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &handlerMenuRepository{findAllErr: errors.New("database unavailable")}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/menus", nil)

	newTestRouter(repo).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(recorder.Body.String(), "internal server error") {
		t.Errorf("response body = %q, want internal server error", recorder.Body.String())
	}
}

func TestMenuHandlerMoveAndReorder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("reorder one menu within siblings", func(t *testing.T) {
		parentID := "00000000-0000-4000-8000-000000000002"
		secondID := "00000000-0000-4000-8000-000000000003"
		repo := &handlerMenuRepository{
			menus: []model.Menu{
				{ID: parentID, Name: "Parent", Position: 0},
				{ID: testMenuID, Name: "First", ParentID: &parentID, Depth: 1, Position: 0},
				{ID: secondID, Name: "Second", ParentID: &parentID, Depth: 1, Position: 1},
			},
		}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPatch,
			"/api/menus/"+testMenuID+"/reorder",
			strings.NewReader(`{"position":1}`),
		)

		newTestRouter(repo).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if repo.menus[1].Position != 1 {
			t.Errorf("unexpected sibling positions after reorder: %#v", repo.menus)
		}
	})

	t.Run("reject invalid reorder position", func(t *testing.T) {
		repo := &handlerMenuRepository{menus: []model.Menu{{ID: testMenuID, Name: "Menu"}}}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPatch,
			"/api/menus/"+testMenuID+"/reorder",
			strings.NewReader(`{"position":-1}`),
		)

		newTestRouter(repo).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
	})

	t.Run("reject missing menu when reordering", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPatch,
			"/api/menus/"+testMenuID+"/reorder",
			strings.NewReader(`{"position":0}`),
		)

		newTestRouter(&handlerMenuRepository{}).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusNotFound, recorder.Body.String())
		}
	})

	t.Run("move menu", func(t *testing.T) {
		parentID := "00000000-0000-4000-8000-000000000002"
		repo := &handlerMenuRepository{
			menus: []model.Menu{
				{ID: testMenuID, Name: "Menu"},
				{ID: parentID, Name: "Parent"},
			},
		}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPatch,
			"/api/menus/"+testMenuID+"/move",
			strings.NewReader(`{"parent_id":"`+parentID+`","position":0}`),
		)

		newTestRouter(repo).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if repo.menus[0].ParentID == nil || *repo.menus[0].ParentID != parentID {
			t.Errorf("menu parent_id = %v, want %s", repo.menus[0].ParentID, parentID)
		}
	})

	t.Run("reject move cycle", func(t *testing.T) {
		parentID := testMenuID
		repo := &handlerMenuRepository{
			menus: []model.Menu{
				{ID: testMenuID, Name: "Parent"},
				{ID: "00000000-0000-4000-8000-000000000002", Name: "Child", ParentID: &parentID, Depth: 1},
			},
		}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPatch,
			"/api/menus/"+testMenuID+"/move",
			strings.NewReader(`{"parent_id":"00000000-0000-4000-8000-000000000002","position":0}`),
		)

		newTestRouter(repo).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
	})

	t.Run("reorder siblings", func(t *testing.T) {
		secondID := "00000000-0000-4000-8000-000000000002"
		repo := &handlerMenuRepository{
			menus: []model.Menu{
				{ID: testMenuID, Position: 0},
				{ID: secondID, Position: 1},
			},
		}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPut,
			"/api/menus/reorder",
			strings.NewReader(`{"parent_id":null,"menu_ids":["`+secondID+`","`+testMenuID+`"]}`),
		)

		newTestRouter(repo).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusNoContent, recorder.Body.String())
		}
		if repo.menus[0].Position != 1 || repo.menus[1].Position != 0 {
			t.Errorf("unexpected reordered positions: %#v", repo.menus)
		}
	})

	t.Run("reject incomplete reorder", func(t *testing.T) {
		repo := &handlerMenuRepository{
			menus: []model.Menu{
				{ID: testMenuID, Position: 0},
				{ID: "00000000-0000-4000-8000-000000000002", Position: 1},
			},
		}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPut,
			"/api/menus/reorder",
			strings.NewReader(`{"menu_ids":["`+testMenuID+`"]}`),
		)

		newTestRouter(repo).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
	})
}

func newTestRouter(repo *handlerMenuRepository) *gin.Engine {
	router := gin.New()
	NewMenuHandler(service.NewMenuService(repo)).Register(router.Group("/api"))
	return router
}

type handlerMenuRepository struct {
	menus           []model.Menu
	findAllErr      error
	nextPositionErr error
}

func (r *handlerMenuRepository) FindAll(context.Context) ([]model.Menu, error) {
	return r.menus, r.findAllErr
}

func (r *handlerMenuRepository) FindByID(_ context.Context, id string) (*model.Menu, error) {
	for i := range r.menus {
		if r.menus[i].ID == id {
			return &r.menus[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *handlerMenuRepository) Create(_ context.Context, menu *model.Menu) error {
	menu.ID = testMenuID
	r.menus = append(r.menus, *menu)
	return nil
}

func (r *handlerMenuRepository) Update(context.Context, *model.Menu) error {
	return nil
}

func (r *handlerMenuRepository) Delete(context.Context, string) error {
	return nil
}

func (r *handlerMenuRepository) NextPosition(context.Context, *string) (int, error) {
	return 0, r.nextPositionErr
}

func (r *handlerMenuRepository) Move(_ context.Context, id string, parentID *string, position, depth int) error {
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

func (r *handlerMenuRepository) Reorder(_ context.Context, parentID *string, menuIDs []string) error {
	for position, id := range menuIDs {
		for i := range r.menus {
			if r.menus[i].ID == id && sameHandlerParent(r.menus[i].ParentID, parentID) {
				r.menus[i].Position = position
				break
			}
		}
	}
	return nil
}

func sameHandlerParent(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
