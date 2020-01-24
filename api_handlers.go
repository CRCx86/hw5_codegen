package main

import (
	"net/http"
	"strconv"
	"strings"
)
import "encoding/json"
import "errors"

var (
	errorUnknown    = errors.New("unknown method")
	errorBad        = errors.New("bad method")
	errorEmptyLogin = errors.New("login must me not empty")
)

type JsonError struct {
	Error string `json:"error"`
}

func (h *MyApi) profile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var login string
	switch r.Method {
	case "GET":
		login = r.URL.Query().Get("login")
		if login == "" {
			js, _ := json.Marshal(JsonError{errorEmptyLogin.Error()})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(js)
			return
		}
	case "POST":
		r.ParseForm()
		login = r.Form.Get("login")
		if login == "" {
			js, _ := json.Marshal(JsonError{errorEmptyLogin.Error()})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(js)
			return
		}
	}
	ProfileParams := ProfileParams{Login: login,}
	user, err := h.Profile(ctx, ProfileParams)
	if err != nil {
		switch err.(type) {
		case ApiError:
			js, _ := json.Marshal(JsonError{err.(ApiError).Err.Error()})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(err.(ApiError).HTTPStatus)
			w.Write(js)
			return
		default:
			js, _ := json.Marshal(JsonError{"bad user"})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(js)
			return
		}
	}
	js, _ := json.Marshal(ResponseMyApiProfile{user, JsonError{""}})
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (h *MyApi) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var login string
	var name string
	var status string
	var age int
	if r.Method != "POST" {
		js, _ := json.Marshal(JsonError{errorBad.Error()})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write(js)
		return
	}
	if r.Header.Get("X-Auth") != "100500" {
		js, _ := json.Marshal(JsonError{"unauthorized"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write(js)
		return
	}
	r.ParseForm()
	login = r.Form.Get("login")

	if login == "" {
		js, _ := json.Marshal(JsonError{errorEmptyLogin.Error()})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
	if !(len(login) >= 10) {
		js, _ := json.Marshal(JsonError{"login len must be >= 10"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
	name = r.Form.Get("name")

	paramname_name := r.Form.Get("full_name")
	if paramname_name == "" {
		name = strings.ToLower(name)
	} else {
		name = paramname_name
	}
	status = r.Form.Get("status")

	if status == "" {
		status = "user"
	}
	m := make(map[string]bool)
	m["user"] = true
	m["moderator"] = true
	m["admin"] = true
	_, prs := m[status]
	if prs == false {
		js, _ := json.Marshal(JsonError{"status must be one of [user, moderator, admin]"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
	age, err := strconv.Atoi(r.Form.Get("age"))
	if err != nil {
		js, _ := json.Marshal(JsonError{"age must be int"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}

	if !(age >= 0) {
		js, _ := json.Marshal(JsonError{"age must be >= 0"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
	if !(age <= 128) {
		js, _ := json.Marshal(JsonError{"age must be <= 128"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
	CreateParams := CreateParams{Age: age, Login: login, Name: name, Status: status,}
	newuser, err := h.Create(ctx, CreateParams)
	if err != nil {
		switch err.(type) {
		case ApiError:
			js, _ := json.Marshal(JsonError{err.(ApiError).Err.Error()})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(err.(ApiError).HTTPStatus)
			w.Write(js)
			return
		default:
			js, _ := json.Marshal(JsonError{"bad user"})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(js)
			return
		}
	}
	js, _ := json.Marshal(ResponseMyApiCreate{newuser, JsonError{""}})
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (h *OtherApi) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var username string
	var name string
	var class string
	var level int
	if r.Method != "POST" {
		js, _ := json.Marshal(JsonError{errorBad.Error()})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write(js)
		return
	}
	if r.Header.Get("X-Auth") != "100500" {
		js, _ := json.Marshal(JsonError{"unauthorized"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write(js)
		return
	}
	r.ParseForm()
	username = r.Form.Get("username")

	if username == "" {
		js, _ := json.Marshal(JsonError{errorEmptyLogin.Error()})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
	if !(len(username) >= 3) {
		js, _ := json.Marshal(JsonError{"username len must be >= 3"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
	name = r.Form.Get("name")

	paramname_name := r.Form.Get("account_name")
	if paramname_name == "" {
		name = strings.ToLower(name)
	} else {
		name = paramname_name
	}
	class = r.Form.Get("class")

	if class == "" {
		class = "warrior"
	}
	m := make(map[string]bool)
	m["warrior"] = true
	m["sorcerer"] = true
	m["rouge"] = true
	_, prs := m[class]
	if prs == false {
		js, _ := json.Marshal(JsonError{"class must be one of [warrior, sorcerer, rouge]"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
	level, err := strconv.Atoi(r.Form.Get("level"))
	if err != nil {
		js, _ := json.Marshal(JsonError{"level must be int"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}

	if !(level >= 1) {
		js, _ := json.Marshal(JsonError{"level must be >= 1"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
	if !(level <= 50) {
		js, _ := json.Marshal(JsonError{"level must be <= 50"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(js)
		return
	}
	OtherCreateParams := OtherCreateParams{Username: username, Name: name, Class: class, Level: level,}
	otheruser, err := h.Create(ctx, OtherCreateParams)
	if err != nil {
		switch err.(type) {
		case ApiError:
			js, _ := json.Marshal(JsonError{err.(ApiError).Err.Error()})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(err.(ApiError).HTTPStatus)
			w.Write(js)
			return
		default:
			js, _ := json.Marshal(JsonError{"bad user"})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(js)
			return
		}
	}
	js, _ := json.Marshal(ResponseOtherApiCreate{otheruser, JsonError{""}})
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (h *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		h.profile(w, r)
	case "/user/create":
		h.create(w, r)
	default:
		js, _ := json.Marshal(JsonError{errorUnknown.Error()})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write(js)
		return
	}
}
func (h *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/create":
		h.create(w, r)
	default:
		js, _ := json.Marshal(JsonError{errorUnknown.Error()})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write(js)
		return
	}
}

type ResponseMyApiProfile struct {
	*User `json:"response"`
	JsonError
}
type ResponseMyApiCreate struct {
	*NewUser `json:"response"`
	JsonError
}
type ResponseOtherApiCreate struct {
	*OtherUser `json:"response"`
	JsonError
}
