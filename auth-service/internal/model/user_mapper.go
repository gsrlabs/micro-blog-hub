package model

import "time"

const dateFormat = "02.01.2006 15:04:05"

func ToDomain(req CreateUserRequest) (*User, error) {

	return &User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}, nil
}

func ToResponse(user *User) UserResponse {

	createdAt := dateFormating(user.CreatedAt)
	updatedAt := dateFormating(user.UpdatedAt)

	return UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func ToUsersResponse(users []*User) []UsersResponse {

	result := make([]UsersResponse, 0, len(users))

	for _, u := range users {

		createdAt := dateFormating(u.CreatedAt)
		updatedAt := dateFormating(u.UpdatedAt)

		user := UsersResponse{
			ID:        u.ID,
			Username:  u.Username,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}

		result = append(result, user)
	}
	return result
}

func dateFormating(date time.Time) string {
	return date.Local().Format(dateFormat)
}
