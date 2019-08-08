package main

import (
	"./database"
	"./pages"
	"flag"
	"fmt"
	guuid "github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"golang.org/x/net/context"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var templates = template.Must(template.ParseFiles("./templates/index.html"))

type App struct {
	database 	*gorm.DB
	router 		*mux.Router
	pages    	map[string]*pages.Page
	tasks		map[string][]string
}

func (a *App) InitPages() {
	// add pages
	a.pages = make(map[string]*pages.Page)
	var empty = make(map[string]interface{})
	a.pages["index"] = &pages.Page{Title: "Home", Filename: "index.html", Data: empty}
	a.pages["status"] = &pages.Page{Title: "Status", Filename: "status.html", Data: empty}
}

func (a *App) InitRouter() {
	a.router = mux.NewRouter()
	// add routes
	a.router.HandleFunc("/", a.Index).Methods("GET")
	a.router.HandleFunc("/add", a.Add).Methods("POST")
	//a.router.HandleFunc("/background", a.Background).Methods("POST", "GET")
}

func (a *App) InitDatabase() {
	var err error
	connUri := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("POSTGRES_PASSWORD"))
	// create connection
	a.database, err = gorm.Open("postgres", connUri)
	if err != nil {
		log.Fatalf("Error! Couldn't connect to database: %s", err.Error())
	}
	// migrate the schema
	a.database.AutoMigrate(&database.Person{})
	// log message
	log.Print("Database Initialized!")

}

func (a *App) Index(w http.ResponseWriter, r *http.Request) {
	people := []database.Person{}
	a.database.Find(&people)
	a.pages["index"].Data["table"] = &people
	renderPage(w, a.pages["index"])
	a.pages["index"].Alert = [2]string{}
}

func (a *App) Add(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	name := r.FormValue("name")
	record := a.database.Where("name = ?", name).First(&database.Person{})
	if !record.RecordNotFound() {
		a.pages["index"].Alert = [2]string{fmt.Sprintf("%s already exists in the database!", name), "danger"}
		renderPage(w, a.pages["index"])
		return
	}
	person := database.Person{
		Name: 		name,
		Timestamp: 	time.Now(),
		UID: 		guuid.New().String()}
	a.database.Create(&person)
	a.pages["index"].Alert = [2]string{fmt.Sprintf("%s was added to the database!", name), "success"}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

//func (a *App) Background(w http.ResponseWriter, r *http.Request) {
//	if r.Method == "POST" {
//		_ = r.ParseForm()
//		name := r.FormValue("name")
//		task, exists := a.tasks[name]
//		if exists {
//			http.Redirect(w, r, fmt.Sprintf("/%s", ), http.StatusSeeOther)
//		} else {
//
//		}
//		go func(){
//			time.Sleep()
//			a.Add(w, r)
//		}()
//	} else {
//
//	}
//	renderPage(w, a.pages["status"])
//}

func renderPage(w http.ResponseWriter, p *pages.Page) {
	err := templates.ExecuteTemplate(w, p.Filename, p)
	if err != nil {
		log.Fatalf("Couldn't render page! Error: %s", err.Error())
	}
}

func main() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second * 15,
		"the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	var app App
	app.InitDatabase()
	defer app.database.Close()
	app.InitRouter()
	app.InitPages()

	address := fmt.Sprintf("0.0.0.0:%s", os.Getenv("APP_PORT"))
	srv := &http.Server{
		Addr:         address,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler: app.router, // Pass our instance of gorilla/mux in.
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		log.Printf("Starting App on %s", address)
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)
}