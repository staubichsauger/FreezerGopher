package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"strconv"
	"time"
)

type Perishable struct {
	gorm.Model
	Type     PerishableType
	Date     time.Time
	Count    int
	Location string
}

type PerishableType struct {
	gorm.Model
	Name           string
	IsFresh        bool
	AdditionalTime time.Duration
}

var (
	tpl *template.Template
	db  *gorm.DB
)

func init() {
	var err error
	tpl = template.Must(template.ParseGlob("static/templates/*.gohtml"))
	db, err = gorm.Open("sqlite3", "food.db")
	if err != nil {
		logrus.Panicf("Unable to open db: %v", err)
	}
}

func main() {
	defer db.Close()

	db.AutoMigrate(&PerishableType{})
	db.AutoMigrate(&Perishable{})

	vendor := http.FileServer(http.Dir("static/vendor/"))
	http.Handle("/vendor/", http.StripPrefix("/vendor/", vendor))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/addType", addTypeHandler)

	http.ListenAndServe(":8181", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var perishableTypes []PerishableType
	//db.Lock()
	db.Find(&perishableTypes)
	//db.Unlock()

	tpl.ExecuteTemplate(w, "index.gohtml", perishableTypes)
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "add.gohtml", nil)
}

func addTypeHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	isFresh, _ := strconv.ParseBool(r.FormValue("isFresh"))
	addedTimeInt, err := strconv.Atoi(r.FormValue("addedTime"))
	if err != nil {
		logrus.Errorf("Error parsing addedTimeInt: %v", err)
	}
	var additionalTime time.Duration
	switch r.FormValue("timeUnit") {
	case "d":
		additionalTime = time.Duration(addedTimeInt) * time.Hour * 24
	case "w":
		additionalTime = time.Duration(addedTimeInt) * time.Hour * 24 * 7
	case "m":
		additionalTime = time.Duration(addedTimeInt) * time.Hour * 24 * 30
	}

	t := PerishableType{
		Model:          gorm.Model{},
		Name:           r.FormValue("name"),
		IsFresh:        isFresh,
		AdditionalTime: additionalTime,
	}

	//db.Lock()
	db.Save(&t)
	//db.Unlock()

	http.Redirect(w, r, "/add", http.StatusFound)
}