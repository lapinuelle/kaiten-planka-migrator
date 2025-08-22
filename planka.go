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
	ID             string   `json:"id"`
	Description    string   `json:"desc"`
	Name           string   `json:"name"`
	KaitenSpaceID  float64  `json:"kaiten_space_id"`
	KaitenSpaceUID string   `json:"kaiten_space_uid"`
	Boards         []string `json:"boards,omitempty"`
}

func planka_api_call(json_data []byte, url string, call_type string) ([]byte, error) {
	plankaUrl, exists := os.LookupEnv("PLANKA_URL")
	if !exists {
		return nil, fmt.Errorf("PLANKA_URL environment variable is not set")
	}
	plankaToken, exists := os.LookupEnv("PLANKA_TOKEN")
	if !exists {
		return nil, fmt.Errorf("PLANKA_TOKEN environment variable is not set")
	}

	req, err := http.NewRequest(call_type, plankaUrl+url, bytes.NewBuffer(json_data))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+plankaToken)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return nil, err
	}
	//defer resp.Body.Close() // Ensure the response body is closed

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}
	return body, nil
}

func delete_planka_projects() error {
	availableProjects, err := get_planka_projects()
	if err != nil {
		return fmt.Errorf("error fetching existing projects: %w", err)
	}

	for _, project := range availableProjects {
		boards, err := get_planka_boards_for_project(project.ID)
		if err != nil {
			return fmt.Errorf("error fetching boards for project %s: %w", project.Name, err)
		}
		for _, boardID := range boards {
			_, err = planka_api_call(nil, "/api/boards/"+boardID, "DELETE")
			if err != nil {
				return fmt.Errorf("error deleting board %s in project %s: %w", boardID, project.Name, err)
			}
			fmt.Printf("Deleted board %s in project %s\n", boardID, project.Name)
		}
		_, err = planka_api_call(nil, "/api/projects/"+project.ID, "DELETE")
		if err != nil {
			return err
		}
	}
	return nil
}

func get_planka_users_emails() ([]string, error) {
	var emails []string
	body, err := planka_api_call(nil, "/api/users", "GET")
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
	userJson, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("error marshalling user data: %w", err)
	}
	body, err := planka_api_call(userJson, "/api/users", "POST")

	if body == nil && err != nil {
		return fmt.Errorf("failed to create user")
	}

	return nil
}

func get_planka_boards_for_project(projectId string) ([]string, error) {
	body, err := planka_api_call(nil, "/api/projects/"+projectId, "GET")
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}
	var projectMap map[string]interface{}
	if err := json.Unmarshal(body, &projectMap); err != nil {
		log.Fatalf("failed to parse JSON: %w", err)
	}
	var boards []string
	board_slice := projectMap["included"].(map[string]interface{})["boards"].([]interface{})
	for _, board := range board_slice {
		boards = append(boards, board.(map[string]interface{})["id"].(string))
	}
	return boards, nil
}

func get_planka_projects() (map[string]CreatedProject, error) {
	body, err := planka_api_call(nil, "/api/projects", "GET")
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
		body, err := planka_api_call(projectJson, "/api/projects", "POST")
		if body == nil && err != nil {
			return CreatedProject{}, fmt.Errorf("failed to create project")
		}

		var createdProject CreatedProject
		var unmBody interface{}

		if err != nil {
			fmt.Println("Error reading response body:", err)
			return CreatedProject{}, err
		}

		if err := json.Unmarshal(body, &unmBody); err != nil {
			log.Fatalf("failed to parse JSON: %w", err)
		}
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
	boardToCreate := PlankaBoard{
		Name:     prefix + board.Title,
		Position: 0,
	}

	boardJson, err := json.Marshal(boardToCreate)
	if err != nil {
		return CreatedPlankaBoard{}, fmt.Errorf("error marshalling project data: %w", err)
	}
	body, err := planka_api_call(boardJson, "/api/projects/"+projectId+"/boards", "POST")
	if body == nil && err != nil {
		return CreatedPlankaBoard{}, fmt.Errorf("failed to create user")
	}

	var createdBoard CreatedPlankaBoard
	err = json.Unmarshal(body, &createdBoard)
	if err != nil {
		return CreatedPlankaBoard{}, fmt.Errorf("error unmarshalling response body: %w", err)
	}
	return createdBoard, nil

}
