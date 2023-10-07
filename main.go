package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type User struct {
	Id       int    `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`
	Token    string `json:"token"`
	Name     string `json:"name"`
	LastName string `json:"lastname"`
	Role     int    `json:"role"`
	OrgId    int    `json:"orgid"`
}

type StatementStruct struct {
	Id             int
	Name           string
	LastName       string
	Date           string
	Status         int
	Statement      string
	PassportSeries string
	Time           time.Time
	UserId         int
	OrgId          int
}

type OrgStatement struct {
	Id          int
	StatementId int
	Text        string
	File        string
}

var statements = []StatementStruct{}
var users = []User{}

func Login(page http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("html_files/login.html")
	if err != nil {
		panic(err)
	}
	tmpl.ExecuteTemplate(page, "login", nil)
}

func HandleOAuthRequest(page http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	deserializedUser := User{}
	err = json.Unmarshal([]byte(string(b)), &deserializedUser)

	users = append(users, deserializedUser)
	if users[0].Role == 2 {
		s2 := strconv.Itoa(users[0].OrgId)
		http.Redirect(page, r, "/statements/"+s2, http.StatusSeeOther)
	} else {
		http.Redirect(page, r, "/", http.StatusSeeOther)
	}

}

func Statements(page http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	connStr := "user=postgres password=123456 dbname=mygovdb sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	row, err2 := db.Query("SELECT * FROM public.statements WHERE orgid=$1", id)

	if err2 != nil {
		panic(err2)
	}

	defer row.Close()

	statements := []StatementStruct{}
	for row.Next() {
		st := StatementStruct{}
		err3 := row.Scan(&st.Id, &st.Name, &st.LastName, &st.Date, &st.Status, &st.Statement, &st.PassportSeries, &st.Time, &st.UserId, &st.OrgId)
		if err3 != nil {
			fmt.Println(err3)
		}
		statements = append(statements, st)
	}

	tmpl, err := template.ParseFiles("html_files/statements.html")
	if err != nil {
		panic(err)
	}
	tmpl.ExecuteTemplate(page, "statements", statements)
}

func CloseStatement(page http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]

	connStr := "user=postgres password=123456 dbname=mygovdb sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	row := db.QueryRow("SELECT * FROM public.statements WHERE id=$1", id)
	st := StatementStruct{}
	err2 := row.Scan(&st.Id, &st.Name, &st.LastName, &st.Date, &st.Status, &st.Statement, &st.PassportSeries, &st.Time, &st.UserId, &st.OrgId)
	if err2 != nil {
		panic(err2)
	}

	tmpl, err := template.ParseFiles("html_files/closeform.html")
	if err != nil {
		panic(err)
	}
	tmpl.ExecuteTemplate(page, "closeform", st)
}

func CloseStatementPost(page http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	sid := r.FormValue("sid")
	orgid := r.FormValue("orgid")
	text := r.FormValue("text")

	file, handler, err := r.FormFile("myFile")
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	dst, _ := os.Create(filepath.Join("temp_files", handler.Filename))
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(page, err.Error(), http.StatusInternalServerError)
		return
	}

	connStr := "user=postgres password=123456 dbname=mygovdb sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	_, err3 := db.Exec("INSERT INTO public.orgstatements (statementid, text, file) VALUES ($1, $2, $3)", sid, text, handler.Filename)

	if err3 != nil {
		panic(err3)
	}

	_, err4 := db.Exec("INSERT INTO public.statementshistory (userid, statementid, date, action, orgid) VALUES ($1, $2, $3, $4, $5)", uid, sid, time.Now(), "close", orgid)

	if err4 != nil {
		panic(err4)
	}

	_, err5 := db.Exec("UPDATE public.statements SET status=$1 WHERE id=$2", 200, sid)
	if err5 != nil {
		panic(err5)
	}

	http.Redirect(page, r, "/statements/"+orgid, http.StatusSeeOther)
}

