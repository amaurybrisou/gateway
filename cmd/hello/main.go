package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

func main() {
	// http.HandleFunc("/", helloHandler)

	router := http.NewServeMux()
	router.HandleFunc("/", helloHandler)
	router.HandleFunc("/hc", helloHandler)
	router.HandleFunc("/custom", helloHandler)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		fmt.Println("Server listening on localhost:8090")
		log.Fatal(http.ListenAndServe("localhost:8090", router)) //nolint
		wg.Done()
	}()

	router1 := http.NewServeMux()
	router1.HandleFunc("/hc", helloHandler)

	wg.Add(1)
	go func() {
		fmt.Println("Server listening on localhost:8091")
		log.Fatal(http.ListenAndServe(":8091", router1)) //nolint
		wg.Done()
	}()

	router2 := http.NewServeMux()
	router2.HandleFunc("/", helloHandler)
	router2.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("healthcheck received", r.Header, r.Host, r.RequestURI)
		w.WriteHeader(200)
		w.Write(nil) //nolint
	})

	wg.Add(1)
	go func() {
		fmt.Println("Server listening on :8092")
		log.Fatal(http.ListenAndServe(":8092", router2)) //nolint
		wg.Done()
	}()

	wg.Wait()
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("request received", r.Header, r.Host, r.RequestURI)
	fmt.Fprintf(w, "Hello, World!\n")
}
