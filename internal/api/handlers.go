package api

import (
	"context"
	"net/http"
	
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"pruebas_doc/internal/models"
)

type UserAPI struct {
	userModel *models.UserModel
}

func NewUserAPI(userModel *models.UserModel) *UserAPI {
	return &UserAPI{userModel: userModel}
}

type CreateUserRequest struct {
	Body struct {
		Email    string `json:"email" doc:"User email - Ej: usuario@dominio.com"`
		Name     string `json:"name" example:"John Doe" doc:"User full name"`
		Password string `json:"password" example:"secret123" doc:"User password"`
		Active   bool   `json:"active" example:"true" doc:"User active status"`
	}
}

type UserResponse struct {
	Body models.UserResponse
}

type UsersResponse struct {
	Body []models.UserResponse
}

type GetUserRequest struct {
	ID string `path:"id" doc:"User ID"`
}

type UpdateUserRequest struct {
	ID   string `path:"id" doc:"User ID"`
	Body struct {
		Email    *string `json:"email,omitempty" example:"updated@example.com"`
		Name     *string `json:"name,omitempty" example:"Jane Doe"`
		Password *string `json:"password,omitempty" example:"newpass123"`
		Active   *bool   `json:"active,omitempty" example:"false"`
	}
}

type DeleteUserRequest struct {
	ID string `path:"id" doc:"User ID"`
}

func (a *UserAPI) CreateUser(ctx context.Context, req *CreateUserRequest) (*UserResponse, error) {
	input := models.UserInput{
		Email:    req.Body.Email,
		Name:     req.Body.Name,
		Password: req.Body.Password,
		Active:   &req.Body.Active,
	}

	user, err := a.userModel.Create(ctx, input)
	if err != nil {
		return nil, huma.Error400BadRequest("Error creating user", err)
	}

	return &UserResponse{Body: *user}, nil
}

func (a *UserAPI) ListUsers(ctx context.Context, req *struct{}) (*UsersResponse, error) {
	users, err := a.userModel.GetAll(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error fetching users", err)
	}
	
	responseUsers := make([]models.UserResponse, len(users))
	for i, u := range users {
		responseUsers[i] = *u
	}
	
	return &UsersResponse{Body: responseUsers}, nil
}

func (a *UserAPI) GetUser(ctx context.Context, req *GetUserRequest) (*UserResponse, error) {
	user, err := a.userModel.GetByID(ctx, req.ID)
	if err != nil {
		return nil, huma.Error404NotFound("User not found", err)
	}
	return &UserResponse{Body: *user}, nil
}

func (a *UserAPI) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*UserResponse, error) {
	input := models.UserUpdateInput{
		Email:    req.Body.Email,
		Name:     req.Body.Name,
		Password: req.Body.Password,
		Active:   req.Body.Active,
	}

	user, err := a.userModel.Update(ctx, req.ID, input)
	if err != nil {
		return nil, huma.Error400BadRequest("Error updating user", err)
	}
	return &UserResponse{Body: *user}, nil
}

func (a *UserAPI) DeleteUser(ctx context.Context, req *DeleteUserRequest) (*struct{}, error) {
	err := a.userModel.Delete(ctx, req.ID)
	if err != nil {
		return nil, huma.Error404NotFound("User not found", err)
	}
	return nil, nil
}

func SetupRoutes(router *http.ServeMux, userAPI *UserAPI, authAPI *AuthAPI) {
	config := huma.DefaultConfig("User CRUD API", "1.0.0")
	config.DocsPath = "/docs"
	config.OpenAPIPath = "/openapi.json"

	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearerAuth": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  "Ingresa tu token JWT en el formato: Bearer <token>",
		},
	}
	
	app := humago.New(router, config)
	
	huma.Register(app, huma.Operation{
		OperationID: "register",
		Method:      http.MethodPost,
		Path:        "/register",
		Description: "Registra un nuevo usuario en el sistema",
		Summary:     "Register new user",
		Tags:        []string{"Auth"},
	}, authAPI.Register)
	
	huma.Register(app, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/login",
		Summary:     "Login user",
		Tags:        []string{"Auth"},
	}, authAPI.Login)
	
	huma.Register(app, huma.Operation{
		OperationID: "get-profile",
		Method:      http.MethodGet,
		Path:        "/profile",
		Summary:     "Get user profile",
		Tags:        []string{"Users"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
		Middlewares: huma.Middlewares{
			AuthMiddleware(app), 
		},
	}, authAPI.GetProfile)
	
	huma.Register(app, huma.Operation{
		OperationID: "list-users",
		Method:      http.MethodGet,
		Path:        "/users",
		Summary:     "List all users",
		Tags:        []string{"Users"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
		Middlewares: huma.Middlewares{
			AuthMiddleware(app),
		},
	}, userAPI.ListUsers)
	
	huma.Register(app, huma.Operation{
		OperationID: "get-user",
		Method:      http.MethodGet,
		Path:        "/users/{id}",
		Summary:     "Get a user by ID",
		Tags:        []string{"Users"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
		Middlewares: huma.Middlewares{
        AuthMiddleware(app), // <-- Pásalo directamente así
        },
	}, userAPI.GetUser)
	
	huma.Register(app, huma.Operation{
		OperationID: "update-user",
		Method:      http.MethodPut,
		Path:        "/users/{id}",
		Summary:     "Update a user",
		Tags:        []string{"Users"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
		Middlewares: huma.Middlewares{
        AuthMiddleware(app), // <-- Pásalo directamente así
        },
	}, userAPI.UpdateUser)
	
	huma.Register(app, huma.Operation{
		OperationID: "delete-user",
		Method:      http.MethodDelete,
		Path:        "/users/{id}",
		Summary:     "Delete a user",
		Tags:        []string{"Users"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
		Middlewares: huma.Middlewares{
        AuthMiddleware(app), // <-- Pásalo directamente así
        },
	}, userAPI.DeleteUser)
}