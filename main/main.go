package main

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
)

/* Bilješke:
Neke stvari nisu mi bile najjasnije. Stoga ću ovdje navesti kako sam razumio zadatak.

/jmbag Ovaj endpoint ne treba biti zasticen sa user/password kombinacijom, dok ostali
moraju biti odbijeni ako pristupni podaci nisu korektni
-> Implementacija zaštite preko WWW-Authenticate i Authorization headera

/multiply Za grupe s vise od jednog studenta
-> Mora postojati barem dva korisnika s nepraznim jmbagom u config.yaml
-> Inače ne množi

I unindented the last line in config.yaml because the syntax was wrong.
*/


type Http struct {
	Address string `yaml:"address"`
	Port string `yaml:"port"`
}

type User struct {
	Name string `yaml:"name"`
	Jmbag string `yaml:"jmbag"`
	Password string `yaml:"password"`
}

type Config struct {
	Jmbag string
	Http struct {
		Address string `yaml:"address"`
		Port string `yaml:"port"`
	}
	Users []struct {
		Name string `yaml:"name"`
		Jmbag string `yaml:"jmbag"`
		Password string `yaml:"password"`
	}
}

func (c *Config) GetConfig() *Config {
	yamlFile, err := os.ReadFile("files/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nConfig: %v\n\n", c)

	return c
}

type Calc struct {
	A      int `json:"a"`
	B      int `json:"b"`
	Result int `json:"result"`
}


func authorize(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(authHeader[6:])
	if err != nil {
		log.Fatal(err)
	}
	decodedAuth := string(decoded[:])

	var c Config
	c.GetConfig()
	
	for _, user := range c.Users {
		if decodedAuth == user.Name + ":" + user.Password {
			return true
		}
	}
	return false
}

func extractParams(r *http.Request) (int, int, error) {
	params := r.URL.Query()
	a, err := strconv.Atoi(params.Get("a"))
	if err != nil {
		return 0, 0, err
	}
	b, err := strconv.Atoi(params.Get("b"))
	if err != nil {
		return 0, 0, err
	}
	return a, b, nil
}

func sum(w http.ResponseWriter, r *http.Request) {
	if !authorize(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted operations"`)
		w.WriteHeader(http.StatusUnauthorized)
		_, err := w.Write([]byte("401 Unauthorized\n"))
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	a, b, err := extractParams(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("400 Bad Request\n"))
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	calc := Calc{
		A:      a,
		B:      b,
		Result: a + b,
	}
	
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(calc)
	if err != nil {
		log.Fatal(err)
	}
}

func checkForTwoStudents() bool {
		// If number of users with non-empty jmbag (students) is less than 2 don't multiply
	var c Config
	c.GetConfig()
	var students []User
	for _, user := range c.Users {
		if user.Jmbag != "" {
			students = append(students, user)
		}
	}
	if len(students) < 2 {
		return false
	}
	return true
}

func multiply(w http.ResponseWriter, r *http.Request) {

	if !authorize(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted operations"`)
		w.WriteHeader(http.StatusUnauthorized)
		_, err := w.Write([]byte("401 Unauthorized\n"))
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if !checkForTwoStudents() {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("At least two student users are required to multiply.\n"))
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	a, b, err := extractParams(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("400 Bad Request\n"))
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	calc := Calc{
		A:      a,
		B:      b,
		Result: a + b,
	}
	
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(calc)
	if err != nil {
		log.Fatal(err)
	}
}

func jmbag(w http.ResponseWriter, r *http.Request) {
	// Return the jmbag from the config file
	var c Config
	c.GetConfig()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(c.Jmbag)
	if err != nil {
		log.Fatal(err)
	}
}

func root(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Hello World!"))
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	var c Config
	c.GetConfig()

	http.HandleFunc("/", root)
	http.HandleFunc("/sum", sum)
	http.HandleFunc("/multiply", multiply)
	http.HandleFunc("/jmbag", jmbag)
	fmt.Println("Server is listening at http://" + c.Http.Address + ":" + c.Http.Port)
	log.Fatal(http.ListenAndServe(c.Http.Address + ":" + c.Http.Port, nil))
}
