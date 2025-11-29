package router

import (
	"net/http"

	"Art-Gallery/controller"
)

// New returns a fully configured router
func New() *http.ServeMux {
	mux := http.NewServeMux()

	// Home page (serve index.html from templates)
	mux.HandleFunc("/", controller.Index)

	// Serve static assets (CSS, JS, images)
	mux.Handle("/assets/",
		http.StripPrefix("/assets/",
			http.FileServer(http.Dir("./assets")),
		),
	)

	// Serve templates (HTML)
	mux.Handle("/templates/",
		http.StripPrefix("/templates/",
			http.FileServer(http.Dir("./templates")),
		),
	)

	return mux
}
