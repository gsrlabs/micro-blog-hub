package model

const dateFormat = "02.01.2006 15:04:05"

func ToDomain(req CreateUserRequest) (*User, error) {

	return &User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}, nil
}

func ToResponse(user *User) UserResponse {

	createdAt := user.CreatedAt.Format(dateFormat)
	updatedAt := user.UpdatedAt.Format(dateFormat)

	return UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
