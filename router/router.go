package router

import (
	"net/http"

	"Art-Gallery/controller"
)

// New returns a fully configured router
func New() *http.ServeMux {
	mux := http.NewServeMux()

	// Home page
	mux.HandleFunc("/", controller.Index)

	// Search page (with filters & pagination)
	mux.HandleFunc("/search", controller.SearchHandler)

	// Random artworks
	mux.HandleFunc("/random", controller.RandomHandler)

	// Serve static assets (CSS, JS, images)
	mux.Handle("/assets/",
		http.StripPrefix("/assets/",
			http.FileServer(http.Dir("./assets")),
		),
	)
	return mux
}
