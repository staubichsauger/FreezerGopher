package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"time"
)

type Perishable struct {
	gorm.Model
	Type     *PerishableType //`gorm:"foreignkey:perishable_types_id"`
	TypeId   int			`sql:"type:integer REFERENCES perishable_types(id)"`
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

	db.Exec("PRAGMA foreign_keys = ON")
	//db.LogMode(true)
	/*db.Exec(`CREATE TABLE IF NOT EXISTS "perishable_types" ("id" integer primary key autoincrement,"created_at" datetime,"updated_at" datetime,"deleted_at" datetime,"name" varchar(255),"is_fresh" bool,"additional_time" integer,"time_unit" varchar(255) );
				 CREATE INDEX idx_perishable_types_deleted_at ON "perishable_types"(deleted_at) ;
				 CREATE TABLE IF NOT EXISTS "perishables" ("id" integer primary key autoincrement,"created_at" datetime,"updated_at" datetime,"deleted_at" datetime,"date" datetime,"count" integer,"location" varchar(255),perishable_types_id integer NOT NULL,FOREIGN KEY(perishable_types_id) REFERENCES perishable_types(id));
				 CREATE INDEX idx_perishables_deleted_at ON "perishables"(deleted_at) ;`)*/
	db.AutoMigrate(&PerishableType{}, &Perishable{})

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

	sort.Slice(perishables, func(i, j int) bool {
		return perishables[i].Date.Before(perishables[j].Date)
	})

	for _, p := range perishables {
		var t PerishableType
		db.Where("id = ?", p.TypeId).First(&t)
		p.Type = &t
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
		var t PerishableType
		db.Where("id = ?", perish.TypeId).First(&t)
		perish.Type = &t
		out.Perishable = PerishablePost{
			Id:       strconv.Itoa(int(perish.ID)),
			Type:     perish.Type.Name,
			Date:     perish.Date.Format("2006-01-02"),
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
	logrus.Info(r.FormValue("date"))
	date, _ := time.Parse("2006-01-02", r.FormValue("date"))
	count, _ := strconv.Atoi(r.FormValue("count"))
	// submit id via submit button value
	var p Perishable
	var t PerishableType
	if db.Where("id = ?", r.FormValue("submit")).First(&p).RecordNotFound() {
		db.Where("name = ?", r.FormValue("type")).First(&t)
		p = Perishable{
			Model:    gorm.Model{},
			Type:     &t,
			TypeId:   int(t.ID),
			Date:     date,
			Count:    count,
			Location: r.FormValue("location"),
		}
	} else {
		p.Date = date
		p.Count = count
		p.Location = r.FormValue("location")
	}
	if p.Count > 0 {
		db.Save(&p)
	} else {
		db.Delete(&p)
	}

	if r.FormValue("submit") == "add" {
		http.Redirect(w, r, "/addPerish", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}