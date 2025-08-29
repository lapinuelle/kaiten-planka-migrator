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
	"strings"
	"sync"
)

var PlankaColors = []string{
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

var (
	// Planka environment variables cache
	plankaURL   string
	plankaToken string
	plankaOnce  sync.Once
	plankaErr   error
)

// initPlankaEnv initializes Planka environment variables once
func initPlankaEnv() error {
	plankaOnce.Do(func() {
		plankaURL, plankaErr = getEnv("PLANKA_URL")
		if plankaErr != nil {
			plankaErr = fmt.Errorf("failed to get PLANKA_URL: %w", plankaErr)
			return
		}

		plankaToken, plankaErr = getEnv("PLANKA_TOKEN")
		if plankaErr != nil {
			plankaErr = fmt.Errorf("failed to get PLANKA_TOKEN: %w", plankaErr)
			return
		}

		// Normalize URL
		plankaURL = strings.TrimRight(plankaURL, "/")
	})
	return plankaErr
}

func plankaAPICall(jsonPayload []byte, endpoint string, method string) ([]byte, error) {

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
	var err error
	if method == "GET" {
		req, err = http.NewRequest(method, plankaURL+endpoint, nil)
	} else {
		req, err = http.NewRequest(method, plankaURL+endpoint, bytes.NewBuffer(jsonPayload))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+plankaToken)

	client := clientPool.Get().(*http.Client)
	defer clientPool.Put(client)

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
		return nil, fmt.Errorf("failed to get PLANKA_URL: %w", err)
	}
	plankaToken, err := getEnv("PLANKA_TOKEN")
	if err != nil {
		return nil, fmt.Errorf("failed to get PLANKA_TOKEN: %w", err)
	}

	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	requestBody := &bytes.Buffer{}
	writer := multipart.NewWriter(requestBody)

	if err := writer.WriteField("name", filename); err != nil {
		return nil, fmt.Errorf("failed to write field 'name': %w", err)
	}
	if err := writer.WriteField("type", "file"); err != nil {
		return nil, fmt.Errorf("failed to write field 'type': %w", err)
	}

	part, err := writer.CreateFormFile("file", filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file part: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file content to part: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", plankaUrl+url, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+plankaToken)

	client := clientPool.Get().(*http.Client)
	defer clientPool.Put(client)

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

	client := clientPool.Get().(*http.Client)
	defer clientPool.Put(client)

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
	emails, err := getPlankaUsersMails()
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

		userID, err := getPlankaUserIDByEmail(email)
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

func deletePlankaProjects() error {
	projects, err := getPlankaProjects()
	if err != nil {
		return fmt.Errorf("error fetching projects: %w", err)
	}

	for _, project := range projects {
		log.Printf("Processing project: %s (ID: %s)", project.Name, project.ID)

		boards, err := getPlankaBoardsForProject(project.ID)
		if err != nil {
			log.Printf("Skipping project %s due to error fetching boards: %v", project.Name, err)
			continue
		}

		for _, boardID := range boards {
			_, err := plankaAPICall(nil, "/api/boards/"+boardID, "DELETE")
			if err != nil {
				log.Printf("Failed to delete board %s in project %s: %v", boardID, project.Name, err)
				continue
			}
			log.Printf("Deleted board %s in project %s", boardID, project.Name)
		}

		_, err = plankaAPICall(nil, "/api/projects/"+project.ID, "DELETE")
		if err != nil {
			log.Printf("Failed to delete project %s: %v", project.Name, err)
			continue
		}
		log.Printf("Deleted project %s", project.Name)
	}
	return nil
}

func getPlankaUsersMails() ([]string, error) {
	var emails []string
	body, err := plankaAPICall(nil, "/api/users", "GET")
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)

	}
	var users map[string]interface{}
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)

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

func getPlankaUserIDByEmail(email string) (string, error) {
	body, err := plankaAPICall(nil, "/api/users", "GET")
	if err != nil {
		return "", fmt.Errorf("failed to fetch users: %w", err)
	}

	type User struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	type UsersResponse struct {
		Items []User `json:"items"`
	}

	var response UsersResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	for _, user := range response.Items {
		if user.Email == email {
			return user.ID, nil
		}
	}
	return "", fmt.Errorf("user with email %s not found", email)
}

