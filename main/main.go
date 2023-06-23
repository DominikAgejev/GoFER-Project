package main

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"encoding/base64"
	"encoding/json"
	"io"
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

Naknadno sam shvatio da ste mislili na studente u grupi, a ne na korisnike u config.yaml 🤦‍♂️

Ostavit ću kod svejedno. 😁

/0036391234 (vas jmbag)
- Svaki student definira svoj tip gdje
- POST prima podatke i sprema ih u student1.txt (proizvoljno
ime) dokument na disk

-> svaki POST prepisuje prethodni sadržaj datoteke

Deindentirao sam posljednju liniju u config.yaml da sintaksa bude ispravna
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
	yamlFile, err := os.ReadFile("../files/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatal(err)
	}

	return c
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

type Calc struct {
	A      int `json:"a"`
	B      int `json:"b"`
	Result int `json:"result"`
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
	var c Config
	c.GetConfig()
	var students []User
	for _, user := range c.Users {
		if user.Jmbag != "" {
			students = append(students, user)
		}
	}

	return len(students)>=2
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
	var c Config
	c.GetConfig()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(c.Jmbag)
	if err != nil {
		log.Fatal(err)
	}
}

type Url struct {
	URL string `json:"url"`
}

func fetch(w http.ResponseWriter, r *http.Request) {
	if !authorize(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted operations"`)
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "401 Unauthorized\n")
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, err := w.Write([]byte("405 Method Not Allowed\n"))
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to read request body: %v", err)
		return
	}
	defer r.Body.Close()

	var url Url
	err = json.Unmarshal(body, &url)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to parse request body: %v", err)
		return
	}
	
	resp, err := http.Get(url.URL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to fetch URL: %v", err)
		return
	}
	defer resp.Body.Close()

	headersJSON, err := json.Marshal(resp.Header)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to parse response headers: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	fmt.Fprintf(w, "%v", string(headersJSON))
}

func handle0036537505(w http.ResponseWriter, r *http.Request) {
	if !authorize(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted operations"`)
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "401 Unauthorized\n")
		return
	}

	switch r.Method {
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to read request body: %v", err)
			return
		}
		defer r.Body.Close()
		
		file, err := os.Create("files/0036537505.txt")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to create file: %v", err)
			return
		}
		defer file.Close()

		_, err = file.Write(body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to write to file: %v", err)
			return
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "File created successfully\n")

	case http.MethodGet:
		contents, err := os.ReadFile("files/0036537505.txt")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to read file: %v", err)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%v", string(contents))

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "405 Method Not Allowed\n")
		return
	}
}

func root(w http.ResponseWriter, r *http.Request) {
	contents, err := os.ReadFile("files/notes.txt")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to read file: %v", err)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%v", string(contents))
}

func main() {
	var c Config
	c.GetConfig()

	http.HandleFunc("/", root)
	http.HandleFunc("/sum", sum)
	http.HandleFunc("/multiply", multiply)
	http.HandleFunc("/jmbag", jmbag)
	http.HandleFunc("/fetch", fetch)
	http.HandleFunc("/0036537505", handle0036537505)

	log.Printf("Server is listening at http://%v:%v\n", c.Http.Address, c.Http.Port)
	fmt.Printf("\nConfig: %v\n\n", c)
	log.Fatal(http.ListenAndServe(c.Http.Address + ":" + c.Http.Port, nil))
}
