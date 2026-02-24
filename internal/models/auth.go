package models

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"pruebas_doc/ent"
	"pruebas_doc/ent/user"
	"pruebas_doc/internal/utils"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type AuthModel struct {
	client *ent.Client
	db	 *sql.DB
}

func NewAuthModel(client *ent.Client, db *sql.DB) *AuthModel {
	return &AuthModel{client: client, db: db}
}

func (m *AuthModel) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	exists, err := m.client.User.Query().
		Where(user.Email(req.Email)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("email already registered")
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(req.Password), 14)
	if err != nil {
		return nil, err
	}

	newUser, err := m.client.User.
		Create().
		SetID(uuid.New().String()).
		SetEmail(req.Email).
		SetName(req.Name).
		SetPassword(string(hashedPass)).
		SetActive(true).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User: UserResponse{
			ID:        newUser.ID,
			Email:     newUser.Email,
			Name:      newUser.Name,
			Active:    newUser.Active,
			CreatedAt: newUser.CreatedAt,
			UpdatedAt: newUser.UpdatedAt,
		},
	}, nil
}

func (m *AuthModel) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	user, err := m.client.User.Query().
		Where(user.Email(req.Email)).
		Only(ctx)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.Active {
		return nil, errors.New("user is inactive")
	}

	token, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User: UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			Active:    user.Active,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}, nil
}