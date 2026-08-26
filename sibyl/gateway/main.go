package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ask", HandleSSE)
	
	fmt.Println("Starting server on :8080")
	http.ListenAndServe(":8080", mux)
}
