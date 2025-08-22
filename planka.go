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

type PlankaProject struct {
	Name        string `json:"name"`
	Description string `json:"desc"`
	Type        string `json:"type"`
}

type PlankaBoard struct {
	Position float64 `json:"position"`
	Name     string  `json:"name"`
}

type CreatedPlankaBoard struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type CreatedProject struct {
	ID             string  `json:"id"`
	Description    string  `json:"desc"`
	Name           string  `json:"name"`
	KaitenSpaceID  float64 `json:"kaiten_space_id"`
	KaitenSpaceUID string  `json:"kaiten_space_uid"`
}

func delete_planka_projects() error {
	availableProjects, err := get_planka_projects()
	if err != nil {
		return fmt.Errorf("error fetching existing projects: %w", err)
	}
	for _, project := range availableProjects {
		plankaUrl, exists := os.LookupEnv("PLANKA_URL")
		if !exists {
			return fmt.Errorf("PLANKA_URL environment variable is not set")
		}
		plankaToken, exists := os.LookupEnv("PLANKA_TOKEN")
		if !exists {
			return fmt.Errorf("PLANKA_TOKEN environment variable is not set")
		}
		req, err := http.NewRequest("DELETE", plankaUrl+"/api/projects/"+project.ID, nil)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", "Bearer "+plankaToken)
		client := &http.Client{}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error sending request:", err)
			return err
		}
		defer resp.Body.Close() // Ensure the response body is closed
	}
	return nil
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

func get_planka_projects() (map[string]CreatedProject, error) {
	plankaUrl, exists := os.LookupEnv("PLANKA_URL")
	if !exists {
		return nil, fmt.Errorf("PLANKA_URL environment variable is not set")
	}
	plankaToken, exists := os.LookupEnv("PLANKA_TOKEN")
	if !exists {
		return nil, fmt.Errorf("PLANKA_TOKEN environment variable is not set")
	}

	req, err := http.NewRequest("GET", plankaUrl+"/api/projects", nil)
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

	var projects map[string]interface{}
	if err := json.Unmarshal(body, &projects); err != nil {
		log.Fatalf("failed to parse JSON: %w", err)
	}
	createdProjects := make(map[string]CreatedProject)
	for _, project := range projects["items"].([]interface{}) {
		if projectMap, ok := project.(map[string]interface{}); ok {
			var projectDescription string
			if projectMap["description"] == nil {
				projectDescription = ""
			} else {
				projectDescription = projectMap["description"].(string)
			}
			createdProjects[projectMap["name"].(string)] = CreatedProject{
				ID:             projectMap["id"].(string),
				Description:    projectDescription,
				Name:           projectMap["name"].(string),
				KaitenSpaceID:  0,
				KaitenSpaceUID: "",
			}
		}
	}

	return createdProjects, nil
}

func create_planka_project(space KaitenSpace) (CreatedProject, error) {
	plankaUrl, exists := os.LookupEnv("PLANKA_URL")
	if !exists {
		return CreatedProject{}, fmt.Errorf("PLANKA_URL environment variable is not set")
	}
	plankaToken, exists := os.LookupEnv("PLANKA_TOKEN")
	if !exists {
		return CreatedProject{}, fmt.Errorf("PLANKA_TOKEN environment variable is not set")
	}

	project := PlankaProject{
		Name:        space.Name,
		Description: "Migrated from Kaiten",
		Type:        "shared",
	}

	projectJson, err := json.Marshal(project)
	if err != nil {
		return CreatedProject{}, fmt.Errorf("error marshalling project data: %w", err)
	}

	availableProjects, err := get_planka_projects()
	if err != nil {
		return CreatedProject{}, fmt.Errorf("error fetching existing projects: %w", err)
	}
	existingProject, exists := availableProjects[project.Name]
	if exists {

		fmt.Printf("Project %s already exists with ID: %s\n", existingProject.Name, existingProject.ID)
		return CreatedProject{
			ID:             existingProject.ID,
			Description:    existingProject.Description,
			Name:           existingProject.Name,
			KaitenSpaceID:  space.ID,
			KaitenSpaceUID: space.UID,
		}, nil
	} else {
		req, err := http.NewRequest("POST", plankaUrl+"/api/projects", bytes.NewBuffer(projectJson))
		if err != nil {
			return CreatedProject{}, err
		}
		req.Header.Add("Authorization", "Bearer "+plankaToken)
		req.Header.Add("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return CreatedProject{}, fmt.Errorf("error sending request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return CreatedProject{}, fmt.Errorf("failed to create project, status code: %d", resp.StatusCode)
		}

		var createdProject CreatedProject
		var unmBody interface{}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return CreatedProject{}, err
		}

		if err := json.Unmarshal(body, &unmBody); err != nil {
			log.Fatalf("failed to parse JSON: %w", err)
		}
		// if err := json.NewDecoder(resp.Body).Decode(&createdProject); err != nil {
		// 	return CreatedProject{}, fmt.Errorf("error decoding response body: %w", err)
		// }
		createdProject.ID = unmBody.(map[string]interface{})["item"].(map[string]interface{})["id"].(string)
		createdProject.Description = ""
		createdProject.Name = unmBody.(map[string]interface{})["item"].(map[string]interface{})["name"].(string)
		createdProject.KaitenSpaceID = space.ID
		createdProject.KaitenSpaceUID = space.UID
		fmt.Printf("%s\n", project)
		return createdProject, nil
	}
}

func create_planka_board(projectId string, board KaitenBoard, prefix string) (CreatedPlankaBoard, error) {
	plankaUrl, exists := os.LookupEnv("PLANKA_URL")
	if !exists {
		return CreatedPlankaBoard{}, fmt.Errorf("PLANKA_URL environment variable is not set")
	}
	plankaToken, exists := os.LookupEnv("PLANKA_TOKEN")
	if !exists {
		return CreatedPlankaBoard{}, fmt.Errorf("PLANKA_TOKEN environment variable is not set")
	}

	boardToCreate := PlankaBoard{
		Name:     prefix + board.Title,
		Position: 0,
	}

	boardJson, err := json.Marshal(boardToCreate)
	if err != nil {
		return CreatedPlankaBoard{}, fmt.Errorf("error marshalling project data: %w", err)
	}
	req, err := http.NewRequest("POST", plankaUrl+"/api/projects/"+projectId+"/boards", bytes.NewBuffer(boardJson))
	if err != nil {
		return CreatedPlankaBoard{}, err
	}
	req.Header.Add("Authorization", "Bearer "+plankaToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	fmt.Printf("%s\n", resp)
	if err != nil {
		return CreatedPlankaBoard{}, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return CreatedPlankaBoard{}, fmt.Errorf("failed to create board, status code: %d", resp.StatusCode)
	}

	var createdBoard CreatedPlankaBoard
	if err := json.NewDecoder(resp.Body).Decode(&createdBoard); err != nil {
		return CreatedPlankaBoard{}, fmt.Errorf("error decoding response body: %w", err)
	}

	// fmt.Printf("%s\n", project)
	return createdBoard, nil

}
