package controller

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const MetBaseURL = "https://collectionapi.metmuseum.org/public/collection/v1"

// STRUCTURES 
type Object struct {
	ObjectID          int    `json:"objectID"`
	Title             string `json:"title"`
	PrimaryImage      string `json:"primaryImage"`
	PrimaryImageSmall string `json:"primaryImageSmall"`
	Department        string `json:"department"`
	ObjectDate        string `json:"objectDate"`
	ArtistDisplayName string `json:"artistDisplayName"`
	ObjectBeginDate   int    `json:"objectBeginDate"`
	ObjectEndDate     int    `json:"objectEndDate"`
}

type SearchResponse struct {
	Total     int   `json:"total"`
	ObjectIDs []int `json:"objectIDs"`
}

type PageData struct {
	Objects     []Object
	Query       string
	TotalResult int
	CurrentPage int
	TotalPages  int
}

func fetchJSON(url string, target interface{}) error {
	client := &http.Client{Timeout: 15 * time.Second}
	
	// Créer une requête avec User-Agent
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	
	// Ajouter un User-Agent pour éviter le blocage 403
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf(" Network Error: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("API returned status %d for %s\n", resp.StatusCode, url)
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func fetchObjectsDetails(ids []int) []Object {
	var objects []Object

	fmt.Printf("⏳ Fetching details for %d items...\n", len(ids))

	for _, id := range ids {
		var obj Object
		url := fmt.Sprintf("%s/objects/%d", MetBaseURL, id)
		
		// Augmenter le délai entre requêtes pour éviter le blocage
		time.Sleep(200 * time.Millisecond)

		if err := fetchJSON(url, &obj); err == nil {
			if obj.PrimaryImage != "" || obj.PrimaryImageSmall != "" {
				objects = append(objects, obj)
				fmt.Print(".")
			}
		} else {
			fmt.Printf("\n Erreur pour ID %d: %v\n", id, err)
		}
	}
	fmt.Println("\n✓ Fetch complete.")
	return objects
}

//  RENDU HTML CÔTÉ SERVEUR 


var funcMap = template.FuncMap{
	"sub": func(a, b int) int { return a - b },
	"add": func(a, b int) int { return a + b },
	"iterate": func(count int) []int {
		var items []int
		for i := 1; i <= count; i++ {
			items = append(items, i)
		}
		return items
	},
}

// Index - Page d'accueil avec œuvres par défaut
func Index(w http.ResponseWriter, r *http.Request) {
	homePaintingIDs := []int{
		199313, 436105, 435702, 437473, 437327, 438417,
		436532, 435813, 204758, 204812, 193628, 250748,
		248146, 24320, 24671, 22364, 23939, 22239,
		24693, 446653, 446273, 22871, 22506, 24960,
	}

	// Pagination
	const perPage = 8
	page := 1

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	totalResults := len(homePaintingIDs)
	totalPages := (totalResults + perPage - 1) / perPage

	start := (page - 1) * perPage
	end := start + perPage

	if start > totalResults {
		start = 0
		end = perPage
		page = 1
	}
	if end > totalResults {
		end = totalResults
	}
	pageIDs := homePaintingIDs[start:end]
	objects := fetchObjectsDetails(pageIDs)

	data := PageData{
		Objects:     objects,
		TotalResult: totalResults,
		CurrentPage: page,
		TotalPages:  totalPages,
	}

	tmpl, err := template.New("gallery.html").Funcs(funcMap).ParseFiles("templates/gallery.html")
	if err != nil {
		http.Error(w, "Could not load template", 500)
		return
	}

	tmpl.Execute(w, data)
}

// HandleSearch - Recherche avec rendu HTML
func HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	dept := r.URL.Query().Get("dept")
	period := r.URL.Query().Get("period")
	pageStr := r.URL.Query().Get("page")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	fmt.Printf("\n🔍 SEARCH: '%s' (page %d)\n", query, page)

	// Construction de l'URL de recherche
	searchURL := fmt.Sprintf("%s/search?q=%s&hasImages=true", MetBaseURL, url.QueryEscape(query))
	if dept != "" {
		searchURL += fmt.Sprintf("&departmentId=%s", dept)
	}

	var searchResp SearchResponse
	if err := fetchJSON(searchURL, &searchResp); err != nil {
		renderError(w, "Erreur lors de la recherche")
		return
	}

	if searchResp.Total == 0 || len(searchResp.ObjectIDs) == 0 {
		renderEmpty(w, query)
		return
	}

	// Limiter à 50 résultats max
	limit := 50
	if len(searchResp.ObjectIDs) < limit {
		limit = len(searchResp.ObjectIDs)
	}

	loadedObjects := fetchObjectsDetails(searchResp.ObjectIDs[:limit])

	// Filtrer par période
	var finalResults []Object
	for _, obj := range loadedObjects {
		match := true
		begin := obj.ObjectBeginDate
		end := obj.ObjectEndDate
		if end == 0 {
			end = begin
		}

		if period == "before1500" && end >= 1500 {
			match = false
		}
		if period == "1500-1800" && (begin > 1800 || end < 1500) {
			match = false
		}
		if period == "after1800" && begin <= 1800 {
			match = false
		}

		if match {
			finalResults = append(finalResults, obj)
		}
	}

	// Pagination
	itemsPerPage := 12
	totalPages := (len(finalResults) + itemsPerPage - 1) / itemsPerPage
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	start := (page - 1) * itemsPerPage
	end := start + itemsPerPage
	if end > len(finalResults) {
		end = len(finalResults)
	}

	pageObjects := finalResults[start:end]

	data := PageData{
		Objects:     pageObjects,
		Query:       query,
		TotalResult: len(finalResults),
		CurrentPage: page,
		TotalPages:  totalPages,
	}

	tmpl, err := template.New("gallery.html").Funcs(funcMap).ParseFiles("templates/gallery.html")
	if err != nil {
		http.Error(w, "Could not load template", 500)
		return
	}
	tmpl.Execute(w, data)
}

// HandleRandom - Œuvres aléatoires avec rendu HTML
func HandleRandom(w http.ResponseWriter, r *http.Request) {
	fmt.Println("\n🎲 Fetching Random...")

	var allObjects SearchResponse
	fetchJSON(MetBaseURL+"/search?q=painting&hasImages=true", &allObjects)

	if len(allObjects.ObjectIDs) == 0 {
		renderEmpty(w, "random")
		return
	}

	rand.Seed(time.Now().UnixNano())
	randomIDs := make([]int, 0)

	for len(randomIDs) < 12 {
		idx := rand.Intn(len(allObjects.ObjectIDs))
		randomIDs = append(randomIDs, allObjects.ObjectIDs[idx])
	}

	results := fetchObjectsDetails(randomIDs)

	data := PageData{
		Objects:     results,
		TotalResult: len(results),
		CurrentPage: 1,
		TotalPages:  1,
	}

	tmpl, err := template.New("gallery.html").Funcs(funcMap).ParseFiles("templates/gallery.html")
	if err != nil {
		http.Error(w, "Could not load template", 500)
		return
	}
	tmpl.Execute(w, data)
}

// HandleObject - Détails d'une œuvre (rendu HTML)
func HandleObject(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", 400)
		return
	}
	id := parts[len(parts)-1]

	var obj Object
	if err := fetchJSON(fmt.Sprintf("%s/objects/%s", MetBaseURL, id), &obj); err != nil {
		http.Error(w, "Object not found", 404)
		return
	}

	tmpl, err := template.ParseFiles("templates/object.html")
	if err != nil {
		http.Error(w, "Could not load template", 500)
		return
	}
	tmpl.Execute(w, obj)
}

// HandleDepartments - API JSON (garde pour le dropdown)
func HandleDepartments(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get(MetBaseURL + "/departments")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	var data interface{}
	json.NewDecoder(resp.Body).Decode(&data)
	json.NewEncoder(w).Encode(data)
}

// Helpers pour rendu d'erreur
func renderError(w http.ResponseWriter, message string) {
	w.WriteHeader(500)
	fmt.Fprintf(w, "<h1>Erreur</h1><p>%s</p>", message)
}

func renderEmpty(w http.ResponseWriter, query string) {
	tmpl, _ := template.New("gallery.html").Funcs(funcMap).ParseFiles("templates/gallery.html")
	data := PageData{
		Objects:     []Object{},
		Query:       query,
		TotalResult: 0,
	}
	tmpl.Execute(w, data)
}
