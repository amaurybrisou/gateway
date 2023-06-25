package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/custom", helloHandler)

	fmt.Println("Server listening on localhost:8090")
	log.Fatal(http.ListenAndServe("localhost:8090", nil))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("request received %s", r.Method)
	fmt.Fprintf(w, "Hello, World!")
}
