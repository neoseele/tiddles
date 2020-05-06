package db

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Person type
type Person struct {
	ID        string   `json:"id,omitempty" bson:"id"`
	Firstname string   `json:"firstname,omitempty" bson:"firstname"`
	Lastname  string   `json:"lastname,omitempty" bson:"lastname"`
	Address   *Address `json:"address,omitempty" bson:"address"`
}

// Address type
type Address struct {
	City  string `json:"city,omitempty" bson:"city"`
	State string `json:"state,omitempty" bson:"state"`
}

var db []Person
var mongoSession *mgo.Session

// Init ...
func Init(session *mgo.Session) {
	if session != nil {
		mongoSession = session
	} else {
		db = append(db, Person{ID: "1", Firstname: "John", Lastname: "Doe", Address: &Address{City: "City X", State: "State X"}})
		// db = append(db, Person{ID: "2", Firstname: "Koko", Lastname: "Doe", Address: &Address{City: "City Z", State: "State Y"}})
	}
}

func getCollection() (*mgo.Collection, *mgo.Session) {
	s := mongoSession.Copy()
	c := s.DB("test").C("people")

	return c, s
}

// GetAll person objects
func GetAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if mongoSession != nil {
		// get all person from mongodb
		c, s := getCollection()
		defer s.Close()

		result := []Person{}
		err := c.Find(nil).All(&result)
		if err != nil {
			log.Printf("RunQuery : ERROR : %s\n", err)
			return
		}

		json.NewEncoder(w).Encode(result)

	} else {
		json.NewEncoder(w).Encode(db)
	}
}

// Get a person object
func Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)

	if mongoSession != nil {
		// find the person from mongodb
		c, s := getCollection()
		defer s.Close()

		p := Person{}
		err := c.Find(bson.M{"id": params["id"]}).One(&p)
		if err != nil {
			log.Printf("RunQuery : ERROR : %s\n", err)
			return
		}
		json.NewEncoder(w).Encode(p)

	} else {
		// find the person from the db slice
		for _, p := range db {
			if p.ID == params["id"] {
				json.NewEncoder(w).Encode(p)
				return
			}
		}
	}
}

// Create a person object
// example: curl -d '{"id":"100", "firstname":"foo", "lastname":"bar"}' -H "Content-Type: application/json" -X POST http://backend:8000/db/100
func Create(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	var p Person
	_ = json.NewDecoder(r.Body).Decode(&p)
	p.ID = params["id"]

	if mongoSession != nil {
		c, s := getCollection()
		defer s.Close()

		err := c.Insert(p)
		if err != nil {
			log.Printf("RunQuery : ERROR : %s\n", err)
			return
		}

		GetAll(w, r)
	} else {
		db = append(db, p)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(db)
	}
}

// Delete a person object
// example: curl -X DELETE http://backend:8000/db/100
func Delete(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	if mongoSession != nil {
		c, s := getCollection()
		defer s.Close()

		err := c.Remove(bson.M{"id": params["id"]})
		if err != nil {
			log.Printf("RunQuery : ERROR : %s\n", err)
			return
		}

		GetAll(w, r)
	} else {
		w.Header().Set("Content-Type", "application/json")
		for i, p := range db {
			if p.ID == params["id"] {
				db = append(db[:i], db[i+1:]...)
				break
			}
			json.NewEncoder(w).Encode(db)
		}
	}
}

// Update a person object
// example: curl -d '{"lastname":"brad"}' -X PUT http://backend:8000/db/100
func Update(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	var p Person
	_ = json.NewDecoder(r.Body).Decode(&p)
	p.ID = params["id"]

	if mongoSession != nil {
		c, s := getCollection()
		defer s.Close()

		err := c.Update(bson.M{"id": params["id"]}, &p)
		if err != nil {
			log.Printf("RunQuery : ERROR : %s\n", err)
			return
		}

		GetAll(w, r)
	} else {
		// remove the matched person from the slice
		for i, p := range db {
			if p.ID == params["id"] {
				// this approach is supposed to be memory leak free when
				// the element of the slice is a pointer or a struct with pointer fields.
				// since the slice here is non of that, we don't have to use this, but
				// I am leaving this bit of code here just for reference.
				// (https://github.com/golang/go/wiki/SliceTricks)
				copy(db[i:], db[i+1:])
				db[len(db)-1] = Person{}
				db = db[:len(db)-1]
				break
			}
		}
		// add the updated person back into the slice
		db = append(db, p)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(db)
	}
}
