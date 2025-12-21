package router

import (
	"net/http"
	
	// CHECK YOUR GO.MOD FOR THE CORRECT IMPORT PATH
	"Art-Gallery/controller" 
)

func New() *http.ServeMux {
	mux := http.NewServeMux()

	// 1. Static Assets (CSS, Images)
	fs := http.FileServer(http.Dir("assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	// 2. HTML Page
	mux.HandleFunc("/", controller.Index)

	// 3. API Routes
	mux.HandleFunc("/api/departments", controller.HandleDepartments)
	mux.HandleFunc("/api/search", controller.HandleSearch)
	mux.HandleFunc("/api/objects", controller.HandleObjects)
	mux.HandleFunc("/api/object/", controller.HandleObject)

	return mux
}
