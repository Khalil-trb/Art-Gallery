package controller

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const MetBaseURL = "https://collectionapi.metmuseum.org/public/collection/v1"

// --- STRUCTURES ---
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

// --- HELPER FUNCTIONS ---

func fetchJSON(url string, target interface{}) error {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("❌ Network Error: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("❌ API returned status %d for %s\n", resp.StatusCode, url)
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

// SAFE FETCH: pour eviter de bloquer
func fetchObjectsDetails(ids []int) []Object {
	var objects []Object

	fmt.Printf("⏳ Fetching details for %d items (Sequential Mode)...\n", len(ids))

	for _, id := range ids {
		var obj Object
		url := fmt.Sprintf("%s/objects/%d", MetBaseURL, id)
		time.Sleep(50 * time.Millisecond)

		if err := fetchJSON(url, &obj); err == nil {
			if obj.PrimaryImage != "" || obj.PrimaryImageSmall != "" {
				objects = append(objects, obj)
				fmt.Print(".")
			}
		}
	}
	fmt.Println("\n✅ Fetch complete.")
	return objects
}

// --- CONTROLLER HANDLERS ---

func Index(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Could not load template", 500)
		return
	}
	tmpl.Execute(w, nil)
}

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query().Get("q")
	dept := r.URL.Query().Get("dept")
	period := r.URL.Query().Get("period")

	fmt.Printf("\n🔍 NEW SEARCH: '%s'\n", query)

	// 1. Get IDs
	searchURL := fmt.Sprintf("%s/search?q=%s&hasImages=true", MetBaseURL, url.QueryEscape(query))
	if dept != "" {
		searchURL += fmt.Sprintf("&departmentId=%s", dept)
	}

	var searchResp SearchResponse
	if err := fetchJSON(searchURL, &searchResp); err != nil {
		fmt.Println("❌ Search API Failed")
		json.NewEncoder(w).Encode([]Object{})
		return
	}

	// 2. Handle "No Results" (API returns null objectIDs)
	if searchResp.Total == 0 || len(searchResp.ObjectIDs) == 0 {
		fmt.Println("⚠️ API found 0 results.")
		json.NewEncoder(w).Encode([]Object{})
		return
	}

	fmt.Printf("✅ API found %d IDs. Processing top 15...\n", searchResp.Total)

	// 3. Limit to 15 items (Safe number)
	limit := 15
	if len(searchResp.ObjectIDs) < limit {
		limit = len(searchResp.ObjectIDs)
	}

	// 4. Fetch Details
	loadedObjects := fetchObjectsDetails(searchResp.ObjectIDs[:limit])

	// 5. Filter Dates
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

	fmt.Printf("📤 Sending %d final objects to browser.\n", len(finalResults))
	json.NewEncoder(w).Encode(finalResults)
}

func HandleRandom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Println("\n🎲 Fetching Random...")

	// Search for "Paintings" to get a pool of IDs
	var allObjects SearchResponse
	// Search for something broad to get random IDs
	fetchJSON(MetBaseURL+"/search?q=painting&hasImages=true", &allObjects)

	if len(allObjects.ObjectIDs) == 0 {
		json.NewEncoder(w).Encode([]Object{})
		return
	}

	rand.Seed(time.Now().UnixNano())
	randomIDs := make([]int, 0)

	for len(randomIDs) < 10 {
		idx := rand.Intn(len(allObjects.ObjectIDs))
		randomIDs = append(randomIDs, allObjects.ObjectIDs[idx])
	}

	results := fetchObjectsDetails(randomIDs)
	json.NewEncoder(w).Encode(results)
}

func HandleObject(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		return
	}
	id := parts[3]

	var obj Object
	fetchJSON(fmt.Sprintf("%s/objects/%s", MetBaseURL, id), &obj)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(obj)
}

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