func RejectStatement(page http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]

	connStr := "user=postgres password=123456 dbname=mygovdb sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	row := db.QueryRow("SELECT * FROM public.statements WHERE id=$1", id)
	st := StatementStruct{}
	err2 := row.Scan(&st.Id, &st.Name, &st.LastName, &st.Date, &st.Status, &st.Statement, &st.PassportSeries, &st.Time, &st.UserId, &st.OrgId)
	if err2 != nil {
		panic(err2)
	}

	tmpl, err := template.ParseFiles("html_files/rejectform.html")
	if err != nil {
		panic(err)
	}
	tmpl.ExecuteTemplate(page, "rejectform", st)
}

func RejectStatementPost(page http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	sid := r.FormValue("sid")
	orgid := r.FormValue("orgid")
	text := r.FormValue("text")

	file, handler, err := r.FormFile("myFile")
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	dst, _ := os.Create(filepath.Join("temp_files", handler.Filename))
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(page, err.Error(), http.StatusInternalServerError)
		return
	}

	connStr := "user=postgres password=123456 dbname=mygovdb sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	_, err3 := db.Exec("INSERT INTO public.orgstatements (statementid, text, file) VALUES ($1, $2, $3)", sid, text, handler.Filename)

	if err3 != nil {
		panic(err3)
	}

	_, err4 := db.Exec("INSERT INTO public.statementshistory (userid, statementid, date, action, orgid) VALUES ($1, $2, $3, $4, $5)", uid, sid, time.Now(), "reject from org", orgid)

	if err4 != nil {
		panic(err4)
	}

	_, err5 := db.Exec("UPDATE public.statements SET status=$1 WHERE id=$2", 220, sid)
	if err5 != nil {
		panic(err5)
	}

	http.Redirect(page, r, "/statements/"+orgid, http.StatusSeeOther)
}

func SendBackStatement(page http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]

	connStr := "user=postgres password=123456 dbname=mygovdb sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	row := db.QueryRow("SELECT * FROM public.statements WHERE id=$1", id)
	st := StatementStruct{}
	err2 := row.Scan(&st.Id, &st.Name, &st.LastName, &st.Date, &st.Status, &st.Statement, &st.PassportSeries, &st.Time, &st.UserId, &st.OrgId)
	if err2 != nil {
		panic(err2)
	}

	tmpl, err := template.ParseFiles("html_files/sendbackst.html")
	if err != nil {
		panic(err)
	}
	tmpl.ExecuteTemplate(page, "sendbackst", st)
}

func SendBackStatementPost(page http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	sid := r.FormValue("sid")
	orgid := r.FormValue("orgid")
	text := r.FormValue("text")

	connStr := "user=postgres password=123456 dbname=mygovdb sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	_, err3 := db.Exec("INSERT INTO public.orgstatements (statementid, text) VALUES ($1, $2)", sid, text)

	if err3 != nil {
		panic(err3)
	}

	_, err4 := db.Exec("INSERT INTO public.statementshistory (userid, statementid, date, action, orgid) VALUES ($1, $2, $3, $4, $5)", uid, sid, time.Now(), "send back from org", orgid)

	if err4 != nil {
		panic(err4)
	}

	_, err5 := db.Exec("UPDATE public.statements SET status=$1 WHERE id=$2", 250, sid)
	if err5 != nil {
		panic(err5)
	}

	http.Redirect(page, r, "/statements/"+orgid, http.StatusSeeOther)
}

//TODO: 1. Добавить OrgId в StatementHistory, 2. Добавить историю в заявлениях, 3. добавить раздел завершеные заявления

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	router := mux.NewRouter()
	http.Handle("/", router)
	router.HandleFunc("/", Login)
	router.HandleFunc("/handleouathcheck", HandleOAuthRequest)
	router.HandleFunc("/statements/{id:[0-9]+}", Statements)
	router.HandleFunc("/closest/{id:[0-9]+}", CloseStatement)
	router.HandleFunc("/closing_statement", CloseStatementPost)
	router.HandleFunc("/rejectst/{id:[0-9]+}", RejectStatement)
	router.HandleFunc("/rejecting_statement", RejectStatementPost)
	router.HandleFunc("/sendbackst/{id:[0-9]+}", SendBackStatement)
	router.HandleFunc("/sendbackst_statement", SendBackStatementPost)
	http.ListenAndServe(":8084", nil)
}
