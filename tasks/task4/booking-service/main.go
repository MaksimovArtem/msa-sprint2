package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	enableFeatureXEnv := os.Getenv("ENABLE_FEATURE_X")
	enableFeatureX := enableFeatureXEnv == "true"
	log.Printf("ENABLE_FEATURE_X=%q enableFeatureX=%t", enableFeatureXEnv, enableFeatureX)

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		if enableFeatureX {
			fmt.Fprintf(w, "pong-feature")
			return
		}
		fmt.Fprintf(w, "pong")
	})

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
