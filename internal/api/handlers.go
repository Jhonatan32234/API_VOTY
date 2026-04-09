package api

import (
	"api_voty/internal/models"
	"api_voty/internal/utils"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type UserAPI struct {
	userModel   *models.UserModel
	pollModel   *models.PollModel
	deviceModel *models.DeviceModel
	fcmClient   *messaging.Client
	Hub         *Hub
}

func NewUserAPI(userModel *models.UserModel, pollModel *models.PollModel, deviceModel *models.DeviceModel, fcmClient *messaging.Client, hub *Hub) *UserAPI {
	return &UserAPI{
		userModel:   userModel,
		pollModel:   pollModel,
		deviceModel: deviceModel,
		fcmClient:   fcmClient,
		Hub:         hub,
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
		Avatar   *string `json:"avatar,omitempty"`
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
    ImageURL   string `json:"image_url,omitempty"` // <-- CRUCIAL
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

		// ... (dentro de ListPolls)
		opts := make([]OptionOutput, len(p.Edges.Options))
		for j, o := range p.Edges.Options {
		    opts[j] = OptionOutput{
		        ID:         fmt.Sprintf("%d", o.ID), 
		        Text:       o.Text, 
		        VotesCount: o.VotesCount,
		        ImageURL:   o.ImageURL, // <--- MAPEA EL CAMPO DE LA DB AQUÍ
		    }
		}
// ...

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

    // CORRECCIÓN: El canal debe ser del tipo que espera el Hub (SocketMessage)
    clientChan := make(chan SocketMessage) 
    a.Hub.Register <- clientChan

    defer func() {
        a.Hub.Unregister <- clientChan
        conn.Close()
    }()

    for update := range clientChan {
        // CORRECCIÓN: No enviamos todo el objeto SocketMessage (que tiene el campo Event),
        // enviamos solo el Payload para que coincida con lo que espera el JSON de la App.
        err := conn.WriteJSON(update.Payload) 
        if err != nil {
            break 
        }
    }
}

type DeviceTokenRequest struct {
	Body struct {
		Token string `json:"token" doc:"FCM Token del dispositivo" example:"fcm-token-xyz-123"`
	}
}

func (a *UserAPI) UpdateDeviceToken(ctx context.Context, input *DeviceTokenRequest) (*struct{}, error) {
	userID := utils.GetUserIDFromContext(ctx)
	err := a.deviceModel.UpdateToken(ctx, userID, input.Body.Token)
	if err != nil {
		return nil, huma.Error500InternalServerError("No se pudo guardar el token del dispositivo", err)
	}
	return nil, nil
}

func (a *UserAPI) sendPushNotification(ctx context.Context, targetToken, title, body string, data map[string]string) {
    if a.fcmClient == nil {
        return
    }

    // Aseguramos que el título y el cuerpo viajen en el mapa de datos
    // Esto permite que tu código Kotlin los extraiga manualmente
    if data == nil {
        data = make(map[string]string)
    }
    data["title"] = title
    data["body"] = body

    message := &messaging.Message{
        // IMPORTANTE: Eliminamos el campo Notification: &messaging.Notification{...}
        // Al enviar SOLO 'Data', forzamos a Android a ejecutar onMessageReceived
        Data:  data, 
        Token: targetToken,
        Android: &messaging.AndroidConfig{
            Priority: "high", // Requerido para que el dispositivo despierte de inmediato
        },
    }

    _, err := a.fcmClient.Send(ctx, message)
    if err != nil {
        log.Printf("Error enviando notificación: %v", err)
        if messaging.IsRegistrationTokenNotRegistered(err) {
            log.Printf("Borrando token FCM inválido: %s", targetToken)
            _ = a.deviceModel.DeleteToken(ctx, targetToken)
        }
    }
}

type TestFCMResponse struct {
	Body struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}
}

func (a *UserAPI) TestFCM(ctx context.Context, input *struct{}) (*TestFCMResponse, error) {
	if a.fcmClient == nil {
		return nil, huma.Error500InternalServerError("Firebase Client no está inicializado. Revisa tu archivo .env y el JSON de credenciales.", nil)
	}

	// Intentamos enviar a un token ficticio para verificar conectividad
	res, err := a.fcmClient.Send(ctx, &messaging.Message{
		Notification: &messaging.Notification{Title: "Prueba", Body: "Backend conectado"},
		Token:        "token_ficticio_para_pruebas",
	})

	resp := &TestFCMResponse{}
	resp.Body.Status = "Conexión con Firebase verificada"
	resp.Body.Message = fmt.Sprintf("Resultado (se esperaba error de token): %v. ID: %s", err, res)
	return resp, nil
}

// Estructura para recibir los datos
type CreatePollRequest struct {
	Body struct {
		Title   string   `json:"title" doc:"Título de la encuesta" example:"¿Cuál es el mejor lenguaje?"`
		Options []string `json:"options" doc:"Lista de opciones" example:"[\"Go\", \"Kotlin\"]"`
	}
}

type CreatePollInput struct {
    // Usamos 'form' para que Huma sepa que vienen en el cuerpo del multipart
    Title   string          `form:"title" doc:"Título de la encuesta"`
    Options []string        `form:"options" doc:"Lista de opciones de texto"`
    Images  []huma.FormFile `form:"images" doc:"Archivos de imagen para las opciones"`
}

func (a *UserAPI) CreatePoll(ctx context.Context, input *CreatePollInput) (*struct{}, error) {
    // 1. Crear la cabecera
    p, err := a.pollModel.Create(ctx, input.Title)
    if err != nil {
        return nil, huma.Error500InternalServerError("Error DB", err)
    }

    var optionsOutput []map[string]interface{}
    var imageUrlsForPush []string

    // 2. Procesar opciones e imágenes
    for i, optText := range input.Options {
        var imagePath string

        // Verificar si hay una imagen para este índice
        // ... dentro del bucle de input.Images
for i, optText := range input.Options {
	print(optText)
    var imagePath string

    if i < len(input.Images) {
        formFile := input.Images[i]

        // 1. Generar nombre único usando el campo Filename que se ve en tu imagen
        ext := filepath.Ext(formFile.Filename)
        fileName := fmt.Sprintf("poll_%d_opt_%d_%d%s", p.ID, i, time.Now().Unix(), ext)
        fullPath := filepath.Join("uploads", fileName)

        // 2. Crear el destino en disco
        out, err := os.Create(fullPath)
        if err != nil {
            return nil, huma.Error500InternalServerError("Error creando archivo local", err)
        }

        // 3. COPIAR DIRECTAMENTE
        // Como formFile tiene el método Read, puedes pasarlo directamente a io.Copy
        _, err = io.Copy(out, formFile)
        
        // Es vital cerrar el archivo de salida y el formFile (que tiene el método Close)
        out.Close()
        formFile.Close() 

        if err != nil {
            return nil, huma.Error500InternalServerError("Error escribiendo imagen", err)
        }

        imagePath = "https://apivoty.jhonatanzc.fun/uploads/" + fileName
        imageUrlsForPush = append(imageUrlsForPush, imagePath)
    }
}
        // Guardar en DB
        _ = a.pollModel.AddOption(ctx, fmt.Sprintf("%d", p.ID), optText, imagePath)

        optionsOutput = append(optionsOutput, map[string]interface{}{
            "id":          fmt.Sprintf("%d", i+1),
            "text":        optText,
            "image_url":   imagePath,
            "votes_count": 0,
        })
    }

    // 4. Construir objeto completo para tiempo real
    fullPoll := map[string]interface{}{
        "id":      fmt.Sprintf("%d", p.ID),
        "title":   p.Title,
        "options": optionsOutput,
        "voted":   false,
        "is_open": true,
    }

    // 5. Notificar por WebSocket (Broadcast)
    go func() {
        a.Hub.Broadcast <- SocketMessage{
            Event:   "poll_created",
            Payload: fullPoll,
        }
    }()

    // 6. Notificaciones Push (FCM)
    go func() {
        tokens, _ := a.deviceModel.GetAllTokens(context.Background())
        
        // Incluimos las URLs de las imágenes en los datos para el WorkManager de Android
        notificationData := map[string]string{
            "type":        "NEW_POLL",
            "poll_id":     fmt.Sprintf("%d", p.ID),
            "image_urls":  strings.Join(imageUrlsForPush, ","),
        }

        for _, token := range tokens {
            a.sendPushNotification(context.Background(), token, "¡Nueva Encuesta!", p.Title, notificationData)
        }
    }()

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
		Avatar:   req.Body.Avatar,
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
    Description: "Crea una encuesta con texto e imágenes mediante multipart/form-data.",
    Tags:        []string{"Voting"},
    Security:    []map[string][]string{{"bearerAuth": {}}},
    Middlewares: huma.Middlewares{AuthMiddleware(app)},
}, func(ctx context.Context, input *CreatePollInput) (*struct{}, error) {
    // Aquí llamas a la lógica de tu userAPI.CreatePoll pasando el input
    return userAPI.CreatePoll(ctx, input)
})

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

	huma.Register(app, huma.Operation{
		OperationID: "update-device-token",
		Method:      http.MethodPost,
		Path:        "/device-token",
		Summary:     "Actualizar Token FCM",
		Description: "Registra o actualiza el token de notificaciones push para el usuario actual.",
		Tags:        []string{"Users"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
		Middlewares: huma.Middlewares{AuthMiddleware(app)},
	}, userAPI.UpdateDeviceToken)

	huma.Register(app, huma.Operation{
		OperationID: "test-fcm",
		Method:      http.MethodGet,
		Path:        "/test-fcm",
		Summary:     "Probar conexión Firebase",
		Description: "Envía una notificación a un token falso para verificar si las credenciales de Google Cloud son válidas.",
		Tags:        []string{"Debug"},
	}, userAPI.TestFCM)
}
