package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"

	"github.com/joho/godotenv"
)

type Space struct {
	Id              uint   `json:"id"`
	Title           string `json:"title"`
	ParentEntityUid string `json:"parent_entity_uid"`
	BoardId         uint   `json:"board_id"`
}

type PlankaUser struct {
	Username            string `json:"username"`
	Name                string `json:"name"`
	Email               string `json:"email"`
	Password            string `json:"password"`
	Phone               string `json:"phone"`
	Organization        string `json:"organization"`
	Language            string `json:"language"`
	SubscribeToOwnCards bool   `json:"subscribeToOwnCards"`
}

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

	req, err := http.NewRequest("POST", plankaUrl+"/api/users", bytes.NewBuffer(userJson))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+plankaToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	fmt.Printf("%s\n", req)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create user, status code: %d", resp.StatusCode)
	}

	return nil
}

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {

	raw_users, err := get_kaiten_users()

	{
		if err != nil {
			log.Fatalf("Error fetching Kaiten users: %v", err)
		}
		var users interface{}
		if err := json.Unmarshal(raw_users.([]byte), &users); err != nil {
			log.Fatalf("failed to parse JSON: %w", err)
		}
		fmt.Printf("Users: %d\n", len(users.([]interface{})))
		emails, err := get_planka_users_emails()
		if err != nil {
			log.Fatalf("Error fetching Planka user emails: %v", err)
		}
		fmt.Printf("Emails: %d\n", len(emails))
		for _, user := range users.([]interface{}) {
			if !slices.Contains(emails, user.(map[string]interface{})["email"].(string)) {
				userData := PlankaUser{
					Username:            user.(map[string]interface{})["username"].(string),
					Name:                user.(map[string]interface{})["full_name"].(string),
					Email:               user.(map[string]interface{})["email"].(string),
					Password:            "1234tempPass", // Default password, should be changed later
					Phone:               "1234567890",
					Organization:        "IPPM",
					Language:            "ru-RU",
					SubscribeToOwnCards: false,
				}
				err := create_planka_user(userData)
				if err != nil {
					log.Printf("Error creating Planka user %s: %v", userData.Username, err)
					continue
				}
				fmt.Printf("Created Planka user: %s\n", userData.Username)

				fmt.Printf("User: %s, Email: %s\n", user.(map[string]interface{})["username"].(string), user.(map[string]interface{})["email"].(string))
			}
		}
	}

}
