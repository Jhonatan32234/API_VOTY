package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"

	"api_voty/ent"
	"api_voty/ent/migrate"
	"api_voty/internal/api"
	"api_voty/internal/models"
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

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error abriendo conexión SQL: %v", err)
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

	hub := api.NewHub()
	go hub.Run() // No olvides poner a correr el hub en segundo plano

	pollModel := models.NewPollModel(client)
	userModel := models.NewUserModel(client, db)

	authModel := models.NewAuthModel(client, db)
	authAPI := api.NewAuthAPI(authModel)
	userAPI := api.NewUserAPI(userModel, pollModel, hub)

	mux := http.NewServeMux()

	api.SetupRoutes(mux, userAPI, authAPI)

	port := ":8000"
	log.Printf("Servidor iniciado en http://localhost%s", port)
	log.Printf("Documentación disponible en http://localhost%s/docs", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatal(err)
	}
}
