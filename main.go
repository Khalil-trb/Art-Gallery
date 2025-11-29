package main

import (
	"log"
	"net/http"

	"Art-Gallery/router"
)

func main() {
	// Load router from router package
	r := router.New()

	log.Println("🚀 Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
