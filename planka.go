package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

var colors = []string{
	"light-mud",
	"piggy-red",
	"pink-tulip",
	"lavender-fields",
	"sugar-plum",
	"antique-blue",
	"morning-sky",
	"summer-sky",
	"french-coast",
	"turquoise-sea",
	"tank-green",
	"bright-moss",
	"fresh-salad",
	"desert-sand",
	"apricot-red",
	"dark-granite",
	"light-concrete",
	"light-mud",
}

type PlankaUser struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type PlankaProject struct {
	Name           string   `json:"name"`
	Description    string   `json:"desc"`
	Type           string   `json:"type"`
	ID             string   `json:"id,omitempty"`
	KaitenSpaceID  float64  `json:"kaiten_space_id"`
	KaitenSpaceUID string   `json:"kaiten_space_uid"`
	Boards         []string `json:"boards,omitempty"`
}

type PlankaCardMember struct {
	UserId string `json:"userId"`
}

type PlankaBoard struct {
	Position float64 `json:"position"`
	Name     string  `json:"name"`
	ID       string  `json:"id,omitempty"`
}

type PlankaList struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PlankaBoardMember struct {
	UserId     string `json:"userId"`
	Role       string `json:"role"`       //whitelist (editor, viewer)
	CanComment *bool  `json:"canComment"` //permisao para comentar
}

