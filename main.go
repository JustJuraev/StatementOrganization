package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
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

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	router := mux.NewRouter()
	http.Handle("/", router)
	router.HandleFunc("/", Login)
	router.HandleFunc("/login_check", LoginPost)
	router.HandleFunc("/statements/{id:[0-9]+}", Statements)
	http.ListenAndServe(":8084", nil)
}
