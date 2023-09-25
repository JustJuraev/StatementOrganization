package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
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

func Login(page http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("html_files/login.html")
	if err != nil {
		panic(err)
	}
	tmpl.ExecuteTemplate(page, "login", nil)
}
func LoginPost(page http.ResponseWriter, r *http.Request) {
	login := r.FormValue("login")
	password := r.FormValue("password")
	if login == "" || password == "" {
		tmpl, err := template.ParseFiles("html_files/login.html")
		if err != nil {
			panic(err)
		}
		tmpl.ExecuteTemplate(page, "login", "Все поля должны быть заполнеными")
	}
	connStr := "user=postgres password=123456 dbname=mygovdb sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	hash := md5.Sum([]byte(password))
	hashedPass := hex.EncodeToString(hash[:])
	res := db.QueryRow("SELECT * FROM public.users WHERE login = $1 AND password = $2", login, hashedPass)
	user := User{}
	err3 := res.Scan(&user.Id, &user.Login, &user.Password, &user.Name, &user.LastName, &user.Role, &user.OrgId)
	if err3 != nil {
		tmpl, err2 := template.ParseFiles("html_files/login.html")
		if err2 != nil {
			panic(err2)
		}
		tmpl.ExecuteTemplate(page, "login", "Неправильный логин или пароль")
	} else {

		if user.Role == 2 {
			s2 := strconv.Itoa(user.OrgId)
			http.Redirect(page, r, "/statements/"+s2, http.StatusSeeOther)
		}
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

	_, err4 := db.Exec("INSERT INTO public.statementshistory (userid, statementid, date, action) VALUES ($1, $2, $3, $4)", uid, sid, time.Now(), "close")

	if err4 != nil {
		panic(err4)
	}

	_, err5 := db.Exec("UPDATE public.statements SET status=$1 WHERE id=$2", 200, sid)
	if err5 != nil {
		panic(err5)
	}

	http.Redirect(page, r, "/statements/"+orgid, http.StatusSeeOther)
}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	router := mux.NewRouter()
	http.Handle("/", router)
	router.HandleFunc("/", Login)
	router.HandleFunc("/login_check", LoginPost)
	router.HandleFunc("/statements/{id:[0-9]+}", Statements)
	router.HandleFunc("/closest/{id:[0-9]+}", CloseStatement)
	router.HandleFunc("/closing_statement", CloseStatementPost)
	http.ListenAndServe(":8084", nil)
}