type PlankaCard struct {
	Position    float64 `json:"position"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
	Start       string  `json:"start,omitempty"`
	DueDate     string  `json:"dueDate,omitempty"`
}

type PlankaUserCreds struct {
	Email    string `json:"emailOrUsername"`
	Password string `json:"password"`
}

type PlankaTaskList struct {
	Position float64 `json:"position"`
	Name     string  `json:"name"`
}

type PlankaTask struct {
	Position    float64 `json:"position"`
	Name        string  `json:"name"`
	IsCompleted bool    `json:"isCompleted"`
}

type PlankaLabel struct {
	Position float64 `json:"position"`
	Name     string  `json:"name"`
	Color    string  `json:"color"`
	Id       string  `json:"id,omitempty"`
}

type PlankaLabelForCard struct {
	ID string `json:"labelId"`
}

func getEnv(name string) (string, error) {
	val, exists := os.LookupEnv(name)
	if !exists {
		return "", fmt.Errorf("%s environment variable is not set", name)
	}
	return val, nil
}

func plankaAPICall(json_data []byte, endpoint string, method string) ([]byte, error) {

	plankaUrl, err := getEnv("PLANKA_URL")
	if err != nil {
		return nil, err
	}
	plankaToken, err := getEnv("PLANKA_TOKEN")
	if err != nil {
		return nil, err
	}

	validMethods := map[string]bool{
		"GET":    true,
		"POST":   true,
		"PUT":    true,
		"DELETE": true,
		"PATCH":  true,
	}
	if !validMethods[method] {
		return nil, fmt.Errorf("invalid HTTP method: %s", method)
	}

	var req *http.Request
	if method == "GET" {
		req, err = http.NewRequest(method, plankaUrl+endpoint, nil)
	} else {
		req, err = http.NewRequest(method, plankaUrl+endpoint, bytes.NewBuffer(json_data))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+plankaToken)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, body)
	}

	return body, nil
}

func plankaUploadFile(filepath string, url string, filename string) ([]byte, error) {
	plankaUrl, err := getEnv("PLANKA_URL")
	if err != nil {
		return nil, err
	}
	plankaToken, err := getEnv("PLANKA_TOKEN")
	if err != nil {
		return nil, err
	}
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("Can't open file")
	}
	defer file.Close()
	requestBody := &bytes.Buffer{}
	writer := multipart.NewWriter(requestBody)
	writer.WriteField("name", filename)
	writer.WriteField("type", "file")
	part, err := writer.CreateFormFile("file", filepath) // "file" is the form field name, "file.txt" is the filename in the request

	if err != nil {
		return nil, fmt.Errorf("Can't create writter from file")
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("Can't copy file to part")
	}
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("Can't close writer")
	}
	req, err := http.NewRequest("POST", plankaUrl+url, requestBody)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())
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

func plankaAPICallByUser(jsonPayload []byte, url string, method string, token string) ([]byte, error) {
	plankaUrl, exists := os.LookupEnv("PLANKA_URL")
	if !exists {
		return nil, fmt.Errorf("PLANKA_URL environment variable is not set")
	}

	req, err := http.NewRequest(method, plankaUrl+url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

func plankaDeleteUser() error {
	emails, err := get_planka_users_emails()
	if err != nil {
		return fmt.Errorf("error fetching Planka user emails: %w", err)
	}

	adminEmail, exists := os.LookupEnv("ADMIN_EMAIL")
	if !exists {
		return fmt.Errorf("ADMIN_EMAIL environment variable is not set")
	}

	var validEmails []string
	for _, email := range emails {
		if email != "" {
			validEmails = append(validEmails, email)
		} else {
			log.Println("Skipping empty email")
		}
	}

	for _, email := range validEmails {
		if email == adminEmail {
			log.Printf("Skipping admin user with email %s\n", email)
			continue
		}

		userID, err := get_planka_userId_by_email(email)
		if err != nil {
			return fmt.Errorf("error fetching Planka user ID for email %s: %w", email, err)
		}

		respBody, err := plankaAPICall(nil, "/api/users/"+userID, "DELETE")
		if err != nil {
			return fmt.Errorf("error deleting Planka user with email %s: %w", email, err)
		}

		log.Printf("Deleted user with email %s (Response: %s)\n", email, string(respBody))
	}
	return nil
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
			_, err = plankaAPICall(nil, "/api/boards/"+boardID, "DELETE")
			if err != nil {
				return fmt.Errorf("error deleting board %s in project %s: %w", boardID, project.Name, err)
			}
			fmt.Printf("Deleted board %s in project %s\n", boardID, project.Name)
		}
		_, err = plankaAPICall(nil, "/api/projects/"+project.ID, "DELETE")
		if err != nil {
			return err
		}
	}
	return nil
}

func get_planka_users_emails() ([]string, error) {
	var emails []string
	body, err := plankaAPICall(nil, "/api/users", "GET")
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

func get_planka_userId_by_email(email string) (string, error) {
	body, err := plankaAPICall(nil, "/api/users", "GET")
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return "", err
	}
	var users map[string]interface{}
	if err := json.Unmarshal(body, &users); err != nil {
		log.Fatalf("failed to parse JSON: %w", err)
	}
	for _, user := range users["items"].([]interface{}) {
		if user.(map[string]interface{})["email"].(string) == email {
			if id, ok := user.(map[string]interface{})["id"].(string); ok {
				return id, nil
			} else {
				log.Printf("User %s does not have a valid id", user.(map[string]interface{})["name"])
			}
		}
	}
	return "", nil
}

func create_planka_user(user PlankaUser) error {
	userJson, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("error marshalling user data: %w", err)
	}
	body, err := plankaAPICall(userJson, "/api/users", "POST")

	if body == nil && err != nil {
		return fmt.Errorf("failed to create user")
	}

	return nil
}

func get_planka_boards_for_project(projectId string) ([]string, error) {
	body, err := plankaAPICall(nil, "/api/projects/"+projectId, "GET")
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

func get_planka_projects() (map[string]PlankaProject, error) {
	body, err := plankaAPICall(nil, "/api/projects", "GET")
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	var projects map[string]interface{}
	if err := json.Unmarshal(body, &projects); err != nil {
		log.Fatalf("failed to parse JSON: %w", err)
	}
	createdProjects := make(map[string]PlankaProject)
	for _, project := range projects["items"].([]interface{}) {
		if projectMap, ok := project.(map[string]interface{}); ok {

			var projectDescription string
			if projectMap["description"] == nil {
				projectDescription = ""
			} else {
				projectDescription = projectMap["description"].(string)
			}
			createdProjects[projectMap["name"].(string)] = PlankaProject{
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

func create_planka_project(space KaitenSpace) (PlankaProject, error) {
	project := PlankaProject{
		Name:        space.Name,
		Description: "Migrated from Kaiten",
		Type:        "shared",
	}

	projectJson, err := json.Marshal(project)
	if err != nil {
		return PlankaProject{}, fmt.Errorf("error marshalling project data: %w", err)
	}

	availableProjects, err := get_planka_projects()
	if err != nil {
		return PlankaProject{}, fmt.Errorf("error fetching existing projects: %w", err)
	}
	existingProject, exists := availableProjects[project.Name]
	if exists {

		fmt.Printf("Project %s already exists with ID: %s\n", existingProject.Name, existingProject.ID)
		return PlankaProject{
			ID:             existingProject.ID,
			Description:    existingProject.Description,
			Name:           existingProject.Name,
			KaitenSpaceID:  space.ID,
			KaitenSpaceUID: space.UID,
		}, nil
	} else {
		body, err := plankaAPICall(projectJson, "/api/projects", "POST")
		if body == nil && err != nil {
			return PlankaProject{}, fmt.Errorf("failed to create project")
		}

		var createdProject PlankaProject
		var unmBody interface{}

		if err != nil {
			fmt.Println("Error reading response body:", err)
			return PlankaProject{}, err
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

func create_planka_board(projectId string, board KaitenBoard, prefix string) (PlankaBoard, error) {
	boardToCreate := PlankaBoard{
		Name:     prefix + board.Title,
		Position: 0,
	}

	boardJson, err := json.Marshal(boardToCreate)
	if err != nil {
		return PlankaBoard{}, fmt.Errorf("error marshalling project data: %w", err)
	}
	body, err := plankaAPICall(boardJson, "/api/projects/"+projectId+"/boards", "POST")
	if body == nil && err != nil {
		return PlankaBoard{}, fmt.Errorf("failed to create user")
	}
	var createdBoardItem interface{}
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return PlankaBoard{}, err
	}
	err = json.Unmarshal(body, &createdBoardItem)
	if err != nil {
		return PlankaBoard{}, fmt.Errorf("error unmarshalling response body: %w", err)
	}
	var createdBoard PlankaBoard
	createdBoard.ID = createdBoardItem.(map[string]interface{})["item"].(map[string]interface{})["id"].(string)
	createdBoard.Name = createdBoardItem.(map[string]interface{})["item"].(map[string]interface{})["name"].(string)
	return createdBoard, nil

}

func set_planka_board_member(boardId string, member string) error {
	var boardMember PlankaBoardMember
	boardMember.UserId = member
	boardMember.Role = "editor"
	canComment := true
	boardMember.CanComment = &canComment
	memberJson, err := json.Marshal(boardMember)
	if err != nil {
		return fmt.Errorf("error marshalling list data: %w", err)
	}
	body, err := plankaAPICall(memberJson, "/api/boards/"+boardId+"/board-memberships", "POST")
	if body == nil && err != nil {
		return fmt.Errorf("failed to create list")
	}
	return nil
}

func create_planka_list(boardId string, column KaitenColumn) (PlankaList, error) {

	listJson, err := json.Marshal(column)
	if err != nil {
		return PlankaList{}, fmt.Errorf("error marshalling list data: %w", err)
	}

	body, err := plankaAPICall(listJson, "/api/boards/"+boardId+"/lists", "POST")
	if body == nil && err != nil {
		return PlankaList{}, fmt.Errorf("failed to create list")
	}
	var createdListItem interface{}
	err = json.Unmarshal(body, &createdListItem)
	if err != nil {
		return PlankaList{}, fmt.Errorf("error unmarshalling response body: %w", err)
	}
	var createdList PlankaList = PlankaList{
		ID:   createdListItem.(map[string]interface{})["item"].(map[string]interface{})["id"].(string),
		Name: createdListItem.(map[string]interface{})["item"].(map[string]interface{})["name"].(string),
	}
	return createdList, nil
}

func set_planka_card_member(cardId string, member string) error {
	var boardMember PlankaCardMember
	boardMember.UserId = member

	memberJson, err := json.Marshal(boardMember)
	if err != nil {
		return fmt.Errorf("error marshalling list data: %w", err)
	}
	body, err := plankaAPICall(memberJson, "/api/cards/"+cardId+"/card-memberships", "POST")
	if body == nil && err != nil {
		return fmt.Errorf("failed to create list")
	}
	return nil
}

func create_planka_card(listId string, card KaitenCard) (string, error) {
	var plankaCard PlankaCard
	plankaCard.Name = card.Title
	plankaCard.Description = card.Description
	plankaCard.Position = card.SortOrder
	plankaCard.Type = "project"

	if card.DueDate != "" {
		plankaCard.DueDate = card.DueDate
	} else {
		if card.StartDate != "" {
			plankaCard.Start = card.StartDate
		}
		if card.EndDate != "" {
			plankaCard.DueDate = card.EndDate
		}
	}
	cardJson, err := json.Marshal(plankaCard)
	if err != nil {
		return "", fmt.Errorf("error marshalling card data: %w", err)
	}
	body, err := plankaAPICall(cardJson, "/api/lists/"+listId+"/cards", "POST")
	if body == nil && err != nil {
		return "", fmt.Errorf("failed to create card")
	}
	var bodyInterface interface{}
	err = json.Unmarshal(body, &bodyInterface)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling response body: %w", err)
	}
	return bodyInterface.(map[string]interface{})["item"].(map[string]interface{})["id"].(string), nil
}

func get_planka_access_token(email string) (string, error) {
	admin_email, exists := os.LookupEnv("ADMIN_EMAIL")
	if !exists {
		return "", fmt.Errorf("PLANKA_URL environment variable is not set")
	}
	var user PlankaUserCreds
	user.Email = email
	if email == admin_email {
		admin_password, exists := os.LookupEnv("ADMIN_PASSWORD")
		if !exists {
			return "", fmt.Errorf("ADMIN_PASSWORD environment variable is not set")
		}
		user.Password = admin_password
	} else {
		user.Password = "1234tempPass"
	}

	userJson, err := json.Marshal(user)
	if err != nil {
		return "", fmt.Errorf("error marshalling user data: %w", err)
	}
	body, err := plankaAPICall(userJson, "/api/access-tokens", "POST")
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return "", err
	}
	var tokenResponse map[string]interface{}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		log.Fatalf("failed to parse JSON: %w", err)
	}
	if token, ok := tokenResponse["item"].(string); ok && token != "" {
		return token, nil
	} else {
		return "", fmt.Errorf("no token found for email %s", email)
	}
}

func create_planka_card_comment(cardId string, comment KaitenComment) error {
	token, err := get_planka_access_token(comment.AuthorEmail)
	if err != nil {
		fmt.Printf("error getting Planka access token for email %s: %w", comment.AuthorEmail, err)
		admin_email, exists := os.LookupEnv("ADMIN_EMAIL")
		if !exists {
			return fmt.Errorf("PLANKA_URL environment variable is not set")
		}
		token, err = get_planka_access_token(admin_email)
		if err != nil {
			fmt.Errorf("error getting Planka access token for email %s: %w", comment.AuthorEmail, err)
			return err
		}
	}
	fmt.Printf("Using token %s for email %s\n", token, comment.AuthorEmail)
	id, err := get_planka_userId_by_email(comment.AuthorEmail)
	if err != nil {
		return fmt.Errorf("error getting Planka user ID for email %s: %w", comment.AuthorEmail, err)
	}
	commentJson, err := json.Marshal(map[string]string{
		"text":   comment.Text,
		"userId": id,
	})
	if err != nil {
		return fmt.Errorf("error marshalling comment data: %w", err)
	}
	body, err := plankaAPICallByUser(commentJson, "/api/cards/"+cardId+"/comments", "POST", token)
	if body == nil && err != nil {
		return fmt.Errorf("failed to create comment")
	}
	return nil
}

func create_planka_card_attachment(cardId string, attachment KaitenAttachment) (string, error) {
	outputFileName := attachment.Name
	// Create the output file
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return "", err
	}

	// Make an HTTP GET request to the URL
	response, err := http.Get(attachment.URL)
	if err != nil {
		fmt.Printf("Error making HTTP request: %v\n", err)
		return "", err
	}
	defer response.Body.Close() // Ensure the response body is closed

	// Check for a successful HTTP status code
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Bad status code: %d %s\n", response.StatusCode, response.Status)
		return "", fmt.Errorf("bad status code: %d", response.StatusCode)
	}

	// Copy the response body to the local file
	_, err = io.Copy(outputFile, response.Body)
	if err != nil {
		fmt.Printf("Error copying data to file: %v\n", err)
		return "", err
	}
	body, err := plankaUploadFile(outputFileName, "/api/cards/"+cardId+"/attachments", attachment.Name)
	if body == nil && err != nil {
		return "", fmt.Errorf("failed to upload attachment")
	}
	fmt.Printf("File downloaded successfully to %s\n", outputFileName)
	defer outputFile.Close() // Ensure the file is closed
	err = os.Remove(outputFileName)
	if err != nil {
		fmt.Printf("Error deleting file: %v\n", err)
		return "", err
	}

	return "", nil
}

func create_planka_card_tasklist(card_id string, checklist KaitenChecklist) (string, error) {
	var tasklist PlankaTaskList
	tasklist.Name = checklist.Name
	tasklist.Position = 0
	json_payload, err := json.Marshal(tasklist)
	if err != nil {
		fmt.Printf("Error marshalling task list to json")
		return "", err
	}
	body, err := plankaAPICall(json_payload, "/api/cards/"+card_id+"/task-lists", "POST")
	if err != nil {
		fmt.Printf("Error sending request to create tasklist")
		return "", err
	}
	var response_json map[string]interface{}
	if err := json.Unmarshal(body, &response_json); err != nil {
		log.Fatalf("failed to parse JSON: %w", err)
	}
	return response_json["item"].(map[string]interface{})["id"].(string), nil
}

func create_planka_task_in_tasklist(list_id string, item KaitenChecklistItem) (string, error) {
	var task PlankaTask
	task.Name = item.Text
	task.Position = 0
	task.IsCompleted = item.Checked
	json_payload, err := json.Marshal(task)
	if err != nil {
		fmt.Printf("Error marshalling task list to json")
		return "", err
	}
	body, err := plankaAPICall(json_payload, "/api/task-lists/"+list_id+"/tasks", "POST")
	if err != nil {
		fmt.Printf("Error sending request to create task")
		return "", err
	}
	var response_json map[string]interface{}
	if err := json.Unmarshal(body, &response_json); err != nil {
		log.Fatalf("failed to parse JSON: %w", err)
	}
	return response_json["item"].(map[string]interface{})["id"].(string), nil
}

func create_planka_label_for_board(boardId string, tag KaitenTag) (PlankaLabel, error) {
	var labelToCreate PlankaLabel
	labelToCreate.Name = tag.Name
	labelToCreate.Color = colors[int(tag.Color)]
	labelToCreate.Position = 0
	json_payload, err := json.Marshal(labelToCreate)
	if err != nil {
		fmt.Printf("Error marshalling task list to json")
		return PlankaLabel{}, err
	}
	body, err := plankaAPICall(json_payload, "/api/boards/"+boardId+"/labels", "POST")
	if err != nil {
		fmt.Printf("Error sending request to create task")
		return PlankaLabel{}, err
	}
	var response_json map[string]interface{}
	if err := json.Unmarshal(body, &response_json); err != nil {
		log.Fatalf("failed to parse JSON: %w", err)
	}
	labelToCreate.Id = response_json["item"].(map[string]interface{})["id"].(string)
	return labelToCreate, nil
}

func create_planka_label_for_card(card_id string, label_id string) error {
	var lbl PlankaLabelForCard
	lbl.ID = label_id
	json_payload, err := json.Marshal(lbl)
	if err != nil {
		fmt.Printf("Error marshalling task list to json")
		return err
	}
	body, err := plankaAPICall(json_payload, "/api/cards/"+card_id+"/card-labels", "POST")
	if err != nil {
		fmt.Printf("Error sending request to create task")
		return err
	}
	log.Printf("%s\n", body)
	return nil
}
