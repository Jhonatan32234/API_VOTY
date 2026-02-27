package api

import (
	"api_voty/internal/models"
	"api_voty/internal/utils"
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type UserAPI struct {
	userModel *models.UserModel
	pollModel *models.PollModel
	Hub       *Hub
}

func NewUserAPI(userModel *models.UserModel, pollModel *models.PollModel, hub *Hub) *UserAPI {
	return &UserAPI{
		userModel: userModel,
		pollModel: pollModel,
		Hub:       hub,
	}
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

// Estructura de salida para la API
type PollOutput struct {
	ID               string         `json:"id"`
	Title            string         `json:"title"`
	Options          []OptionOutput `json:"options"`
	Voted            bool           `json:"voted"`
	SelectedOptionID string         `json:"selected_option_id,omitempty"`
	IsOpen           bool           `json:"is_open"`
}

type OptionOutput struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	VotesCount int    `json:"votes_count"`
}

type ListPollsResponse struct {
	Body []PollOutput
}

type UpdatePollRequest struct {
	ID   string `path:"id"`
	Body struct {
		Title   string   `json:"title"`
		IsOpen  bool     `json:"is_open"`
		Options []string `json:"options,omitempty"`
	}
}

func (a *UserAPI) UpdatePoll(ctx context.Context, input *UpdatePollRequest) (*GetPollResponse, error) {
	pollID, _ := strconv.Atoi(input.ID)
	
	p, err := a.pollModel.Update(ctx, pollID, input.Body.Title, input.Body.IsOpen, input.Body.Options)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error al actualizar", err)
	}
	println(p)

	// Mapeamos a PollOutput (Reutilizando la lógica de GetPoll)
    // Esto asegura que el "voted" y "selected_option_id" se mantengan correctos
	return a.GetPoll(ctx, &GetPollRequest{ID: input.ID}) 
}

type GetPollRequest struct {
	ID string `path:"id" doc:"ID de la encuesta"`
}

type GetPollResponse struct {
	Body PollOutput
}

func (a *UserAPI) GetPoll(ctx context.Context, input *GetPollRequest) (*GetPollResponse, error) {
	// Convertimos el ID de string a int (asumiendo que tus IDs son enteros)
	pollID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("ID de encuesta inválido", err)
	}

	userID := utils.GetUserIDFromContext(ctx)
	
	p, err := a.pollModel.GetByIDWithUserStatus(ctx, pollID, userID)
	if err != nil {
		return nil, huma.Error404NotFound("Encuesta no encontrada", err)
	}

	// Reutilizamos la lógica de mapeo
	voted := len(p.Edges.Votes) > 0
	var selectedID string
	if voted && p.Edges.Votes[0].Edges.PollOption != nil {
		selectedID = fmt.Sprintf("%d", p.Edges.Votes[0].Edges.PollOption.ID)
	}

	opts := make([]OptionOutput, len(p.Edges.Options))
	for j, o := range p.Edges.Options {
		opts[j] = OptionOutput{
			ID:         fmt.Sprintf("%d", o.ID),
			Text:       o.Text,
			VotesCount: o.VotesCount,
		}
	}

	return &GetPollResponse{
		Body: PollOutput{
			ID:               fmt.Sprintf("%d", p.ID),
			Title:            p.Title,
			Options:          opts,
			Voted:            voted,
			SelectedOptionID: selectedID,
			IsOpen:           p.IsOpen,
		},
	}, nil
}

type DeletePollRequest struct {
	ID string `path:"id" doc:"ID de la encuesta a eliminar"`
}

func (a *UserAPI) DeletePoll(ctx context.Context, input *DeletePollRequest) (*struct{}, error) {
	err := a.pollModel.Delete(ctx, input.ID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error al eliminar encuesta", err)
	}
	return nil, nil
}

func (a *UserAPI) ListPolls(ctx context.Context, input *struct{}) (*ListPollsResponse, error) {
	// Obtenemos el ID del usuario desde el JWT
	userID := utils.GetUserIDFromContext(ctx)
	polls, err := a.pollModel.ListAllWithUserStatus(ctx, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error al listar", err)
	}

	output := make([]PollOutput, len(polls))
	for i, p := range polls {
		voted := len(p.Edges.Votes) > 0
		var selectedID string
		if voted {
			// p.Edges.Votes[0] es el voto del usuario
			// .Edges.PollOption es la relación cargada gracias al .WithPollOption() anterior
			if p.Edges.Votes[0].Edges.PollOption != nil {
				selectedID = fmt.Sprintf("%d", p.Edges.Votes[0].Edges.PollOption.ID)
			}
		}

		opts := make([]OptionOutput, len(p.Edges.Options))
		for j, o := range p.Edges.Options {
			opts[j] = OptionOutput{ID: fmt.Sprintf("%d", o.ID), Text: o.Text, VotesCount: o.VotesCount}
		}

		output[i] = PollOutput{
			ID:               fmt.Sprintf("%d", p.ID),
			Title:            p.Title,
			Options:          opts,
			Voted:            voted,
			SelectedOptionID: selectedID,
			IsOpen:           p.IsOpen,
		}
	}

	return &ListPollsResponse{Body: output}, nil
}

