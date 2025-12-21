package main

import (
	"fmt"
	"log"
	"net/http"

	// CHECK YOUR GO.MOD FOR THE CORRECT IMPORT PATH
	"Art-Gallery/router"
)

func main() {
	// Initialize the router
	r := router.New()

	fmt.Println("Server starting on http://localhost:8080")
	
	// Start the server using the router we just built
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}