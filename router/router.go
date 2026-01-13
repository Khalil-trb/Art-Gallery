package router

import (
    "Art-Gallery/controller"
    "net/http"
)

func New() *http.ServeMux {
    mux := http.NewServeMux()

    // 1. Static Assets (CSS, JS, Images)
    // Make sure your CSS/JS files are in a folder named "assets"
    fs := http.FileServer(http.Dir("assets"))
    mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

    // 2. HTML Page
    mux.HandleFunc("/", controller.Index)

    // 3. API Routes
    mux.HandleFunc("/api/departments", controller.HandleDepartments) // Filters dropdown
    mux.HandleFunc("/api/search", controller.HandleSearch)           // Search + Date Filter
    mux.HandleFunc("/api/random", controller.HandleRandom)           // NEW: Smart Random Selection
    mux.HandleFunc("/api/object/", controller.HandleObject)          // Single Object Details

    return mux
}
