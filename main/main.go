package main

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
)

/*
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
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	fmt.Printf("\n--- config:\n%v\n\n", c)

	return c
}

type Calc struct {
		A int
		B int
		Result int
	}

// Create a simple server that takes in two numbers through the URL and return a json with the sum of those numbers.
// Example: http://localhost:80/sum?a=2&b=1 should return {"a": 2, "b": 1, "result": 3}
func sum(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	a, err := strconv.Atoi(params.Get("a"))
	if err != nil {
		log.Fatal(err)
	}
	b, err := strconv.Atoi(params.Get("b"))
	if err != nil {
		log.Fatal(err)
	}

	calc := Calc{
		A:      a,
		B:      b,
		Result: a + b,
	}
	// Return the result in JSON format
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
	http.HandleFunc("/jmbag", jmbag)
	log.Fatal(http.ListenAndServe(c.Http.Address + ":" + c.Http.Port, nil))
}
