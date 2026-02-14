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

	createdAt := user.CreatedAt.Local().Format(dateFormat)
	updatedAt := user.UpdatedAt.Local().Format(dateFormat)

	return UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func ToUsersResponse(users []*User) []UsersResponse {
	// Сразу выделяем память под нужное количество элементов

	result := make([]UsersResponse, 0, len(users))

	for _, user := range users {
		// Добавляем .Local() для корректного отображения времени
		createdAt := user.CreatedAt.Local().Format(dateFormat)
		updatedAt := user.UpdatedAt.Local().Format(dateFormat)

		res := UsersResponse{
			ID:        user.ID,
			Username:  user.Username,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}

		result = append(result, res)
	}
	return result
}