func createPlankaUser(user PlankaUser) error {
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

func getPlankaBoardsForProject(projectId string) ([]string, error) {
	body, err := plankaAPICall(nil, "/api/projects/"+projectId, "GET")
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	var projectMap map[string]interface{}
	if err := json.Unmarshal(body, &projectMap); err != nil {
		return nil, err
	}
	var boards []string
	board_slice := projectMap["included"].(map[string]interface{})["boards"].([]interface{})
	for _, board := range board_slice {
		boards = append(boards, board.(map[string]interface{})["id"].(string))
	}
	return boards, nil
}

func getPlankaProjects() (map[string]PlankaProject, error) {
	body, err := plankaAPICall(nil, "/api/projects", "GET")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch projects: %w", err)
	}

	type ProjectItem struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}

	type ProjectsResponse struct {
		Items []ProjectItem `json:"items"`
	}

	var response ProjectsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	createdProjects := make(map[string]PlankaProject)
	for _, item := range response.Items {
		createdProjects[item.Name] = PlankaProject{
			ID:             item.ID,
			Description:    item.Description,
			Name:           item.Name,
			KaitenSpaceID:  0,
			KaitenSpaceUID: "",
		}
	}
	return createdProjects, nil
}

func createPlankaProject(space KaitenSpace) (PlankaProject, error) {
	project := PlankaProject{
		Name:        space.Name,
		Description: "Migrated from Kaiten",
		Type:        "shared",
	}

	projectJson, err := json.Marshal(project)
	if err != nil {
		return PlankaProject{}, fmt.Errorf("error marshalling project data: %w", err)
	}

	availableProjects, err := getPlankaProjects()
	if err != nil {
		return PlankaProject{}, fmt.Errorf("error fetching existing projects: %w", err)
	}
	existingProject, exists := availableProjects[project.Name]
	if exists {

		log.Printf("Project %s already exists with ID: %s\n", existingProject.Name, existingProject.ID)
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
			return PlankaProject{}, fmt.Errorf("error reading response body: %w", err)
		}

		if err := json.Unmarshal(body, &unmBody); err != nil {
			log.Fatalf("failed to parse JSON: %w", err)
		}
		createdProject.ID = unmBody.(map[string]interface{})["item"].(map[string]interface{})["id"].(string)
		createdProject.Description = ""
		createdProject.Name = unmBody.(map[string]interface{})["item"].(map[string]interface{})["name"].(string)
		createdProject.KaitenSpaceID = space.ID
		createdProject.KaitenSpaceUID = space.UID

		return createdProject, nil
	}
}

