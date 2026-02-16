package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/config"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/model"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) Create(ctx context.Context, user *model.User) (uuid.UUID, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockAuthRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockAuthRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockAuthRepository) UpdateProfile(ctx context.Context, id uuid.UUID, username string) error {
	args := m.Called(ctx, id, username)
	return args.Error(0)
}

func (m *MockAuthRepository) UpdateEmail(ctx context.Context, id uuid.UUID, email string) error {
	args := m.Called(ctx, id, email)
	return args.Error(0)
}

func (m *MockAuthRepository) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	args := m.Called(ctx, id, hash)
	return args.Error(0)
}

func (m *MockAuthRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAuthRepository) GetUsers(ctx context.Context, limit, offset int) ([]*model.User, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.User), args.Error(1)
}

func setup(t *testing.T) (*authService, *MockAuthRepository, *config.Config) {
	mockRepo := new(MockAuthRepository)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:          "test-secret",
			ExpirationHours: 24,
		},
	}

	logger := zap.NewNop()

	svc := NewAuthService(mockRepo, logger, cfg).(*authService)
	return svc, mockRepo, cfg
}

////////////////////////////////////////////////////////////
//////////////////// REGISTER //////////////////////////////
////////////////////////////////////////////////////////////

func TestRegister(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()

	req := &model.CreateUserRequest{
		Username: "user",
		Email:    "user@test.com",
		Password: "password",
	}

	expectedID := uuid.New()

	repo.On("Create", ctx, mock.MatchedBy(func(u *model.User) bool {
		return u.Username == req.Username &&
			u.Email == req.Email &&
			bcrypt.CompareHashAndPassword([]byte(u.Password), []byte("password")) == nil
	})).Return(expectedID, nil).Once()

	id, err := svc.Register(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, expectedID, id)
	repo.AssertExpectations(t)
}

func TestRegister_RepoError(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()

	repo.On("Create", ctx, mock.Anything).
		Return(uuid.Nil, errors.New("db error")).Once()

	id, err := svc.Register(ctx, &model.CreateUserRequest{
		Username: "u", Email: "e", Password: "p",
	})

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, id)
}

////////////////////////////////////////////////////////////
//////////////////// LOGIN /////////////////////////////////
////////////////////////////////////////////////////////////

func TestLogin(t *testing.T) {
	svc, repo, cfg := setup(t)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	user := &model.User{
		ID:       uuid.New(),
		Username: "john",
		Email:    "john@test.com",
		Password: string(hash),
	}

	repo.On("GetByEmail", ctx, user.Email).
		Return(user, nil).Once()

	token, err := svc.Login(ctx, &model.LoginRequest{
		Email:    user.Email,
		Password: "secret",
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	parsed, err := jwt.ParseWithClaims(token, &model.UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWT.Secret), nil
		})

	assert.NoError(t, err)
	claims := parsed.Claims.(*model.UserClaims)

	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, "auth-service", claims.Issuer)
	assert.WithinDuration(t,
		time.Now().Add(24*time.Hour),
		claims.ExpiresAt.Time,
		time.Minute,
	)
}

func TestLogin_InvalidPassword(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	user := &model.User{ID: uuid.New(), Email: "e", Password: string(hash)}

	repo.On("GetByEmail", ctx, "e").
		Return(user, nil).Once()

	token, err := svc.Login(ctx, &model.LoginRequest{
		Email: "e", Password: "wrong",
	})

	assert.Error(t, err)
	assert.Empty(t, token)
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()

	repo.On("GetByEmail", ctx, "x").
		Return(nil, errors.New("not found")).Once()

	token, err := svc.Login(ctx, &model.LoginRequest{
		Email: "x", Password: "p",
	})

	assert.Error(t, err)
	assert.Empty(t, token)
}

