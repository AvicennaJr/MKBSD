package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	url           = "https://storage.googleapis.com/panels-api/data/20240916/media-1a-i-p~s"
	maxRetries    = 3
	retryDelay    = 5 * time.Second
	clientTimeout = 2 * time.Minute
)

func downloadImage(client *http.Client, imageURL, filePath string, wg *sync.WaitGroup) {
	defer wg.Done()

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := func() error {
			resp, err := client.Get(imageURL)
			if err != nil {
				return fmt.Errorf("error downloading image: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to download image: %d", resp.StatusCode)
			}

			file, err := os.Create(filePath)
			if err != nil {
				return fmt.Errorf("error creating file: %v", err)
			}
			defer file.Close()

			_, err = io.Copy(file, resp.Body)
			if err != nil {
				return fmt.Errorf("error writing image to file: %v", err)
			}

			return nil
		}()

		if err == nil {
			fmt.Printf("🖼️ Saved image to %s\n", filePath)
			return
		}

		if attempt < maxRetries-1 {
			fmt.Printf("Attempt %d failed: %v. Retrying in %v...\n", attempt+1, err, retryDelay)
			time.Sleep(retryDelay)
		} else {
			fmt.Printf("Failed to download image after %d attempts: %v\n", maxRetries, err)
		}
	}
}

func cleanFilename(filename string) string {
	// Remove query parameters
	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}
	return filename
}

func main() {
	asciiArt()
	time.Sleep(5 * time.Second)

	client := &http.Client{Timeout: clientTimeout}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("⛔ Error fetching JSON file: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("⛔ Failed to fetch JSON file: %d\n", resp.StatusCode)
		return
	}

	var jsonData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&jsonData); err != nil {
		fmt.Printf("⛔ Error decoding JSON: %v\n", err)
		return
	}

	data, ok := jsonData["data"].(map[string]interface{})
	if !ok {
		fmt.Println("⛔ JSON does not have a \"data\" property at its root.")
		return
	}

	downloadDir := "./images"
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		fmt.Printf("⛔ Error creating directory: %v\n", err)
		return
	}
	fmt.Printf("📁 Created directory: %s\n", downloadDir)

	var wg sync.WaitGroup

	for _, subproperty := range data {
		if subproperty, ok := subproperty.(map[string]interface{}); ok {
			if imageURL, ok := subproperty["dhd"].(string); ok {
				fmt.Println("🔍 Found image URL!")

				filename := filepath.Base(imageURL)
				filename = cleanFilename(filename)

				filePath := filepath.Join(downloadDir, filename)

				wg.Add(1)
				go downloadImage(client, imageURL, filePath, &wg)

				time.Sleep(250 * time.Millisecond)
			}
		}
	}

	wg.Wait()
}

func asciiArt() {
	fmt.Println(`
 /$$      /$$ /$$   /$$ /$$$$$$$   /$$$$$$  /$$$$$$$
| $$$    /$$$| $$  /$$/| $$__  $$ /$$__  $$| $$__  $$
| $$$$  /$$$$| $$ /$$/ | $$  \ $$| $$  \__/| $$  \ $$
| $$ $$/$$ $$| $$$$$/  | $$$$$$$ |  $$$$$$ | $$  | $$
| $$  $$$| $$| $$  $$  | $$__  $$ \____  $$| $$  | $$
| $$\  $ | $$| $$\  $$ | $$  \ $$ /$$  \ $$| $$  | $$
| $$ \/  | $$| $$ \  $$| $$$$$$$/|  $$$$$$/| $$$$$$$/
|__/     |__/|__/  \__/|_______/  \______/ |_______/`)
	fmt.Println("🤑 Starting downloads from your favorite sellout grifter's wallpaper app...")
}
