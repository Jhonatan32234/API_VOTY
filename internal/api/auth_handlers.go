package api

import (
	"context"

	"pruebas_doc/internal/models"
	"pruebas_doc/internal/utils"

	"github.com/danielgtaylor/huma/v2"
)

type AuthAPI struct {
	authModel *models.AuthModel
}

func NewAuthAPI(authModel *models.AuthModel) *AuthAPI {
	return &AuthAPI{authModel: authModel}
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

func (a *AuthAPI) GetProfile(ctx context.Context, req *struct{}) (*ProfileResponse, error) {
	userID := utils.GetUserIDFromContext(ctx)
	if userID == "" {
		return nil, huma.Error401Unauthorized("Not authenticated")
	}
	
	email := utils.GetUserEmailFromContext(ctx)
	
	return &ProfileResponse{
		Body: models.UserResponse{
			ID:     userID,
			Email:  email,
			Name:   "Test User",
			Active: true,
		},
	}, nil
}