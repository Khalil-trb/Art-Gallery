package main

import (
	"fmt"
	"log"
	"net/http"

	
	"Art-Gallery/router"
)

func main() {
	// Lance le router
	r := router.New()

	fmt.Println("Server starting on http://localhost:8080")
	
	// Lance le serveur
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}