func TestLogin_TokenSignError(t *testing.T) {
	svc, repo, cfg := setup(t)
	ctx := context.Background()

	// Ломаем secret
	cfg.JWT.Secret = ""

	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	user := &model.User{
		ID:       uuid.New(),
		Username: "john",
		Email:    "john@test.com",
		Password: string(hash),
	}

	repo.On("GetByEmail", ctx, user.Email).
		Return(user, nil).Once()

	token, err := svc.Login(ctx, &model.LoginRequest{
		Email:    user.Email,
		Password: "secret",
	})

	assert.Error(t, err)
	assert.Equal(t, "failed to generate token", err.Error())
	assert.Empty(t, token)
}

////////////////////////////////////////////////////////////
//////////////////// GET ///////////////////////////////////
////////////////////////////////////////////////////////////

func TestGetByID(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()
	id := uuid.New()

	user := &model.User{ID: id}

	repo.On("GetByID", ctx, id).Return(user, nil).Once()

	res, err := svc.GetByID(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, user, res)

}

func TestGetByID_Error(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()
	id := uuid.New()

	repo.On("GetByID", ctx, id).
		Return(nil, errors.New("error")).Once()

	res, err := svc.GetByID(ctx, id)

	assert.Error(t, err)
	assert.Nil(t, res)

}

func TestGetByEmail(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()

	user := &model.User{Email: "a"}

	repo.On("GetByEmail", ctx, "a").
		Return(user, nil).Once()

	res, err := svc.GetByEmail(ctx, "a")

	assert.NoError(t, err)
	assert.Equal(t, user, res)
}

func TestGetByEmail_Error(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()

	repo.On("GetByEmail", ctx, "x").
		Return(nil, errors.New("db")).Once()

	res, err := svc.GetByEmail(ctx, "x")

	assert.Error(t, err)
	assert.Nil(t, res)
}

////////////////////////////////////////////////////////////
//////////////////// CHANGE PROFILE ////////////////////////
////////////////////////////////////////////////////////////

func TestChangeProfile(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()
	id := uuid.New()

	repo.On("UpdateProfile", ctx, id, "new").
		Return(nil).Once()

	err := svc.ChangeProfile(ctx, id,
		&model.ChangeProfileRequest{NewUsername: "new"})

	assert.NoError(t, err)
}

