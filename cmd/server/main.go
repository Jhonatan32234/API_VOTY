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

	"pruebas_doc/ent"
	"pruebas_doc/internal/api"
	"pruebas_doc/internal/models"
)

func main() {
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
		dbUser = "root"
	}
	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		dbPass = "pass"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "test_db"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbUser, dbPass, dbHost, dbPort, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error abriendo conexión SQL: %v", err)
	}

	drv := entsql.OpenDB(dialect.MySQL, db)
	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("Error en migraciones: %v", err)
	}

	userModel := models.NewUserModel(client, db)
	authModel := models.NewAuthModel(client, db)

	userAPI := api.NewUserAPI(userModel)
	authAPI := api.NewAuthAPI(authModel)

	mux := http.NewServeMux()

	api.SetupRoutes(mux, userAPI, authAPI)

	port := ":8000"
	log.Printf("Servidor iniciado en http://localhost%s", port)
	log.Printf("Documentación disponible en http://localhost%s/docs", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatal(err)
	}
}