func (a *UserAPI) SubscribeVotes(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Canal local para este cliente específico
	clientChan := make(chan VoteUpdate)
	a.Hub.Register <- clientChan

	// Asegurar limpieza al desconectar
	defer func() {
		a.Hub.Unregister <- clientChan
		conn.Close()
	}()

	// Escuchar actualizaciones del Hub y enviarlas al móvil
	for update := range clientChan {
		err := conn.WriteJSON(update)
		if err != nil {
			break // Si falla la escritura (ej: el móvil perdió señal), cerramos
		}
	}
}

// Estructura para recibir los datos
type CreatePollRequest struct {
	Body struct {
		Title   string   `json:"title" doc:"Título de la encuesta" example:"¿Cuál es el mejor lenguaje?"`
		Options []string `json:"options" doc:"Lista de opciones" example:"[\"Go\", \"Kotlin\"]"`
	}
}

func (a *UserAPI) CreatePoll(ctx context.Context, input *CreatePollRequest) (*struct{}, error) {
	// 1. Crear la encuesta
	p, err := a.pollModel.Create(ctx, input.Body.Title)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error al crear la encuesta", err)
	}

	// 2. Crear las opciones
	for _, optText := range input.Body.Options {
		if err := a.pollModel.AddOption(ctx, fmt.Sprintf("%d", p.ID), optText); err != nil {
			return nil, huma.Error500InternalServerError("Error al crear las opciones", err)
		}
	}

	return nil, nil
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
			AuthMiddleware(app),
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
			AuthMiddleware(app),
		},
	}, userAPI.DeleteUser)

	huma.Register(app, huma.Operation{
		OperationID: "post-vote",
		Method:      http.MethodPost,
		Path:        "/polls/{poll_id}/vote/{option_id}",
		Summary:     "Emitir un voto",
		Tags:        []string{"Voting"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
		Middlewares: huma.Middlewares{AuthMiddleware(app)},
	}, userAPI.PostVote)

	router.HandleFunc("/ws/votes", userAPI.SubscribeVotes)

	huma.Register(app, huma.Operation{
		OperationID: "create-poll",
		Method:      http.MethodPost,
		Path:        "/polls",
		Summary:     "Crear una nueva encuesta",
		Description: "Crea una encuesta con sus opciones iniciales. Solo para administradores (en el futuro).",
		Tags:        []string{"Voting"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
		Middlewares: huma.Middlewares{AuthMiddleware(app)},
	}, userAPI.CreatePoll)

	huma.Register(app, huma.Operation{
		OperationID: "list-polls",
		Method:      http.MethodGet,
		Path:        "/polls",
		Summary:     "Listar encuestas",
		Tags:        []string{"Voting"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
		Middlewares: huma.Middlewares{AuthMiddleware(app)},
	}, userAPI.ListPolls)

	// Actualizar Encuesta
	huma.Register(app, huma.Operation{
		OperationID: "update-poll",
		Method:      http.MethodPut,
		Path:        "/polls/{id}",
		Summary:     "Actualizar encuesta",
		Tags:        []string{"Voting"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
		Middlewares: huma.Middlewares{AuthMiddleware(app)},
	}, userAPI.UpdatePoll)

	// Eliminar Encuesta
	huma.Register(app, huma.Operation{
		OperationID: "delete-poll",
		Method:      http.MethodDelete,
		Path:        "/polls/{id}",
		Summary:     "Eliminar encuesta",
		Tags:        []string{"Voting"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
		Middlewares: huma.Middlewares{AuthMiddleware(app)},
	}, userAPI.DeletePoll)
	
	huma.Register(app, huma.Operation{
    OperationID: "get-poll-by-id",
    Method:      http.MethodGet,
    Path:        "/polls/{id}",
    Summary:     "Obtener detalle de una encuesta",
    Tags:        []string{"Voting"},
    Security:    []map[string][]string{{"bearerAuth": {}}},
    Middlewares: huma.Middlewares{AuthMiddleware(app)},
}, userAPI.GetPoll)
}
