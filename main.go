package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

// HomeResponse stores information about the api
type HomeResponse struct {
	Alive     bool     `json:"alive"`
	Resources []string `json:"resources"`
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HomeResponse{Alive: true, Resources: []string{"insta"}})
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func main() {
	godotenv.Load() // Load from .env file if it's there
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	router := mux.NewRouter()
	router.Use(commonMiddleware)

	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/insta", InstaHandler)

	originsOk := handlers.AllowedOrigins([]string{"*"})
	headersOk := handlers.AllowedHeaders([]string{})
	methodsOk := handlers.AllowedMethods([]string{"GET"})

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("Listening on '%s'\n", addr)
	log.Fatal(http.ListenAndServe(addr, handlers.CORS(originsOk, headersOk, methodsOk)(router)))
}
