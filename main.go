package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

// URL represents the structure of a URL object
type URL struct {
	Original string `json:"original"`
	Short    string `json:"short"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./data/database.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// // Load and execute the SQL file
	// err = executeSQLFile(db, "./data/load.sql")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	r := mux.NewRouter()

	// CORS middleware
	r.Use(corsMiddleware)

	// Endpoint to handle shortening URLs
	r.HandleFunc("/api/shorten", shortenURL).Methods("POST")

	// Endpoint to redirect shortened URLs
	r.HandleFunc("/{shortened}", redirectToOriginal).Methods("GET")

	// Start the server
	log.Println("Server started on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func executeSQLFile(db *sql.DB, filepath string) error {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %v", err)
	}

	_, err = db.Exec(string(content))
	if err != nil {
		return fmt.Errorf("failed to execute SQL file: %v", err)
	}

	return nil
}

func shortenURL(w http.ResponseWriter, r *http.Request) {
	var url URL
	err := json.NewDecoder(r.Body).Decode(&url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Error decoding JSON: %v", err)
		return
	}

	// Generate a unique shortened URL ID
	shortenedID := generateShortenedID()
	shortenedURL := "http://localhost:8080/" + shortenedID

	// Insert the original and shortened URL into the database
	insertURL := `INSERT INTO urls (original_url, shortened_url) VALUES (?, ?)`
	_, err = db.Exec(insertURL, url.Original, shortenedID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error inserting into database: %v", err)
		return
	}

	// Respond with the shortened URL as JSON
	response := URL{Original: url.Original, Short: shortenedURL}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func redirectToOriginal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortenedID := vars["shortened"]

	// Check if the shortened URL exists in the database
	var originalURL string
	query := `SELECT original_url FROM urls WHERE shortened_url = ?`
	err := db.QueryRow(query, shortenedID).Scan(&originalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		log.Printf("Error querying database: %v", err)
		return
	}

	// Redirect to the original URL
	http.Redirect(w, r, originalURL, http.StatusFound)
}

func generateShortenedID() string {
	// Generate a random 6-character string for the shortened URL ID
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
