package controller

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
)

const MetBaseURL = "https://collectionapi.metmuseum.org/public/collection/v1"

// --- 1. HTML HANDLER ---

// Index serves the HTML file
func Index(w http.ResponseWriter, r *http.Request) {
	// Ensure index.html is inside the "templates" folder
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Could not load template", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	tmpl.Execute(w, nil)
}

// --- 2. API HANDLERS (The Logic) ---

// Helper function
func proxyRequest(w http.ResponseWriter, url string) {
	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, "Failed to call Met API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, resp.Body)
}

func HandleDepartments(w http.ResponseWriter, r *http.Request) {
	proxyRequest(w, MetBaseURL+"/departments")
}

func HandleObjects(w http.ResponseWriter, r *http.Request) {
	proxyRequest(w, MetBaseURL+"/objects")
}

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query param", http.StatusBadRequest)
		return
	}
	proxyRequest(w, fmt.Sprintf("%s/search?q=%s", MetBaseURL, query))
}

func HandleObject(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id := pathParts[3]
	proxyRequest(w, fmt.Sprintf("%s/objects/%s", MetBaseURL, id))
}