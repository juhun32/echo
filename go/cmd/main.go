package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file (ignoring if running in cloud/docker)")
	}

	if err := initS3Client(); err != nil {
		log.Printf("Warning: S3 disabled: %v", err)
	} else {
		downloadAndMergeFromS3()
		startBackgroundSync()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/chat", handleChat)
	mux.HandleFunc("/history", handleHistory)
	mux.HandleFunc("/cache-stats", handleCacheStats)

	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
	}).Handler(mux)

	fmt.Println("Echo backend listening on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		panic(err)
	}
}
