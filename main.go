package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strconv"
	"text/template"

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

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	http.HandleFunc("/", Login)
	http.HandleFunc("/login_check", LoginPost)
	http.ListenAndServe(":8084", nil)
}