func TestChangeProfile_Errors(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()
	id := uuid.New()

	repo.On("UpdateProfile", ctx, id, "dup").
		Return(repository.ErrDuplicateUsername).Once()

	err := svc.ChangeProfile(ctx, id,
		&model.ChangeProfileRequest{NewUsername: "dup"})
	assert.ErrorIs(t, err, repository.ErrDuplicateUsername)

	repo.On("UpdateProfile", ctx, id, "x").
		Return(errors.New("db crash")).Once()

	err = svc.ChangeProfile(ctx, id,
		&model.ChangeProfileRequest{NewUsername: "x"})
	assert.Equal(t, "internal error", err.Error())

	repo.On("UpdateProfile", ctx, id, "nf").
		Return(repository.ErrNotFound).Once()

	err = svc.ChangeProfile(ctx, id,
		&model.ChangeProfileRequest{NewUsername: "nf"})
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

////////////////////////////////////////////////////////////
//////////////////// CHANGE EMAIL //////////////////////////
////////////////////////////////////////////////////////////

func TestChangeEmail(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()
	id := uuid.New()

	repo.On("UpdateEmail", ctx, id, "e").
		Return(nil).Once()

	err := svc.ChangeEmail(ctx, id,
		&model.ChangeEmailRequest{NewEmail: "e"})

	assert.NoError(t, err)

	repo.On("UpdateEmail", ctx, id, "dup").
		Return(repository.ErrDuplicateEmail).Once()

	err = svc.ChangeEmail(ctx, id,
		&model.ChangeEmailRequest{NewEmail: "dup"})
	assert.ErrorIs(t, err, repository.ErrDuplicateEmail)

	repo.On("UpdateEmail", ctx, id, "nf").
		Return(repository.ErrNotFound).Once()

	err = svc.ChangeEmail(ctx, id,
		&model.ChangeEmailRequest{NewEmail: "nf"})
	assert.ErrorIs(t, err, repository.ErrNotFound)

	repo.On("UpdateEmail", ctx, id, "x").
		Return(errors.New("db")).Once()

	err = svc.ChangeEmail(ctx, id,
		&model.ChangeEmailRequest{NewEmail: "x"})
	assert.Equal(t, "internal error", err.Error())
}

////////////////////////////////////////////////////////////
//////////////////// CHANGE PASSWORD ///////////////////////
////////////////////////////////////////////////////////////

func TestChangePassword(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()
	id := uuid.New()

	hash, _ := bcrypt.GenerateFromPassword([]byte("old"), bcrypt.DefaultCost)
	user := &model.User{ID: id, Password: string(hash)}

	repo.On("GetByID", ctx, id).
		Return(user, nil).Once()

	repo.On("UpdatePassword", ctx, id, mock.MatchedBy(func(h string) bool {
		return bcrypt.CompareHashAndPassword([]byte(h), []byte("new")) == nil
	})).Return(nil).Once()

	err := svc.ChangePassword(ctx, id,
		&model.ChangePasswordRequest{
			OldPassword: "old",
			NewPassword: "new",
		})

	assert.NoError(t, err)

	repo.On("GetByID", ctx, id).
		Return(nil, errors.New("db")).Once()

	err = svc.ChangePassword(ctx, id,
		&model.ChangePasswordRequest{
			OldPassword: "old",
			NewPassword: "new",
		})
	assert.Error(t, err)

}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()
	id := uuid.New()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	user := &model.User{ID: id, Password: string(hash)}

	repo.On("GetByID", ctx, id).
		Return(user, nil).Once()

	err := svc.ChangePassword(ctx, id,
		&model.ChangePasswordRequest{
			OldPassword: "wrong",
			NewPassword: "new",
		})

	assert.Equal(t, "invalid old password", err.Error())
}

func TestChangePassword_UpdateError(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()
	id := uuid.New()

	hash, _ := bcrypt.GenerateFromPassword([]byte("old"), bcrypt.DefaultCost)
	user := &model.User{ID: id, Password: string(hash)}

	repo.On("GetByID", ctx, id).
		Return(user, nil).Once()

	repo.On("UpdatePassword", ctx, id, mock.Anything).
		Return(errors.New("db")).Once()

	err := svc.ChangePassword(ctx, id,
		&model.ChangePasswordRequest{
			OldPassword: "old",
			NewPassword: "new",
		})

	assert.Equal(t, "internal error", err.Error())
}

////////////////////////////////////////////////////////////
//////////////////// DELETE ////////////////////////////////
////////////////////////////////////////////////////////////

func TestDelete(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()
	id := uuid.New()

	repo.On("Delete", ctx, id).
		Return(nil).Once()

	err := svc.Delete(ctx, id)
	assert.NoError(t, err)
}

func TestDelete_Error(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()
	id := uuid.New()

	repo.On("Delete", ctx, id).
		Return(errors.New("db")).Once()

	err := svc.Delete(ctx, id)
	assert.Error(t, err)
}

////////////////////////////////////////////////////////////
//////////////////// GET USERS /////////////////////////////
////////////////////////////////////////////////////////////

func TestGetUsers(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()

	users := []*model.User{{ID: uuid.New()}}

	repo.On("GetUsers", ctx, 10, 0).
		Return(users, nil).Once()

	res, err := svc.GetUsers(ctx, -1, -1)

	assert.NoError(t, err)
	assert.Equal(t, users, res)

}

func TestGetUsers_Error(t *testing.T) {
	svc, repo, _ := setup(t)
	ctx := context.Background()

	repo.On("GetUsers", ctx, 10, 0).
		Return(nil, errors.New("db")).Once()

	res, err := svc.GetUsers(ctx, -1, -1)

	assert.Error(t, err)
	assert.Nil(t, res)
}
