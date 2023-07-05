package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

func main() {
	// http.HandleFunc("/", helloHandler)

	router := http.NewServeMux()
	router.HandleFunc("/hello", helloHandler)
	router.HandleFunc("/healthcheck", helloHandler)
	router.HandleFunc("/hello/custom", helloHandler)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		fmt.Println("Server listening on localhost:8090")
		log.Fatal(http.ListenAndServe("localhost:8090", router)) //nolint
		wg.Done()
	}()

	router1 := http.NewServeMux()
	router1.HandleFunc("/healthcheck", helloHandler)

	wg.Add(1)
	go func() {
		fmt.Println("Server listening on localhost:8091")
		log.Fatal(http.ListenAndServe(":8091", router1)) //nolint
		wg.Done()
	}()

	router2 := http.NewServeMux()
	router2.HandleFunc("/", helloHandler)
	router2.HandleFunc("/hc", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 4)
		fmt.Println("request received", r.Header, r.Host, r.RequestURI)
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
