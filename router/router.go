package router

import (
    "Art-Gallery/controller"
    "net/http"
)

func New() *http.ServeMux {
    mux := http.NewServeMux()

    // Static Assets
    fs := http.FileServer(http.Dir("assets"))
    mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

    // Pages HTML (plus d'API JSON)
    mux.HandleFunc("/", controller.Index)
    mux.HandleFunc("/search", controller.HandleSearch)
    mux.HandleFunc("/random", controller.HandleRandom)
    mux.HandleFunc("/object/", controller.HandleObject)
    
    // Garde seulement pour le dropdown
    mux.HandleFunc("/api/departments", controller.HandleDepartments)

    return mux
}
