package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"

	"api_voty/ent"
	"api_voty/ent/migrate"
	"api_voty/internal/api"
	"api_voty/internal/models"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "3306"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "rnonet"
	}
	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		dbPass = "pass"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "test_db"
	}

	ctx := context.Background()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbUser, dbPass, dbHost, dbPort, dbName)

	log.Printf("Intentando conectar a DB: %s en %s:%s como usuario %s", dbName, dbHost, dbPort, dbUser)

	var db *sql.DB
	var err error

	// Implementación de reintentos para la conexión a la base de datos
	for i := 0; i < 10; i++ {
		db, err = sql.Open("mysql", dsn)
		if err == nil {
			err = db.Ping()
		}

		if err == nil {
			break
		}

		log.Printf("Esperando a la base de datos... (intento %d/10): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("No se pudo conectar a la base de datos tras varios intentos: %v", err)
	}
	drv := entsql.OpenDB(dialect.MySQL, db)
	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	// En el momento de la migración
	if err := client.Schema.Create(
		ctx,
		migrate.WithForeignKeys(true), // Asegura que gestione FKs
		migrate.WithDropColumn(true),  // Permite cambios estructurales
		migrate.WithDropIndex(true),
	); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	// Inicializar Firebase Messaging
	var fcmClient *messaging.Client
	fbConfigPath := os.Getenv("FIREBASE_CONFIG_PATH")
	if fbConfigPath != "" {
		opt := option.WithCredentialsFile(fbConfigPath)
		app, err := firebase.NewApp(ctx, nil, opt)
		if err != nil {
			log.Printf("Error inicializando Firebase: %v", err)
		} else {
			fcmClient, err = app.Messaging(ctx)
			if err != nil {
				log.Printf("Error obteniendo cliente de mensajería: %v", err)
			}
		}
	}

	if _, err := os.Stat("uploads"); os.IsNotExist(err) {
        os.Mkdir("uploads", 0755)
    }

	hub := api.NewHub()
	go hub.Run() // No olvides poner a correr el hub en segundo plano

	pollModel := models.NewPollModel(client)
	userModel := models.NewUserModel(client, db)
	deviceModel := models.NewDeviceModel(client)

	authModel := models.NewAuthModel(client, db)
	authAPI := api.NewAuthAPI(authModel, userModel)
	userAPI := api.NewUserAPI(userModel, pollModel, deviceModel, fcmClient, hub)

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./uploads"))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", fs))

	api.SetupRoutes(mux, userAPI, authAPI)

	handlerConMultipart := api.MultipartMiddleware(mux) // Asegúrate de que MultipartMiddleware sea exportable (nombre en mayúscula)

	port := ":8000"
	log.Printf("Servidor iniciado en http://localhost%s", port)
	log.Printf("Documentación disponible en http://localhost%s/docs", port)

	if err := http.ListenAndServe(port, handlerConMultipart); err != nil {
    log.Fatal(err)
}
}
