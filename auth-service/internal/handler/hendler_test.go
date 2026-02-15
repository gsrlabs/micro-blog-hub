package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/config"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/model"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// mockAuthService реализует service.AuthService интерфейс
type mockAuthService struct {
	mock.Mock
}

func (m *mockAuthService) Register(ctx context.Context, req *model.CreateUserRequest) (uuid.UUID, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockAuthService) Login(ctx context.Context, req *model.LoginRequest) (string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Error(1)
}

func (m *mockAuthService) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *mockAuthService) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *mockAuthService) ChangeProfile(ctx context.Context, id uuid.UUID, req *model.ChangeProfileRequest) error {
	args := m.Called(ctx, id, req)
	return args.Error(0)
}

func (m *mockAuthService) ChangeEmail(ctx context.Context, id uuid.UUID, req *model.ChangeEmailRequest) error {
	args := m.Called(ctx, id, req)
	return args.Error(0)
}

func (m *mockAuthService) ChangePassword(ctx context.Context, id uuid.UUID, req *model.ChangePasswordRequest) error {
	args := m.Called(ctx, id, req)
	return args.Error(0)
}

func (m *mockAuthService) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockAuthService) GetUsers(ctx context.Context, limit, offset int) ([]*model.User, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*model.User), args.Error(1)
}

// ----------------- HELPERS -----------------
func performRequest(h http.Handler, method, path string, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}



