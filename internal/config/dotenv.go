package config

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func LoadDotEnv() error {
	candidates := []string{
		".env",
		filepath.Join("..", ".env"),
		filepath.Join("..", "..", ".env"),
	}

	var loaded bool
	for _, path := range candidates {
		if err := loadFile(path); err == nil {
			log.Printf("Successfully loaded .env file from: %s", path)
			loaded = true
			break
		} else {
			log.Printf("Failed to load .env file from %s: %v (trying next candidate)", path, err)
		}
	}

	if !loaded {
		log.Println("Warning: No .env file found - using default values or environment variables")
	}

	return nil
}

func loadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var loadedVars []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			log.Printf("Environment variable %s already exists - skipping from .env", key)
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			log.Printf("Failed to set environment variable %s: %v", key, err)
			continue
		}
		loadedVars = append(loadedVars, key)
	}

	log.Printf("Loaded %d environment variables from %s", len(loadedVars), path)

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading .env file: %v", err)
		return err
	}

	return nil
}
