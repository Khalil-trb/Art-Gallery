package router

import (
	"Art-Gallery/controller"
	"net/http")

func New() *http.ServeMux {
	mux := http.NewServeMux()

	// CSS + Images
	fs := http.FileServer(http.Dir("assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	// 2. HTML
	mux.HandleFunc("/", controller.Index)

	// 3. Routes des API
	mux.HandleFunc("/api/departments", controller.HandleDepartments)
	mux.HandleFunc("/api/search", controller.HandleSearch)
	mux.HandleFunc("/api/objects", controller.HandleObjects)
	mux.HandleFunc("/api/object/", controller.HandleObject)

	return mux
}