func TestAuthHandler_SignUp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	h := NewAuthHandler(mockSvc, logger, &config.Config{})

	r := gin.New()
	r.POST("/signup", h.SignUp)

	userID := uuid.New()
	mockSvc.On("Register", mock.Anything, mock.Anything).Return(userID, nil)

	body := `{"username":"test","email":"test@test.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "user registered")
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_SignIn(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "secret", ExpirationHours: 1}}
	h := NewAuthHandler(mockSvc, logger, cfg)

	r := gin.New()
	r.POST("/signin", h.SignIn)

	mockSvc.On("Login", mock.Anything, mock.Anything).Return("token123", nil)

	body := `{"email":"test@test.com","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "token")
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_GetProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	h := NewAuthHandler(mockSvc, logger, &config.Config{})

	r := gin.New()
	r.GET("/profile", h.GetProfile)

	id := uuid.New()
	user := &model.User{ID: id, Username: "user1", Email: "email@test.com"}

	mockSvc.On("GetByID", mock.Anything, id).Return(user, nil)

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userID", id)

	h.GetProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user1")
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_ChangeProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	h := NewAuthHandler(mockSvc, logger, &config.Config{})

	id := uuid.New()

	r := gin.New()
	r.PUT("/user/profile", func(c *gin.Context) {
		c.Set("userID", id) // эмулируем middleware
		h.ChangeProfile(c)
	})

	mockSvc.On("ChangeProfile", mock.Anything, id, mock.Anything).Return(nil)

	// Ключ json совпадает с полем структуры
	body := `{"new_username":"newname"}`
	req := httptest.NewRequest(http.MethodPut, "/user/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "profile updated successfully")
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_GetUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	h := NewAuthHandler(mockSvc, logger, &config.Config{})

	users := []*model.User{
		{ID: uuid.New(), Username: "u1", Email: "e1@test.com"},
		{ID: uuid.New(), Username: "u2", Email: "e2@test.com"},
	}
	mockSvc.On("GetUsers", mock.Anything, 10, 0).Return(users, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/users", nil)

	h.GetUsers(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "u1")
	assert.Contains(t, w.Body.String(), "u2")
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_ChangeEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	h := NewAuthHandler(mockSvc, logger, &config.Config{}) // ✅ через конструктор

	id := uuid.New()
	mockSvc.On("ChangeEmail", mock.Anything, id, mock.Anything).Return(nil)

	// Ключ должен совпадать с тегом `json:"new_email"` в ChangeEmailRequest
	body := `{"new_email":"new@test.com"}`
	req := httptest.NewRequest(http.MethodPut, "/user/email", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userID", id)

	h.ChangeEmail(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "email updated successfully")
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_ChangePassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	h := NewAuthHandler(mockSvc, logger, &config.Config{})

	id := uuid.New()
	mockSvc.On("ChangePassword", mock.Anything, id, mock.Anything).Return(nil)

	body := `{"old_password":"oldpassword","new_password":"newpassword"}`
	req := httptest.NewRequest(http.MethodPut, "/user/password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userID", id)

	h.ChangePassword(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "password updated successfully")
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	h := NewAuthHandler(mockSvc, logger, &config.Config{})

	id := uuid.New()
	mockSvc.On("Delete", mock.Anything, id).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/user", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userID", id)

	h.Delete(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user has been deleted successfully")
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_GetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	h := NewAuthHandler(mockSvc, logger, &config.Config{})

	id := uuid.New()
	user := &model.User{ID: id, Username: "user1", Email: "email@test.com"}
	mockSvc.On("GetByID", mock.Anything, id).Return(user, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.GetByID(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user1")
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_GetByEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	h := NewAuthHandler(mockSvc, logger, &config.Config{})

	email := "email@test.com"
	user := &model.User{ID: uuid.New(), Username: "user1", Email: email}
	mockSvc.On("GetByEmail", mock.Anything, email).Return(user, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/search?email="+email, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	h.GetByEmail(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user1")
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_SignUp_Errors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	h := NewAuthHandler(mockSvc, logger, &config.Config{})

	r := gin.New()
	r.POST("/signup", h.SignUp)

	// invalid JSON
	w := performRequest(r, "POST", "/signup", `invalid-json`, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// validation error (missing password)
	w = performRequest(r, "POST", "/signup", `{"username":"u","email":"test@test.com"}`, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// service error
	mockSvc.On("Register", mock.Anything, mock.Anything).Return(uuid.Nil, errors.New("db error"))
	body := `{"username":"testuser","email":"test@test.com","password":"password123"}`
	w = performRequest(r, "POST", "/signup", body, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_SignIn_Errors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	h := NewAuthHandler(mockSvc, logger, &config.Config{App: config.AppConfig{Mode: "release"}})

	r := gin.New()
	r.POST("/signin", h.SignIn)

	// invalid JSON
	w := performRequest(r, "POST", "/signin", `invalid-json`, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// validation error
	w = performRequest(r, "POST", "/signin", `{"email":""}`, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// service error
	mockSvc.On("Login", mock.Anything, mock.Anything).Return("", errors.New("invalid credentials"))
	body := `{"email":"test@test.com","password":"password123"}`
	w = performRequest(r, "POST", "/signin", body, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_GetByID_Errors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	h := NewAuthHandler(mockSvc, logger, &config.Config{})

	r := gin.New()
	r.GET("/users/:id", h.GetByID)

	// invalid uuid
	w := performRequest(r, "GET", "/users/invalid-uuid", "", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// not found
	id := uuid.New()
	mockSvc.On("GetByID", mock.Anything, id).Return(&model.User{}, errors.New("not found"))
	w = performRequest(r, "GET", "/users/"+id.String(), "", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_AuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"
	h := &AuthHandler{cfg: &config.Config{JWT: config.JWTConfig{Secret: secret}}}
	r := gin.New()
	r.GET("/protected", h.AuthMiddleware, func(c *gin.Context) { c.Status(http.StatusOK) })

	id := uuid.New()
	username := "tester"

	// no token
	w := performRequest(r, "GET", "/protected", "", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// invalid token
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// valid token
	token := generateTestToken(id, username, secret, false)
	req = httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_ChangeProfile_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	h := NewAuthHandler(mockSvc, logger, &config.Config{})

	r := gin.New()
	id := uuid.New()
	r.PUT("/user/profile", func(c *gin.Context) {
		c.Set("userID", id)
		h.ChangeProfile(c)
	})

	// invalid JSON
	w := performRequest(r, "PUT", "/user/profile", `{"wrong_field":"x"}`, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// valid
	mockSvc.On("ChangeProfile", mock.Anything, id, mock.Anything).Return(nil)
	w = performRequest(r, "PUT", "/user/profile", `{"new_username":"newname"}`, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_ChangeProfile_Errors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := &mockAuthService{}
	logger := zap.NewNop()
	h := NewAuthHandler(mockSvc, logger, &config.Config{})
	id := uuid.New()

	t.Run("Unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodPut, "/user/profile", nil)
		c.Request = req

		h.ChangeProfile(c)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Validation Failed", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("userID", id)
		req := httptest.NewRequest(http.MethodPut, "/user/profile", strings.NewReader(`{"username":""}`))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req

		h.ChangeProfile(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "validation failed")
	})

	t.Run("Duplicate Username", func(t *testing.T) {
	mockSvc := &mockAuthService{} // новый мок
	h := NewAuthHandler(mockSvc, logger, &config.Config{})
	mockSvc.On("ChangeProfile", mock.Anything, id, mock.Anything).Return(repository.ErrDuplicateUsername)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", id)
	body := `{"new_username":"taken"}`
	req := httptest.NewRequest(http.MethodPut, "/user/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChangeProfile(c)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "username already taken")
	mockSvc.AssertExpectations(t)
})

t.Run("User Not Found", func(t *testing.T) {
	mockSvc := &mockAuthService{} // снова новый мок
	h := NewAuthHandler(mockSvc, logger, &config.Config{})
	mockSvc.On("ChangeProfile", mock.Anything, id, mock.Anything).Return(repository.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", id)
	body := `{"new_username":"okname"}`
	req := httptest.NewRequest(http.MethodPut, "/user/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	h.ChangeProfile(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
})
}

func TestAuthHandler_ChangeEmail_Errors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := &mockAuthService{}
	h := NewAuthHandler(mockSvc, zap.NewNop(), &config.Config{})
	id := uuid.New()

	t.Run("Validation Failed", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("userID", id)
		req := httptest.NewRequest(http.MethodPut, "/user/email", strings.NewReader(`{"new_email":"bademail"}`))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req

		h.ChangeEmail(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "validation failed")
	})

	t.Run("Duplicate Email", func(t *testing.T) {
		mockSvc.On("ChangeEmail", mock.Anything, id, mock.Anything).Return(repository.ErrDuplicateEmail)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("userID", id)
		req := httptest.NewRequest(http.MethodPut, "/user/email", strings.NewReader(`{"new_email":"taken@test.com"}`))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req

		h.ChangeEmail(c)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "email already taken")
	})
}

func TestAuthHandler_ChangePassword_Errors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := &mockAuthService{}
	h := NewAuthHandler(mockSvc, zap.NewNop(), &config.Config{})
	id := uuid.New()

	t.Run("Validation Failed", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("userID", id)
		req := httptest.NewRequest(http.MethodPut, "/user/password", strings.NewReader(`{"old_password":"","new_password":"short"}`))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req

		h.ChangePassword(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "validation failed")
	})

	t.Run("Wrong Old Password", func(t *testing.T) {
		mockSvc.On("ChangePassword", mock.Anything, id, mock.Anything).Return(fmt.Errorf("invalid old password"))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("userID", id)
		req := httptest.NewRequest(http.MethodPut, "/user/password", strings.NewReader(`{"old_password":"wrong","new_password":"password123"}`))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req

		h.ChangePassword(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "wrong old password")
	})
}

