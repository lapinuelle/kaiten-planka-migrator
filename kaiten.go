package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func get_kaiten_users() (interface{}, error) {
	kaitenUrl, exists := os.LookupEnv("KAITEN_URL")
	if !exists {
		return nil, fmt.Errorf("KAITEN_URL environment variable is not set")
	}
	kaitenToken, exists := os.LookupEnv("KAITEN_TOKEN")
	if !exists {
		return nil, fmt.Errorf("KAITEN_TOKEN environment variable is not set")
	}
	req, err := http.NewRequest("GET", kaitenUrl+"/api/latest/company/users", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+kaitenToken)
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return nil, err
	}
	defer resp.Body.Close() // Ensure the response body is closed

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	return body, err
}
