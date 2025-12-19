package controller

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

const BaseURL = "https://collectionapi.metmuseum.org/public/collection/v1"

// ----------------------
// Structs
// ----------------------

type Department struct {
	DepartmentID int    `json:"departmentId"`
	DisplayName  string `json:"displayName"`
}

type DepartmentsResponse struct {
	Departments []Department `json:"departments"`
}

type SearchResponse struct {
	Total     int   `json:"total"`
	ObjectIDs []int `json:"objectIDs"`
}

type Object struct {
	ObjectID          int    `json:"objectID"`
	Title             string `json:"title"`
	ArtistDisplayName string `json:"artistDisplayName"`
	ObjectDate        string `json:"objectDate"`
	Department        string `json:"department"`
	DepartmentID      int    `json:"departmentId"`
	PrimaryImage      string `json:"primaryImage"`
	PrimaryImageSmall string `json:"primaryImageSmall"`
	ObjectBeginDate   int    `json:"objectBeginDate"`
	ObjectEndDate     int    `json:"objectEndDate"`
	CreditLine        string `json:"creditLine"`
	Medium            string `json:"medium"`
	IsPublicDomain    bool   `json:"isPublicDomain"`
}

type PageData struct {
	Query          string
	Objects        []*Object
	Departments    []Department
	SelectedDept   int
	SelectedPeriod string
	CurrentPage    int
	TotalPages     int
}

// ----------------------
// Helper Functions
// ----------------------

// Fetch JSON from URL
func fetchJSON(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

// Load all departments
func LoadDepartments() ([]Department, error) {
	var res DepartmentsResponse
	if err := fetchJSON(fmt.Sprintf("%s/departments", BaseURL), &res); err != nil {
		return nil, err
	}
	return res.Departments, nil
}

// Search object IDs by query
func SearchObjects(query string) ([]int, error) {
	var res SearchResponse
	if err := fetchJSON(fmt.Sprintf("%s/search?q=%s", BaseURL, query), &res); err != nil {
		return nil, err
	}
	return res.ObjectIDs, nil
}

// Load single object
func LoadObject(id int) (*Object, error) {
	var obj Object
	if err := fetchJSON(fmt.Sprintf("%s/objects/%d", BaseURL, id), &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

// Load multiple objects
func LoadObjects(ids []int) ([]*Object, error) {
	objects := []*Object{}
	for _, id := range ids {
		obj, err := LoadObject(id)
		if err != nil {
			log.Println("Failed to load object", id, err)
			continue
		}
		if obj != nil {
			objects = append(objects, obj)
		}
	}
	return objects, nil
}

// Filter objects
func FilterObjects(objects []*Object, departmentID int, period string) []*Object {
	filtered := []*Object{}
	for _, obj := range objects {
		if obj == nil {
			continue
		}
		if departmentID != 0 && obj.DepartmentID != departmentID {
			continue
		}
		begin := obj.ObjectBeginDate
		end := obj.ObjectEndDate
		switch period {
		case "before1500":
			if end >= 1500 {
				continue
			}
		case "1500-1800":
			if begin > 1800 || end < 1500 {
				continue
			}
		case "after1800":
			if begin <= 1800 {
				continue
			}
		}
		if obj.PrimaryImage == "" && obj.PrimaryImageSmall == "" {
			continue
		}
		filtered = append(filtered, obj)
	}
	return filtered
}

// Load random objects
func LoadRandom(count int) ([]*Object, error) {
	var res SearchResponse
	if err := fetchJSON(fmt.Sprintf("%s/objects", BaseURL), &res); err != nil {
		return nil, err
	}
	allIDs := res.ObjectIDs
	rand.Seed(time.Now().UnixNano())

	picked := make(map[int]struct{})
	randomIDs := []int{}
	for len(randomIDs) < count && len(randomIDs) < len(allIDs) {
		i := allIDs[rand.Intn(len(allIDs))]
		if _, exists := picked[i]; !exists {
			picked[i] = struct{}{}
			randomIDs = append(randomIDs, i)
		}
	}
	return LoadObjects(randomIDs)
}

// ----------------------
// Template Helper
// ----------------------

func renderTemplate(w http.ResponseWriter, data PageData) {
	funcMap := template.FuncMap{
		"eq":  func(a, b interface{}) bool { return a == b },
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}
	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles("./templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ----------------------
// Handlers
// ----------------------

// Index page
func Index(w http.ResponseWriter, r *http.Request) {
	defaultIDs := []int{199313, 436105, 435702, 437473, 437327, 438417, 435813, 204758, 204812, 193628, 250748, 248146, 24320, 24671, 22364, 23939, 22239, 24693}
	objects, _ := LoadObjects(defaultIDs)
	departments, _ := LoadDepartments()

	data := PageData{
		Objects:     objects,
		Departments: departments,
		CurrentPage: 1,
		TotalPages:  1,
	}

	renderTemplate(w, data)
}

// Search page with filters & pagination
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	deptStr := r.URL.Query().Get("department")
	period := r.URL.Query().Get("period")
	pageStr := r.URL.Query().Get("page")

	deptID := 0
	if deptStr != "" {
		if d, err := strconv.Atoi(deptStr); err == nil {
			deptID = d
		}
	}

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			page = p
		}
	}

	var objects []*Object
	if query != "" {
		ids, _ := SearchObjects(query)
		allObjects, _ := LoadObjects(ids)
		objects = FilterObjects(allObjects, deptID, period)
	}

	const perPage = 6
	start := (page - 1) * perPage
	end := start + perPage
	if start > len(objects) {
		start = len(objects)
	}
	if end > len(objects) {
		end = len(objects)
	}
	pageObjects := objects[start:end]

	departments, _ := LoadDepartments()

	data := PageData{
		Query:          query,
		Objects:        pageObjects,
		Departments:    departments,
		SelectedDept:   deptID,
		SelectedPeriod: period,
		CurrentPage:    page,
		TotalPages:     (len(objects) + perPage - 1) / perPage,
	}

	renderTemplate(w, data)
}

// Random artworks
func RandomHandler(w http.ResponseWriter, r *http.Request) {
	objects, _ := LoadRandom(20)
	departments, _ := LoadDepartments()

	data := PageData{
		Objects:     objects,
		Departments: departments,
		CurrentPage: 1,
		TotalPages:  1,
	}

	renderTemplate(w, data)
}
