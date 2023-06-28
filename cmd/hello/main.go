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
	log.Fatal(http.ListenAndServe("localhost:8090", nil)) //nolint
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("request received %#v\n", r.Header)
	fmt.Fprintf(w, "Hello, World!\n")
}
