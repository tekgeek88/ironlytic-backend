package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	// Only load .env files locally — never in containerized (K8s) environments
	if env != "staging" && env != "production" {
		filename := ".env." + env
		if err := godotenv.Load(filename); err != nil {
			log.Printf("⚠️  Could not load %s (falling back to .env): %v", filename, err)
			if err := godotenv.Load(); err != nil {
				log.Printf("⚠️  Could not load fallback .env either: %v", err)
			} else {
				log.Println("✅ Loaded fallback .env")
			}
		} else {
			log.Printf("✅ Loaded environment config from %s", filename)
		}
	} else {
		log.Printf("🚀 ENV is set to %s — skipping godotenv", env)
	}
}
