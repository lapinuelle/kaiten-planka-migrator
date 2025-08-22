package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func get_planka_users_emails() ([]string, error) {
	var emails []string
	plankaUrl, exists := os.LookupEnv("PLANKA_URL")
	if !exists {
		return nil, fmt.Errorf("PLANKA_URL environment variable is not set")
	}
	plankaToken, exists := os.LookupEnv("PLANKA_TOKEN")
	if !exists {
		return nil, fmt.Errorf("PLANKA_TOKEN environment variable is not set")
	}
	req, err := http.NewRequest("GET", plankaUrl+"/api/users", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+plankaToken)
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
	var users map[string]interface{}
	if err := json.Unmarshal(body, &users); err != nil {
		log.Fatalf("failed to parse JSON: %w", err)
	}
	for _, user := range users["items"].([]interface{}) {
		if email, ok := user.(map[string]interface{})["email"].(string); ok && email != "" {
			emails = append(emails, email)
		} else {
			log.Printf("User %s does not have a valid email", user.(map[string]interface{})["name"])
		}
	}

	return emails, nil
}

func create_planka_user(user PlankaUser) error {
	plankaUrl, exists := os.LookupEnv("PLANKA_URL")
	if !exists {
		return fmt.Errorf("PLANKA_URL environment variable is not set")
	}
	plankaToken, exists := os.LookupEnv("PLANKA_TOKEN")
	if !exists {
		return fmt.Errorf("PLANKA_TOKEN environment variable is not set")
	}

	userJson, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("error marshalling user data: %w", err)
	}
	fmt.Printf("%s\n", userJson)

	req, err := http.NewRequest("POST", plankaUrl+"/api/users", bytes.NewBuffer(userJson))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+plankaToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to create user, status code: %d", resp.StatusCode)
	}

	return nil
}
