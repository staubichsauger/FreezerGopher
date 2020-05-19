package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Perishable struct {
	gorm.Model
	Type     *PerishableType //`gorm:"foreignkey:perishable_types_id"`
	TypeId   int             `sql:"type:integer REFERENCES perishable_types(id)"`
	Date     time.Time
	Count    int
	Location string
	Comment  string
}

type PerishablePost struct {
	Id       string
	Type     string
	Date     string
	OrigDate string
	Count    string
	Location string
	Comment  string
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
	tpl       *template.Template
	db        *gorm.DB
	urlPrefix string
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

	var found bool
	urlPrefix, found = os.LookupEnv("FREEZER_PREFIX")
	if found && (urlPrefix != "") {
		if !strings.HasPrefix(urlPrefix, "/") {
			urlPrefix = "/" + urlPrefix
		}
		if strings.HasSuffix(urlPrefix, "/") {
			urlPrefix = strings.TrimSuffix(urlPrefix, "/")
		}
	} else {
		urlPrefix = ""
	}

	logrus.Infof("Starting with urlPrefix: %v", urlPrefix)

	db.Exec("PRAGMA foreign_keys = ON")
	//db.LogMode(true)
	/*db.Exec(`CREATE TABLE IF NOT EXISTS "perishable_types" ("id" integer primary key autoincrement,"created_at" datetime,"updated_at" datetime,"deleted_at" datetime,"name" varchar(255),"is_fresh" bool,"additional_time" integer,"time_unit" varchar(255) );
	CREATE INDEX idx_perishable_types_deleted_at ON "perishable_types"(deleted_at) ;
	CREATE TABLE IF NOT EXISTS "perishables" ("id" integer primary key autoincrement,"created_at" datetime,"updated_at" datetime,"deleted_at" datetime,"date" datetime,"count" integer,"location" varchar(255),perishable_types_id integer NOT NULL,FOREIGN KEY(perishable_types_id) REFERENCES perishable_types(id));
	CREATE INDEX idx_perishables_deleted_at ON "perishables"(deleted_at) ;`)*/
	db.AutoMigrate(&PerishableType{}, &Perishable{})

	vendor := http.FileServer(http.Dir("static/vendor/"))
	http.Handle(urlPrefix+"/vendor/", http.StripPrefix(urlPrefix+"/vendor/", vendor))

	http.HandleFunc(urlPrefix+"/", indexHandler)
	http.HandleFunc(urlPrefix+"/addType", addTypeHandler)
	http.HandleFunc(urlPrefix+"/addPerish", addPerishableHandler)
	http.HandleFunc(urlPrefix+"/addTypePost", addTypePostHandler)
	http.HandleFunc(urlPrefix+"/addPerishPost", addPerishablePostHandler)
	http.HandleFunc(urlPrefix+"/manageType", manageTypeHandler)

	http.ListenAndServe(":8181", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	type PerishableDisplay struct {
		Perish   Perishable
		OrigDate time.Time
	}
	var perishables []Perishable
	out := struct {
		P         []PerishablePost
		UrlPrefix string
	}{
		UrlPrefix: urlPrefix,
	}

	//db.Lock()
	db.Find(&perishables)
	var perishablesDisplay []PerishableDisplay
	//fmt.Printf("Perishables retrieved: %v", perishables)

	for idx, p := range perishables {
		var t PerishableType
		db.Where("id = ?", p.TypeId).First(&t)
		perishables[idx].Type = &t
		var addedTime time.Duration
		switch t.TimeUnit {
		case "days":
			addedTime = time.Hour * 24 * time.Duration(t.AdditionalTime)
		case "weeks":
			addedTime = time.Hour * 24 * time.Duration(t.AdditionalTime) * 7
		case "months":
			addedTime = time.Hour * 24 * time.Duration(t.AdditionalTime) * 30
		}
		var pt PerishableDisplay
		pt.OrigDate = p.Date
		perishables[idx].Date = p.Date.Add(addedTime)
		pt.Perish = perishables[idx]
		perishablesDisplay = append(perishablesDisplay, pt)
	}

	sort.Slice(perishablesDisplay, func(i, j int) bool {
		return perishablesDisplay[i].Perish.Date.Before(perishablesDisplay[j].Perish.Date)
	})

	for _, p := range perishablesDisplay {
		out.P = append(out.P, PerishablePost{
			Id:       strconv.Itoa(int(p.Perish.ID)),
			Type:     p.Perish.Type.Name,
			Date:     p.Perish.Date.Format("2006-01-02"),
			OrigDate: p.OrigDate.Format("2006-01-02"),
			Count:    strconv.Itoa(p.Perish.Count),
			Location: p.Perish.Location,
			Comment:  p.Perish.Comment,
		})
	}

	//db.Unlock()
	tpl.ExecuteTemplate(w, "index.gohtml", out)
}

func manageTypeHandler(w http.ResponseWriter, r *http.Request) {
	var perishableTypes []PerishableType
	out := struct {
		P         []PerishableTypePost
		UrlPrefix string
	}{
		UrlPrefix: urlPrefix,
	}
	//db.Lock()
	db.Find(&perishableTypes)
	for _, pt := range perishableTypes {
		out.P = append(out.P, PerishableTypePost{
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
	out := struct {
		P         PerishableTypePost
		UrlPrefix string
	}{
		P: PerishableTypePost{
			Name:           r.FormValue("name"),
			IsFresh:        r.FormValue("isFresh"),
			AdditionalTime: r.FormValue("addedTime"),
			TimeUnit:       r.FormValue("timeUnit"),
		},
		UrlPrefix: urlPrefix,
	}
	tpl.ExecuteTemplate(w, "add.gohtml", out)
}

func addPerishableHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var types []PerishableType
	db.Find(&types)

	out := struct {
		Perishable PerishablePost
		Types      []string
		UrlPrefix  string
	}{
		UrlPrefix: urlPrefix,
	}

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
			Comment:  perish.Comment,
		}
	} else {
		out.Perishable = PerishablePost{
			Id: "-1",
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
		http.Redirect(w, r, urlPrefix+"/addType", http.StatusFound)
		return
	}
	http.Redirect(w, r, urlPrefix+"/manageType", http.StatusFound)
}

func addPerishablePostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	date, _ := time.Parse("2006-01-02", r.FormValue("date"))
	count, _ := strconv.Atoi(r.FormValue("count"))
	// submit id via submit button value
	var p Perishable
	var t PerishableType
	if db.Where("id = ?", r.FormValue("submit")).First(&p).RowsAffected == 0 {
		db.Where("name = ?", r.FormValue("type")).First(&t)
		p = Perishable{
			Model:    gorm.Model{},
			Type:     &t,
			TypeId:   int(t.ID),
			Date:     date,
			Count:    count,
			Location: r.FormValue("location"),
			Comment:  r.FormValue("comment"),
		}
		db.Save(&p)
	} else {
		p.Count = count
		p.Location = r.FormValue("location")
		p.Comment = r.FormValue("comment")
		db.Save(&p)
	}
	//fmt.Println("Perishable: %v", p)
	if p.Count <= 0 {
		db.Delete(&p)
	}

	if r.FormValue("submit") == "add" {
		http.Redirect(w, r, urlPrefix+"/addPerish", http.StatusFound)
		return
	}
	http.Redirect(w, r, urlPrefix+"/", http.StatusFound)
}
