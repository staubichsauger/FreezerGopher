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
	AdditionalTime int
	TimeUnit	   string
}

type PerishableTypeString struct {
	Name           string
	IsFresh        bool
	AdditionalTime string
	TimeUnit	   string
}

type PerishableTypePost struct {
	Name           string
	IsFresh        string
	AdditionalTime string
	TimeUnit       string
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
	http.HandleFunc("/addType", addTypeHandler)
	http.HandleFunc("/addTypePost", addTypePostHandler)
	http.HandleFunc("/manageType", manageTypeHandler)

	http.ListenAndServe(":8181", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var perishableTypes []PerishableType
	var out []PerishableTypeString
	//db.Lock()
	db.Find(&perishableTypes)
	for _, pt := range perishableTypes {
		out = append(out, PerishableTypeString{
			Name:           pt.Name,
			IsFresh:        pt.IsFresh,
			AdditionalTime: strconv.Itoa(pt.AdditionalTime),
			TimeUnit:       pt.TimeUnit,
		})
	}
	//db.Unlock()
	tpl.ExecuteTemplate(w, "index.gohtml", out)
}

func manageTypeHandler(w http.ResponseWriter, r *http.Request) {
	var perishableTypes []PerishableType
	var out []PerishableTypeString
	//db.Lock()
	db.Find(&perishableTypes)
	for _, pt := range perishableTypes {
		out = append(out, PerishableTypeString{
			Name:           pt.Name,
			IsFresh:        pt.IsFresh,
			AdditionalTime: strconv.Itoa(pt.AdditionalTime),
			TimeUnit:       pt.TimeUnit,
		})
	}
	//db.Unlock()
	tpl.ExecuteTemplate(w, "manage.gohtml", out)
}

func addTypeHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	out := PerishableTypePost{
		Name:           r.FormValue("name"),
		IsFresh:        r.FormValue("isFresh"),
		AdditionalTime: r.FormValue("addedTime"),
		TimeUnit:       r.FormValue("timeUnit"),
	}
	tpl.ExecuteTemplate(w, "add.gohtml", out)
}

func addTypePostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	isFresh, _ := strconv.ParseBool(r.FormValue("isFresh"))
	addedTimeInt, err := strconv.Atoi(r.FormValue("addedTime"))
	if err != nil {
		logrus.Errorf("Error parsing addedTimeInt: %v", err)
	}

	timeUnit := r.FormValue("timeUnit")

	var t PerishableType
	if  db.Where("name = ?", r.FormValue("name")).First(&t).RecordNotFound() {
		t = PerishableType{
			Model:          gorm.Model{},
			Name:           r.FormValue("name"),
			IsFresh:        isFresh,
			AdditionalTime: addedTimeInt,
			TimeUnit:	    timeUnit,
		}
	} else {
		t.IsFresh = isFresh
		t.AdditionalTime = addedTimeInt
		t.TimeUnit = timeUnit
	}
	db.Save(&t)

	if r.FormValue("submit") == "add" {
		http.Redirect(w, r, "/addType", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/manageType", http.StatusFound)
}