package app

import (
	"../database"
	"../pages"
	//"../tasks"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var templates = template.Must(template.ParseFiles("./templates/index.html", "./templates/status.html"))
var wg sync.WaitGroup
var mu sync.RWMutex

type Task struct {
	Name  		string
	Status		string
	Messages 	chan string
	Updates		[]string
}

func (t *Task) updateStatus(status string) {
	t.Status = status
}

func (t *Task) Run(name string, application *App) {
	defer wg.Done()
	mu.RLock()
	t.Name = name
	time.Sleep(time.Second * 3)
	t.updateStatus("Running!")
	t.Updates = append(t.Updates, fmt.Sprintf("%s: Checking if %s exists in the database...", time.Now().Format(time.RFC3339), name))
	//t.Messages <- fmt.Sprintf("Checking if %s exists in the database...", name)
	time.Sleep(time.Second * 3)
	record := application.Database.Where("name = ?", name).First(&database.Person{})
	if !record.RecordNotFound() {
		//t.Messages <- fmt.Sprintf("%s already exists in the database!", name)
		t.Updates = append(t.Updates, fmt.Sprintf("%s: %s already exists in the database!!!", time.Now().Format(time.RFC3339), name))
		t.updateStatus("Complete!")
	} else {
		//t.Messages <- "Doesn't exist! Proceeding with record creation..."
		t.Updates = append(t.Updates, fmt.Sprintf("%s: %s doesn't exist! Proceeding with record creation...", time.Now().Format(time.RFC3339), name))
		time.Sleep(time.Second * 3)
		//t.Messages <- "Hmm... Gonna take my time with this one..."
		t.Updates = append(t.Updates, fmt.Sprintf("%s: Hmm... %s?? Gonna take my time with this one...", time.Now().Format(time.RFC3339), name))
		time.Sleep(time.Second * 3)
		person := database.Person{Name: name, Timestamp: time.Now(), UID: uuid.New().String()}
		//t.Messages <- "Ok... almost done..."
		t.Updates = append(t.Updates, fmt.Sprintf("%s: Ok, %s. Almost done...", time.Now().Format(time.RFC3339), name))
		time.Sleep(time.Second * 3)
		//t.Messages <- fmt.Sprintf("Adding %s to the database...", name)
		t.Updates = append(t.Updates, fmt.Sprintf("%s: Adding %s to the database...", time.Now().Format(time.RFC3339), name))
		application.Database.Create(&person)
		time.Sleep(time.Second * 3)
		//t.Messages <- fmt.Sprintf("Done! %s is all set.", name)
		t.Updates = append(t.Updates, fmt.Sprintf("%s: Congratulations, %s! We're done...", time.Now().Format(time.RFC3339), name))
		t.updateStatus("Complete!")
	}
	mu.RUnlock()
	//close(t.Messages)
}

type App struct {
	Database 	*gorm.DB
	Router		*mux.Router
	Pages		map[string]*pages.Page
	Tasks		map[string]*Task
}

func (a *App) InitApp() {
	// create router
	a.Router = mux.NewRouter()
	// add routes
	a.Router.HandleFunc("/", a.Index).Methods("GET")
	a.Router.HandleFunc("/add", a.Add).Methods("POST")
	a.Router.HandleFunc("/background", a.Background).Methods("POST")
	a.Router.HandleFunc("/status/{name}", a.Status).Methods("GET", "POST")
	log.Print("Router Initialized!")

	// add pages
	a.Pages = make(map[string]*pages.Page)
	a.Pages["index"] = &pages.Page{Name:"index", Filename:"index.html", Data:make(map[string]interface{})}
	a.Pages["status"] = &pages.Page{Name:"status", Filename:"status.html", Data:make(map[string]interface{})}
	log.Print("Added Page Templates!")

	// add tasks
	a.Tasks = make(map[string]*Task)

	// create database connection
	var err error
	connUri := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("POSTGRES_PASSWORD"))
	a.Database, err = gorm.Open("postgres", connUri)
	if err != nil {
		log.Fatalf("Error! Couldn't connect to database: %s", err.Error())
	}
	// migrate the schema
	a.Database.AutoMigrate(&database.Person{})
	// log message
	log.Print("Database Initialized!")
}

func (a *App) Index(w http.ResponseWriter, r *http.Request) {
	people := []database.Person{}
	a.Database.Find(&people)
	a.Pages["index"].Data["table"] = people
	// render template
	renderPage(w, a.Pages["index"])
	delete(a.Pages["index"].Data, "alert")
}

func (a *App) Add(w http.ResponseWriter, r *http.Request) {
	// parse form values
	_ = r.ParseForm()
	name := r.FormValue("name")
	// check if name exists
	record := a.Database.Where("name = ?", name).First(&database.Person{})
	if !record.RecordNotFound() {
		// if exists redirect with danger alert
		a.Pages["index"].Data["alert"] = [2]string{fmt.Sprintf("%s already exists in the database!", name), "danger"}
	} else {
		// create new record redirect with success alert
		person := database.Person{
			Name:      name,
			Timestamp: time.Now(),
			UID:       uuid.New().String()}
		a.Database.Create(&person)
		a.Pages["index"].Data["alert"] = [2]string{fmt.Sprintf("%s was added to the database!", name), "success"}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) Background(w http.ResponseWriter, r *http.Request) {
	// parse name from form
	_ = r.ParseForm()
	name := r.FormValue("name")
	// check if there's already a live Task for this name
	_, exists := a.Tasks[name]
	// already a task - redirect to status page for name
	if exists {
		http.Redirect(w, r, fmt.Sprintf("/status/%s", name), http.StatusSeeOther)
	// no task; create one
	} else {
		newTask := Task{Status: "Starting!", Messages: make(chan string, 5)}
		a.Tasks[name] = &newTask
		wg.Add(1)
		// run long running task in goroutine
		go newTask.Run(name, a)
		http.Redirect(w, r, fmt.Sprintf("/status/%s", name), http.StatusSeeOther)
	}
}

func (a *App) Status(w http.ResponseWriter, r *http.Request) {
	// clean up a finished task
	if r.Method == "POST" {
		_ = r.ParseForm()
		name := r.FormValue("name")
		_, exists := a.Tasks[name]
		if exists {
			log.Printf("Deleting task: %s", name)
			delete(a.Tasks, name)
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	// get a status update
	} else if r.Method == "GET" {
		name := mux.Vars(r)["name"]
		_, exists := a.Tasks[name]
		// task doesn't exist - redirect; user needs to use the yellow button
		if !exists {
			a.Pages["index"].Data["alert"] = [2]string{fmt.Sprintf("Need to do a long submit for %s first", name), "warning"}
			http.Redirect(w, r, "/", http.StatusSeeOther)
		} else {
			a.Pages["status"].Data["task"] = a.Tasks[name]
			renderPage(w, a.Pages["status"])
		}
	}
}

func renderPage(w http.ResponseWriter, page *pages.Page) {
	// render page template - page contains the data to be used when rendering
	err := templates.ExecuteTemplate(w, page.Filename, page)
	if err != nil {
		log.Fatalf("Couldn't render %s page! Error: %s", page, err.Error())
	}
}
