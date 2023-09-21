package main

import (
	"fmt"
	"github.com/maxGitham77/vueapi/internal/data"
	"github.com/maxGitham77/vueapi/internal/driver"
	"log"
	"net/http"
	"os"
)

type config struct {
	port int
}

// application is the type for all data we want to share with the various part of the application.
// We will share this information in most cases by using this type as the receiver for functions
type application struct {
	config      config
	infoLog     *log.Logger
	errorLog    *log.Logger
	models      data.Models
	environment string
}

// main is the main entry point for the application
func main() {
	var cfg config
	cfg.port = 8081

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	/*dsn := "host=localhost port=5432 user=postgres password=password dbname=vueapi sslmode=disable timezone=UTC connect_timeout=5"*/
	dsn := os.Getenv("DSN")
	environment := os.Getenv("ENV")
	db, err := driver.ConnectPostgres(dsn)
	if err != nil {
		log.Fatal("Cannot connect to database")
	}
	defer db.SQL.Close()

	app := &application{
		config:      cfg,
		infoLog:     infoLog,
		errorLog:    errorLog,
		models:      data.New(db.SQL),
		environment: environment,
	}

	err = app.serve()
	if err != nil {
		log.Fatal(err)
	}
}

// Serve starts the web server
func (app *application) serve() error {
	/*http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Okay    bool   `json:"okay"`
			Message string `json"message"`
		}
		payload.Okay = true
		payload.Message = "Hello, world"

		out, err := json.MarshalIndent(payload, "", "\t")
		if err != nil {
			app.errorLog.Println(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(out)
	}*/

	app.infoLog.Println("API listening on port", app.config.port)
	//return http.ListenAndServe(fmt.Sprintf(":%d", app.config.port), nil)

	serv := &http.Server{
		Addr:    fmt.Sprintf(":%d", app.config.port),
		Handler: app.routes(),
	}

	return serv.ListenAndServe()

}
