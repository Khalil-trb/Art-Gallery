package controller

import (
	"html/template"
	"net/http"
	"path/filepath"
)

// Index handles the home page
func Index(w http.ResponseWriter, r *http.Request) {
	// Path to your index.html file
	path := filepath.Join("templates", "index.html")

	tpl, err := template.ParseFiles(path)
	if err != nil {
		http.Error(w, "Error loading template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tpl.Execute(w, nil)
}
