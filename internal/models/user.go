package models

import (
	"context"
	"time"

	"database/sql"
	"pruebas_doc/ent"
	"pruebas_doc/ent/user"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Active   *bool  `json:"active"`
}

type UserUpdateInput struct {
	Email    *string `json:"email"`
	Name     *string `json:"name"`
	Password *string `json:"password"`
	Active   *bool   `json:"active"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserModel struct {
	client *ent.Client
	db     *sql.DB
}

func NewUserModel(client *ent.Client, db *sql.DB) *UserModel {
    return &UserModel{
        client: client,
        db:     db,
    }
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func (m *UserModel) Create(ctx context.Context, input UserInput) (*UserResponse, error) {
	hashedPass, err := hashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	active := true
	if input.Active != nil {
		active = *input.Active
	}

	user, err := m.client.User.
		Create().
		SetID(uuid.New().String()).
		SetEmail(input.Email).
		SetName(input.Name).
		SetPassword(hashedPass).
		SetActive(active).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	
	if err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Active:    user.Active,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

func (m *UserModel) GetAll(ctx context.Context) ([]*UserResponse, error) {
    query := "SELECT id, email, name, active, created_at, updated_at FROM users"
    
    rows, err := m.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var responses []*UserResponse
    for rows.Next() {
        u := &UserResponse{}
        err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Active, &u.CreatedAt, &u.UpdatedAt)
        if err != nil {
            return nil, err
        }
        responses = append(responses, u)
    }
    return responses, nil
}

func (m *UserModel) GetByID(ctx context.Context, id string) (*UserResponse, error) {
	u, err := m.client.User.
		Query().
		Where(user.ID(id)).
		Only(ctx)
	
	if err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Active:    u.Active,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}, nil
}

func (m *UserModel) Update(ctx context.Context, id string, input UserUpdateInput) (*UserResponse, error) {
	update := m.client.User.UpdateOneID(id).
		SetUpdatedAt(time.Now())

	if input.Email != nil {
		update.SetEmail(*input.Email)
	}
	if input.Name != nil {
		update.SetName(*input.Name)
	}
	if input.Password != nil {
		hashedPass, err := hashPassword(*input.Password)
		if err != nil {
			return nil, err
		}
		update.SetPassword(hashedPass)
	}
	if input.Active != nil {
		update.SetActive(*input.Active)
	}

	u, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Active:    u.Active,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}, nil
}

func (m *UserModel) Delete(ctx context.Context, id string) error {
	return m.client.User.
		DeleteOneID(id).
		Exec(ctx)
}