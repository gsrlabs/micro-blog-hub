package model

func ToDomain(req CreateUserRequest) (*User, error) {

	return &User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}, nil
}

func ToResponse(user *User) UserResponse {

	return UserResponse{
		ID:       user.ID,
		Username: user.Username,
	}
}
