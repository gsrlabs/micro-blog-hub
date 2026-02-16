package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMappers(t *testing.T) {
	t.Run("ToDomain", func(t *testing.T) {
		req := CreateUserRequest{
			Username: "tester",
			Email:    "test@example.com",
			Password: "securepassword",
		}
		user, err := ToDomain(req)
		assert.NoError(t, err)
		assert.Equal(t, req.Username, user.Username)
		assert.Equal(t, req.Email, user.Email)
		assert.Equal(t, req.Password, user.Password)
	})

	t.Run("ToResponse", func(t *testing.T) {
		id := uuid.New()
		now := time.Date(2026, 2, 15, 13, 0, 0, 0, time.UTC)

		user := &User{
			ID:        id,
			Username:  "tester",
			Email:     "test@example.com",
			CreatedAt: now,
			UpdatedAt: now,
		}

		resp := ToResponse(user)
		assert.Equal(t, id, resp.ID)
		assert.Equal(t, user.Username, resp.Username)
		// Проверяем формат даты (02.01.2006 15:04:05)
		assert.Equal(t, now.Local().Format(dateFormat), resp.CreatedAt)
	})

	t.Run("ToUsersResponse", func(t *testing.T) {
		users := []*User{
			{ID: uuid.New(), Username: "user1", CreatedAt: time.Now()},
			{ID: uuid.New(), Username: "user2", CreatedAt: time.Now()},
		}

		resp := ToUsersResponse(users)
		assert.Len(t, resp, 2)
		assert.Equal(t, users[0].Username, resp[0].Username)
		assert.Equal(t, users[1].Username, resp[1].Username)
	})
}
