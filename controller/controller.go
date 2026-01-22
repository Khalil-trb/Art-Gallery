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
	"sync"
	"time"
)

// API
const MetBaseURL = "https://collectionapi.metmuseum.org/public/collection/v1"

// numero unique pour communiquer avec le musée 
var client = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
}

// SYSTÈME DE CACHE
var (
	objectCache = make(map[int]Object)
	cacheMutex  sync.RWMutex
)

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
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf(" Network Error: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 403 {
			fmt.Println("BLOCAGE API (403) - Attendre 1 minute...")
		}
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

// FONCTION MODIFIÉE AVEC CACHE
func fetchObjectsDetails(ids []int) []Object {
	var objects []Object
	var idsToFetch []int

	// ÉTAPE 1 : Verification du cache
	for _, id := range ids {
		cacheMutex.RLock()
		cachedObj, found := objectCache[id]
		cacheMutex.RUnlock()

		if found {
			objects = append(objects, cachedObj)
		} else {
			idsToFetch = append(idsToFetch, id)
		}
	}

	if len(idsToFetch) == 0 {
		return objects
	}

	fmt.Printf("Fetching %d items from API (Found %d in cache)...\n", len(idsToFetch), len(objects))
	for _, id := range idsToFetch {
		var obj Object
		url := fmt.Sprintf("%s/objects/%d", MetBaseURL, id)

		time.Sleep(300 * time.Millisecond)

		if err := fetchJSON(url, &obj); err == nil {
			if obj.PrimaryImage != "" || obj.PrimaryImageSmall != "" {
				objects = append(objects, obj)
				cacheMutex.Lock()
				objectCache[id] = obj
				cacheMutex.Unlock()

				fmt.Print(".")
			}
		} else {
			fmt.Printf("x")
		}
	}
	fmt.Println("\n✓ Done.")
	return objects
}

// RENDU HTML CÔTÉ SERVEUR

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

// Page d'accueil
func Index(w http.ResponseWriter, r *http.Request) {
	homePaintingIDs := []int{
		199313, 436105, 435702, 437473, 437327, 438417,
		436532, 435813, 204758, 204812, 193628, 250748,
		248146, 24320, 24671, 22364, 23939, 22239,
		24693, 446653, 446273, 22871, 22506, 24960,
	}
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
	if start < 0 {
		start = 0
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

// HandleSearch
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

	limit := 50
	if len(searchResp.ObjectIDs) < limit {
		limit = len(searchResp.ObjectIDs)
	}

	loadedObjects := fetchObjectsDetails(searchResp.ObjectIDs[:limit])

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

	itemsPerPage := 12
	totalPages := (len(finalResults) + itemsPerPage - 1) / itemsPerPage

	if page > totalPages && totalPages > 0 {
		page = totalPages
	}
	if totalPages == 0 {
		page = 1
	}

	start := (page - 1) * itemsPerPage
	end := start + itemsPerPage
	if end > len(finalResults) {
		end = len(finalResults)
	}

	var pageObjects []Object
	if start < len(finalResults) {
		pageObjects = finalResults[start:end]
	}

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

// Oeuvres aleatoires
func HandleRandom(w http.ResponseWriter, r *http.Request) {
	fmt.Println("\n Fetching Random...")

	var allObjects SearchResponse
	fetchJSON(MetBaseURL+"/search?q=painting&hasImages=true", &allObjects)

	if len(allObjects.ObjectIDs) == 0 {
		renderEmpty(w, "random")
		return
	}

	rand.Seed(time.Now().UnixNano())
	randomIDs := make([]int, 0)

	maxItems := 12
	if len(allObjects.ObjectIDs) < 12 {
		maxItems = len(allObjects.ObjectIDs)
	}

	usedIndices := make(map[int]bool)

	for len(randomIDs) < maxItems {
		idx := rand.Intn(len(allObjects.ObjectIDs))
		if !usedIndices[idx] {
			usedIndices[idx] = true
			randomIDs = append(randomIDs, allObjects.ObjectIDs[idx])
		}
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

// Objets
func HandleObject(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", 400)
		return
	}
	// Récupère l'ID depuis l'URL
	idStr := parts[len(parts)-1]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", 400)
		return
	}

	// VERIFICATION CACHE (Optimisation)
	cacheMutex.RLock()
	cachedObj, found := objectCache[id]
	cacheMutex.RUnlock()

	var obj Object
	if found {
		// Recuperer depuis le cache
		obj = cachedObj
		fmt.Println(" Objet récupéré depuis le cache !")
	} else {
		// Sinon on télécharge
		if err := fetchJSON(fmt.Sprintf("%s/objects/%d", MetBaseURL, id), &obj); err != nil {
			http.Error(w, "Object not found", 404)
			return
		}
		// Et on sauvegarde
		cacheMutex.Lock()
		objectCache[id] = obj
		cacheMutex.Unlock()
	}

	tmpl, err := template.ParseFiles("templates/object.html")
	if err != nil {
		http.Error(w, "Could not load template", 500)
		return
	}
	tmpl.Execute(w, obj)
}

// Departements
func HandleDepartments(w http.ResponseWriter, r *http.Request) {
	resp, err := client.Get(MetBaseURL + "/departments")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	var data interface{}
	json.NewDecoder(resp.Body).Decode(&data)
	json.NewEncoder(w).Encode(data)
}

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
