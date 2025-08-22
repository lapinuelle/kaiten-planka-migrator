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
	delete_planka_projects()

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
	spaces, err := get_kaiten_spaces()
	if err != nil {
		log.Fatalf("Error fetching Kaiten spaces: %v", err)
	}
	plankaProjects := make(map[string]CreatedProject)
	for _, space := range spaces {
		if err != nil {
			log.Fatalf("Error getting boards for space")
		}

		if space.ParentID == "" {
			// fmt.Printf("Project to be created: %s\n", space.Name)
			plankaProject, err := create_planka_project(space)
			plankaProjects[plankaProject.KaitenSpaceUID] = plankaProject
			if err != nil {
				log.Printf("Error creating Planka project for space %s: %v", space.Name, err)
				continue
			}
			fmt.Printf("Planka project: %s with ID: %s\n", plankaProject.Name, plankaProject.ID)

			// Here you can add code to create columns
		}
	}
	for _, space := range spaces {
		boardTitlePrefix := ""
		boards, err := get_kaiten_boards_for_space(space)
		if err != nil {
			log.Fatalf("Error getting boards for space")
		}
		if len(boards) > 1 {
			boardTitlePrefix = space.Name + ": "
			fmt.Printf("%s\n", boardTitlePrefix)
		}

		spaceIdforBoard := space.UID
		if space.ParentID != "" {
			spaceIdforBoard = space.ParentID
		}
		spaceUIDforBoardCreation := spaces[spaceIdforBoard].UID
		for _, kaiten_board := range boards {
			if len(boards) < 2 {
				kaiten_board.Title = space.Name
			}
			fmt.Printf("Board named %s created in project %s\n", boardTitlePrefix+kaiten_board.Title, plankaProjects[spaceUIDforBoardCreation].Name)
			board, err := create_planka_board(plankaProjects[spaceUIDforBoardCreation].ID, kaiten_board, boardTitlePrefix)
			if err != nil {
				log.Printf("Error creating Planka board for project %s: %v", plankaProjects[spaceUIDforBoardCreation].ID, err)
				continue
			}
			fmt.Printf("%s\n", board)
		}

	}

}
