package main

import (
	"encoding/json"
	"fmt"
	"log"
	"slices"

	"github.com/joho/godotenv"
)

type PlankaUser struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
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
			log.Fatalf("failed to parse JSON: %s", err)
		}
		emails, err := get_planka_users_emails()
		if err != nil {
			log.Fatalf("Error fetching Planka user emails: %v", err)
		}
		for _, user := range users.([]interface{}) {
			if !slices.Contains(emails, user.(map[string]interface{})["email"].(string)) {
				name := user.(map[string]interface{})["full_name"].(string)
				if name == "" {
					name = user.(map[string]interface{})["username"].(string)
				}
				userData := PlankaUser{
					Username: user.(map[string]interface{})["username"].(string),
					Name:     name,
					Email:    user.(map[string]interface{})["email"].(string),
					Password: "1234tempPass",
					Role:     "projectOwner",
				}
				err := create_planka_user(userData)
				if err != nil {
					log.Printf("Error creating Planka user %s: %v", userData.Username, err)
					continue
				}
				fmt.Printf("Created Planka user: %s\n", userData.Username)
			}
		}
	}

}
