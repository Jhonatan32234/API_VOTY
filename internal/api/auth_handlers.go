package api

import (
	"context"

	"api_voty/internal/models"
	"api_voty/internal/utils"

	"github.com/danielgtaylor/huma/v2"
)

type AuthAPI struct {
	authModel *models.AuthModel
	userModel *models.UserModel
}

func NewAuthAPI(authModel *models.AuthModel, userModel *models.UserModel) *AuthAPI {
	return &AuthAPI{authModel: authModel,userModel: userModel,}
}

type RegisterRequest struct {
	Body models.RegisterRequest
}

type LoginRequest struct {
	Body models.LoginRequest
}

type AuthResponse struct {
	Body models.AuthResponse
}

func (a *AuthAPI) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	result, err := a.authModel.Register(ctx, req.Body)
	if err != nil {
		return nil, huma.Error400BadRequest("Registration failed", err)
	}
	return &AuthResponse{Body: *result}, nil
}

// Login handler
func (a *AuthAPI) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	result, err := a.authModel.Login(ctx, req.Body)
	if err != nil {
		return nil, huma.Error401Unauthorized("Login failed", err)
	}
	return &AuthResponse{Body: *result}, nil
}

type ProfileResponse struct {
	Body models.UserResponse
}

func (a *AuthAPI) GetProfile(ctx context.Context, input *struct{}) (*ProfileResponse, error) {
    userID := utils.GetUserIDFromContext(ctx) 
    
    // Ahora a.userModel ya existe porque lo añadimos arriba
    user, err := a.userModel.GetByID(ctx, userID)
    if err != nil {
        return nil, huma.Error404NotFound("Usuario no encontrado", err)
    }
    
    // Usamos ProfileResponse que es el tipo que definiste justo arriba en el archivo
    return &ProfileResponse{Body: *user}, nil
}