func createPlankaBoard(projectId string, board KaitenBoard, prefix string) (PlankaBoard, error) {
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
		return PlankaBoard{}, fmt.Errorf("error reading response body: %w", err)
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

func setPlankaBoardMember(boardId string, member string) error {
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

func createPlankaList(boardId string, column KaitenColumn) (PlankaList, error) {

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

func setPlankaCardNumber(cardId string, member string) error {
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

func createPlankaCard(listId string, card KaitenCard) (string, error) {
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

func getPlankaAccessToken(email string) (string, error) {
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
		return "", fmt.Errorf("error reading response body: %w", err)
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

func createPlankaCommentForCard(cardId string, comment KaitenComment) error {
	token, err := getPlankaAccessToken(comment.AuthorEmail)
	if err != nil {
		log.Printf("error getting Planka access token for email %s: %w", comment.AuthorEmail, err)
		admin_email, exists := os.LookupEnv("ADMIN_EMAIL")
		if !exists {
			return fmt.Errorf("PLANKA_URL environment variable is not set")
		}
		token, err = getPlankaAccessToken(admin_email)
		if err != nil {
			return fmt.Errorf("error getting Planka access token for email %s: %w", comment.AuthorEmail, err)
		}
	}
	log.Printf("Using token %s for email %s\n", token, comment.AuthorEmail)
	id, err := getPlankaUserIDByEmail(comment.AuthorEmail)
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
		fmt.Println("Response body:", body)
		return fmt.Errorf("failed to create comment")
	}
	return nil
}

func createPlankaAttachmentForCard(cardId string, attachment KaitenAttachment) (string, error) {
	outputFileName := attachment.Name
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return "", fmt.Errorf("error creating file: %w", err)
	}

	response, err := http.Get(attachment.URL)
	if err != nil {
		return "", fmt.Errorf("error making HTTP request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status code: %d", response.StatusCode)
	}

	_, err = io.Copy(outputFile, response.Body)
	if err != nil {
		return "", fmt.Errorf("error copying data to file: %w", err)
	}
	body, err := plankaUploadFile(outputFileName, "/api/cards/"+cardId+"/attachments", attachment.Name)
	if body == nil && err != nil {
		return "", fmt.Errorf("failed to upload attachment")
	}
	log.Printf("File downloaded successfully to %s\n", outputFileName)
	defer outputFile.Close()
	err = os.Remove(outputFileName)
	if err != nil {
		return "", fmt.Errorf("error deleting file: %w", err)
	}

	return "", nil
}

func createPlankaTasklistForCard(cardId string, checklist KaitenChecklist) (string, error) {
	var tasklist PlankaTaskList
	tasklist.Name = checklist.Name
	tasklist.Position = 0
	jsonPayload, err := json.Marshal(tasklist)
	if err != nil {
		return "", fmt.Errorf("rrror marshalling task list to json: %w", err)
	}
	body, err := plankaAPICall(jsonPayload, "/api/cards/"+cardId+"/task-lists", "POST")
	if err != nil {
		return "", fmt.Errorf("error sending request to create tasklist: %w", err)
	}
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return "", fmt.Errorf("failed to parse json: %w", err)
	}
	return jsonResponse["item"].(map[string]interface{})["id"].(string), nil
}

func createPlankaTaskInTasklist(listId string, item KaitenChecklistItem) (string, error) {
	var task PlankaTask
	task.Name = item.Text
	task.Position = 0
	task.IsCompleted = item.Checked
	jsonPayload, err := json.Marshal(task)
	if err != nil {
		return "", fmt.Errorf("error marshalling task list to json: %w", err)
	}
	body, err := plankaAPICall(jsonPayload, "/api/task-lists/"+listId+"/tasks", "POST")
	if err != nil {
		return "", fmt.Errorf("error sending request to create task: %w", err)
	}
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}
	return jsonResponse["item"].(map[string]interface{})["id"].(string), nil
}

func createPlankaLabelForBoard(boardId string, tag KaitenTag) (PlankaLabel, error) {
	var labelToCreate PlankaLabel
	labelToCreate.Name = tag.Name
	labelToCreate.Color = PlankaColors[int(tag.Color)]
	labelToCreate.Position = 0
	jsonPayload, err := json.Marshal(labelToCreate)
	if err != nil {
		return PlankaLabel{}, fmt.Errorf("error marshalling task list to json: %w", err)

	}
	body, err := plankaAPICall(jsonPayload, "/api/boards/"+boardId+"/labels", "POST")
	if err != nil {
		return PlankaLabel{}, fmt.Errorf("error sending request to create task: %w", err)
	}
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return PlankaLabel{}, fmt.Errorf("failed to parse JSON: %w", err)
	}
	labelToCreate.Id = jsonResponse["item"].(map[string]interface{})["id"].(string)
	return labelToCreate, nil
}

func createPlankaLabelForCard(cardId string, labelId string) error {
	var lbl PlankaLabelForCard
	lbl.ID = labelId
	jsonPayload, err := json.Marshal(lbl)
	if err != nil {
		return fmt.Errorf("error marshalling task list to json: %w", err)
	}
	_, err = plankaAPICall(jsonPayload, "/api/cards/"+cardId+"/card-labels", "POST")
	if err != nil {
		return fmt.Errorf("error sending request to create task: %w", err)
	}
	return nil
}
