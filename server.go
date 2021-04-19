package main

import "math/rand"
import "os"
import "strings"
import "fmt"
import "time"
import "io/ioutil"
import "sync"
import "net/http"
import "encoding/json"

type Person struct {
  Name string `json:"name"`
  ID string `json:"id"`
  Age int `json:"hahe"`
}

type personHandlers struct {
  sync.Mutex
  store map[string]Person
}

func (h *personHandlers) persons(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
  case "GET":
    h.get(w,r)
    return
  case "POST":
    h.post(w,r)
    return
  default:
    w.WriteHeader(http.StatusMethodNotAllowed)
    w.Write([]byte("method not allowed"))
    return
  }
}

func (h *personHandlers) get(w http.ResponseWriter, r *http.Request) {
  persons := make([]Person, len(h.store))


  h.Lock()
  i:= 0
  for _, person := range h.store {
    persons[i] = person
    i++
  }
  h.Unlock()

  jsonBytes, err := json.Marshal(persons)
  if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write([]byte(err.Error()))
    return
  }

  w.Header().Add("content=type", "application/json")
  w.WriteHeader(http.StatusOK)
  w.Write(jsonBytes)
}
func (h *personHandlers) getRandomPerson(w http.ResponseWriter, r *http.Request) {
  ids := make([]string, len(h.store))
  h.Lock()

  i := 0
  for id := range h.store {
    ids[i] = id
    i++
  }
  defer h.Unlock()

  var target string
  if len(ids) == 0 {
    w.WriteHeader(http.StatusNotFound)
    return
  } else if len(ids) == 1 {
    target = ids[0]
  } else {
    rand.Seed(time.Now().UnixNano())
    target = ids[rand.Intn(len(ids))]

    w.Header().Add("location", fmt.Sprintf("/persons/%s", target))
    w.WriteHeader(http.StatusFound)
  }
}
func (h *personHandlers) getPerson(w http.ResponseWriter, r *http.Request) {
  parts := strings.Split(r.URL.String(), "/")
  if len(parts) != 3 {
    w.WriteHeader(http.StatusNotFound)
    return
  }

  if parts[2] == "random" {
    h.getRandomPerson(w,r)
    return
  }
  h.Lock()
  person, ok := h.store[parts[2]]
  h.Unlock()
  if !ok{
    w.WriteHeader(http.StatusNotFound)
    return
  }

  jsonBytes, err := json.Marshal(person)
  if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write([]byte(err.Error()))
    return
  }

  w.Header().Add("content=type", "application/json")
  w.WriteHeader(http.StatusOK)
  w.Write(jsonBytes)
}

func (h *personHandlers) post(w http.ResponseWriter, r *http.Request) {
  bodyBytes, err := ioutil.ReadAll(r.Body)
  defer r.Body.Close()
  if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write([]byte(err.Error()))
    return
  }

  ct := r.Header.Get("content-type")
  if ct != "application/json" {
    w.WriteHeader(http.StatusUnsupportedMediaType)
    w.Write([]byte(fmt.Sprintf("need content-type 'application/json', but got '%s'", ct)))
    return
  }

  var person Person
  err = json.Unmarshal(bodyBytes, &person)
  if err != nil {
    w.WriteHeader(http.StatusBadRequest)
    w.Write([]byte(err.Error()))
  }

  person.ID = fmt.Sprintf("%d", time.Now().UnixNano())

  h.Lock()
  h.store[person.ID] = person
  defer h.Unlock()

}

func newPersonHandlers() *personHandlers {
  return &personHandlers{
    store: map[string]Person{},
  }
}

type adminPortal struct {
  password string
}

func newAdminPortal() *adminPortal{
  password := os.Getenv("ADMIN_PASSWORD")
  if password == "" {
    panic("password not set")
  }

  return &adminPortal{password: password}
}

func (a adminPortal) handler(w http.ResponseWriter, r *http.Request) {
  user, pass, ok := r.BasicAuth()
  if !ok || user != "admin" || pass != a.password {
    w.WriteHeader(http.StatusUnauthorized)
    w.Write([]byte("401 - Unauthorized"))
    return
  }

  w.Write([]byte("Admin portal"))
}

func main () {
  admin := newAdminPortal()
  personHandlers := newPersonHandlers()
  http.HandleFunc("/persons", personHandlers.persons)
  http.HandleFunc("/persons/", personHandlers.getPerson)
  http.HandleFunc("/admin", admin.handler)
  err:= http.ListenAndServe(":8080", nil)
  if err != nil {
    panic(err)
  }
}
