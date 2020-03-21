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
	Type     PerishableType `gorm:"foreignkey:TypeID"`
	Date     time.Time
	Count    int
	Location string
}

type PerishablePost struct {
	Id 		 string
	Type     string
	Date     string
	Count    string
	Location string
}

type PerishableType struct {
	gorm.Model
	Name           string
	IsFresh        bool
	AdditionalTime int
	TimeUnit       string
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
	http.HandleFunc("/addPerish", addPerishableHandler)
	http.HandleFunc("/addTypePost", addTypePostHandler)
	http.HandleFunc("/addPerishPost", addPerishablePostHandler)
	http.HandleFunc("/manageType", manageTypeHandler)

	http.ListenAndServe(":8181", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var perishables []Perishable
	var out []PerishablePost
	//db.Lock()
	db.Find(&perishables)

	for _, p := range perishables {
		logrus.Info(p.Type.Name)
		out = append(out, PerishablePost{
			Id:				strconv.Itoa(int(p.ID)),
			Type:			p.Type.Name,
			Date:			p.Date.String(),
			Count:			strconv.Itoa(p.Count),
			Location:		p.Location,
		})
	}

	//db.Unlock()
	tpl.ExecuteTemplate(w, "index.gohtml", out)
}

func manageTypeHandler(w http.ResponseWriter, r *http.Request) {
	var perishableTypes []PerishableType
	var out []PerishableTypePost
	//db.Lock()
	db.Find(&perishableTypes)
	for _, pt := range perishableTypes {
		out = append(out, PerishableTypePost{
			Name:           pt.Name,
			IsFresh:        strconv.FormatBool(pt.IsFresh),
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

func addPerishableHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var types []PerishableType
	db.Find(&types)

	out := struct {
		Perishable PerishablePost
		Types 		[]string
	}{}

	for _, t := range types {
		out.Types = append(out.Types, t.Name)
	}

	var perish Perishable
	if r.FormValue("id") != "" {
		db.Where("id = ?", r.FormValue("id")).First(&perish)
		out.Perishable = PerishablePost{
			Id:       strconv.Itoa(int(perish.ID)),
			Type:     perish.Type.Name,
			Date:     perish.Date.String(),
			Count:    strconv.Itoa(perish.Count),
			Location: perish.Location,
		}
	} else {
		out.Perishable = PerishablePost{
			Id: 	  "-1",
		}
	}

	tpl.ExecuteTemplate(w, "addPerishable.gohtml", out)
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
	if db.Where("name = ?", r.FormValue("name")).First(&t).RecordNotFound() {
		t = PerishableType{
			Model:          gorm.Model{},
			Name:           r.FormValue("name"),
			IsFresh:        isFresh,
			AdditionalTime: addedTimeInt,
			TimeUnit:       timeUnit,
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

func addPerishablePostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	date, _ := time.Parse("", r.FormValue("date"))
	count, _ := strconv.Atoi(r.FormValue("count"))
	// submit id via submit button value
	var p Perishable
	var t PerishableType
	if db.Where("id = ?", r.FormValue("submit")).First(&p).RecordNotFound() {
		db.Where("name = ?", r.FormValue("type")).First(&t)
		p = Perishable{
			Model:    gorm.Model{},
			Type:     t,
			Date:     date,
			Count:    count,
			Location: r.FormValue("location"),
		}
	} else {
		p.Date = date
		p.Count = count
		p.Location = r.FormValue("location")
	}
	db.Save(&p)

	if r.FormValue("submit") == "add" {
		http.Redirect(w, r, "/addPerish", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}