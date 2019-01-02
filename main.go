package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
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
	port := ":8080"
	router := mux.NewRouter()

	router.Use(commonMiddleware)

	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/insta", InstaHandler)

	originsOk := handlers.AllowedOrigins([]string{"*"})
	headersOk := handlers.AllowedHeaders([]string{})
	methodsOk := handlers.AllowedMethods([]string{"GET"})

	fmt.Printf("Listening on port '%s'\n", port)
	log.Fatal(http.ListenAndServe(port, handlers.CORS(originsOk, headersOk, methodsOk)(router)))
}
