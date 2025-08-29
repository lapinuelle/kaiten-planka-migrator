package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type KaitenSpace struct {
	ID        float64  `json:"id"`
	Name      string   `json:"name"`
	ParentID  string   `json:"parent_id"`
	UID       string   `json:"uid"`
	ChildIdDs []string `json:"child_ids,omitempty"` // Optional field for child IDs
}

type KaitenBoard struct {
	ID    float64 `json:"id"`
	Title string  `json:"title"`
}

type KaitenColumn struct {
	Position float64 `json:"position"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Id       float64 `json:"id"`
	BoardID  float64 `json:"board_id"`
}

type KaitenCard struct {
	ID          float64   `json:"id"`
	BoardID     float64   `json:"board_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	SortOrder   float64   `json:"sort_order"`
	Members     []string  `json:"members"`
	DueDate     string    `json:"due_date,omitempty"`
	StartDate   string    `json:"start_date,omitempty"`
	EndDate     string    `json:"end_date,omitempty"`
	TagIds      []float64 `json:"tag_ids,omitempty"`
	Archived    bool      `json:"archived"`
	Checklists  []float64 `json:"checklists,omitempty"`
}

type KaitenComment struct {
	ID          float64 `json:"id"`
	AuthorEmail string  `json:"author_email"`
	CreatedAt   string  `json:"created_at"`
	Text        string  `json:"text"`
}

type KaitenAttachment struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type KaitenChecklist struct {
	Name  string                `json:"name"`
	Items []KaitenChecklistItem `json:"items"`
}

type KaitenChecklistItem struct {
	Text    string `json:"name"`
	Checked bool   `json:"checked"`
}

type KaitenTag struct {
	Id    float64 `json:"id"`
	Name  string  `json:"name"`
	Color float64 `json:"color"`
}

type KaitenUser struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Username string `json:"username"`
}

var (
	kaitenLimiter = rate.NewLimiter(rate.Every(time.Second/4), 2)

	kaitenURL   string
	kaitenToken string
	envInitOnce sync.Once
	envInitErr  error
)

func initKaitenEnv() error {
	envInitOnce.Do(func() {
		kaitenURL, envInitErr = getEnv("KAITEN_URL")
		if envInitErr != nil {
			envInitErr = fmt.Errorf("failed to get KAITEN_URL: %w", envInitErr)
			return
		}

		kaitenToken, envInitErr = getEnv("KAITEN_TOKEN")
		if envInitErr != nil {
			envInitErr = fmt.Errorf("failed to get KAITEN_TOKEN: %w", envInitErr)
			return
		}

		kaitenURL = strings.TrimRight(kaitenURL, "/")
	})
	return envInitErr
}

func kaitenAPICall(url string, method string) ([]byte, error) {
	return kaitenAPICallWithContext(context.Background(), url, method)
}

func kaitenAPICallWithContext(ctx context.Context, url string, method string) ([]byte, error) {
	if err := initKaitenEnv(); err != nil {
		return nil, err
	}

	fullURL := kaitenURL + "/" + strings.TrimPrefix(url, "/")

	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+kaitenToken)
	req.Header.Set("Content-Type", "application/json")

	client := clientPool.Get().(*http.Client)
	defer clientPool.Put(client)

	if err := kaitenLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("failed to wait for rate limiter: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func getKaitenUsers() (interface{}, error) {
	return kaitenAPICall("/api/latest/users", "GET")
}

func getKaitenTags() (map[float64]KaitenTag, error) {
	body, err := kaitenAPICall("/api/latest/tags", "GET")
	if err != nil {
		return nil, fmt.Errorf("failed to get tags from Kaiten: %w", err)
	}

	type Tag struct {
		ID    float64 `json:"id"`
		Name  string  `json:"name"`
		Color float64 `json:"color"`
	}

	var tags []Tag
	if err := json.Unmarshal(body, &tags); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	result := make(map[float64]KaitenTag)
	for _, tag := range tags {
		result[tag.ID] = KaitenTag{
			Name:  tag.Name,
			Id:    tag.ID,
			Color: tag.Color,
		}
	}

	return result, nil
}

func getKaitenSpaces() (map[string]KaitenSpace, error) {
	body, err := kaitenAPICall("/api/latest/spaces", "GET")

	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var json_spaces []interface{}
	spaces := make(map[string]KaitenSpace)

	if err := json.Unmarshal(body, &json_spaces); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}
	for _, space := range json_spaces {
		if spaceMap, ok := space.(map[string]interface{}); ok {
			var parent_uid string
			if spaceMap["parent_entity_uid"] != nil {
				parent_uid = spaceMap["parent_entity_uid"].(string)
			} else {
				parent_uid = ""
			}
			spaces[spaceMap["uid"].(string)] = KaitenSpace{
				ID:       spaceMap["id"].(float64),
				Name:     spaceMap["title"].(string),
				ParentID: parent_uid,
				UID:      spaceMap["uid"].(string),
			}
			log.Printf("%s\n", spaces[spaceMap["uid"].(string)].ChildIdDs)
		} else {
			return nil, fmt.Errorf("error converting space to KaitenSpace struct: %w", err)
		}
	}
	for _, space := range json_spaces {
		if spaceMap, ok := space.(map[string]interface{}); ok {
			if spaceMap["parent_entity_uid"] != nil {
				parentSpace := spaces[spaceMap["parent_entity_uid"].(string)]
				log.Printf("Parent Space: %s, Space: %s\n", parentSpace.Name, spaceMap["title"].(string))
				parentSpace.ChildIdDs = append(spaces[spaceMap["parent_entity_uid"].(string)].ChildIdDs, spaceMap["uid"].(string))
				spaces[spaceMap["parent_entity_uid"].(string)] = parentSpace
			}
		}
	}

	return spaces, nil
}

func getKaitenBoardsForSpace(space KaitenSpace) ([]KaitenBoard, error) {
	body, err := kaitenAPICall("/api/latest/spaces/"+strconv.FormatFloat(space.ID, 'f', -1, 64)+"/boards", "GET")
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var json_boards []interface{}
	var boards []KaitenBoard
	if err := json.Unmarshal(body, &json_boards); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}
	for _, brd := range json_boards {
		boards = append(boards, KaitenBoard{
			ID:    brd.(map[string]interface{})["id"].(float64),
			Title: brd.(map[string]interface{})["title"].(string),
		})
	}
	return boards, nil

}

func getKaitenColumnsForBoard(boardId float64) ([]KaitenColumn, error) {
	body, err := kaitenAPICall("/api/latest/boards/"+strconv.FormatFloat(boardId, 'f', -1, 64)+"/columns", "GET")
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var jsonColumns []interface{}

	if err := json.Unmarshal(body, &jsonColumns); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}
	var columns []KaitenColumn
	for _, col := range jsonColumns {
		columns = append(columns, KaitenColumn{
			Position: col.(map[string]interface{})["sort_order"].(float64),
			Name:     col.(map[string]interface{})["title"].(string),
			Id:       col.(map[string]interface{})["id"].(float64),
			BoardID:  boardId,
		})
	}
	return columns, nil
}

func getKaitenCardsForColumn(columnId float64) ([]KaitenCard, error) {
	body, err := kaitenAPICall("/api/latest/cards?column_ids="+strconv.FormatFloat(columnId, 'f', -1, 64), "GET")
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var jsonCards []interface{}

	if err := json.Unmarshal(body, &jsonCards); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}
	var cards []KaitenCard
	for _, jsonCard := range jsonCards {
		card, err := getKaitenCardById(jsonCard.(map[string]interface{})["id"].(float64))
		if err != nil {
			return nil, fmt.Errorf("error getting card by ID: %w", err)
		}
		cards = append(cards, card)
	}
	return cards, nil
}

func getKaitenCardById(cardId float64) (KaitenCard, error) {
	body, err := kaitenAPICall("/api/latest/cards/"+strconv.FormatFloat(cardId, 'f', -1, 64), "GET")
	if err != nil {
		return KaitenCard{}, fmt.Errorf("error reading response body: %w", err)
	}

	var jsonCard map[string]any

	if err := json.Unmarshal(body, &jsonCard); err != nil {
		return KaitenCard{}, fmt.Errorf("error parsing JSON: %w", err)
	}
	var card KaitenCard
	if jsonCard["checklists"] != nil {
		for _, checklist := range jsonCard["checklists"].([]any) {
			if checklist.(map[string]any)["id"] != nil {
				card.Checklists = append(card.Checklists, checklist.(map[string]any)["id"].(float64))
			}
		}
	}
	card.ID = jsonCard["id"].(float64)
	card.Title = jsonCard["title"].(string)
	if jsonCard["description"] == nil || jsonCard["description"] == "" {
		card.Description = " "

	} else {
		card.Description = jsonCard["description"].(string)
	}
	if jsonCard["archived"] != nil {
		card.Archived = jsonCard["archived"].(bool)
	} else {
		card.Archived = false
	}
	if jsonCard["due_date"] != nil {
		card.DueDate = jsonCard["due_date"].(string)
		card.StartDate = ""
		card.EndDate = ""
	} else {
		card.DueDate = ""
		if jsonCard["planned_start"] != nil {
			card.StartDate = jsonCard["planned_start"].(string)
		} else {
			card.StartDate = ""
		}
		if jsonCard["planned_end"] != nil {
			card.EndDate = jsonCard["planned_end"].(string)
		} else {
			card.EndDate = ""
		}
	}
	var tagIds []float64
	if jsonCard["tag_ids"] != nil {
		for _, id := range jsonCard["tag_ids"].([]interface{}) {
			tagIds = append(tagIds, id.(float64))
		}
		card.TagIds = tagIds
	}
	if jsonCard["sort_order"].(float64) < 1 {
		card.SortOrder = 1
	} else {
		card.SortOrder = jsonCard["sort_order"].(float64)
	}

	if jsonCard["members"] != nil {
		for _, member := range jsonCard["members"].([]interface{}) {
			card.Members = append(card.Members, member.(map[string]interface{})["email"].(string))
		}
	}
	if jsonCard["properties"] != nil {
		if jsonCard["properties"].(map[string]interface{}) != nil {
			for _, value := range jsonCard["properties"].(map[string]interface{}) {
				card.Description = "## Детализация\n\n" + value.(string) + "\n\n## Описание" + card.Description
			}
		}
	}
	return card, nil
}

func getKaitenCommentsForCard(cardId float64) ([]KaitenComment, error) {
	body, err := kaitenAPICall("/api/latest/cards/"+strconv.FormatFloat(cardId, 'f', -1, 64)+"/comments", "GET")
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if string(body) == "[]" {
		return nil, nil
	}

	var jsonComments []interface{}
	if err := json.Unmarshal(body, &jsonComments); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}
	var comments []KaitenComment
	for _, cmt := range jsonComments {
		comments = append(comments, KaitenComment{
			ID:          cmt.(map[string]interface{})["id"].(float64),
			AuthorEmail: cmt.(map[string]interface{})["author"].(map[string]interface{})["email"].(string),
			CreatedAt:   cmt.(map[string]interface{})["created"].(string),
			Text:        cmt.(map[string]interface{})["text"].(string),
		})
	}
	return comments, nil
}

func getKaitenAttachmentsForCard(cardId float64) ([]KaitenAttachment, error) {
	attachments, err := kaitenAPICall("/api/latest/cards/"+strconv.FormatFloat(cardId, 'f', -1, 64)+"/files", "GET")
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	if string(attachments) == "[]" {
		return nil, nil
	}
	var attachmentsInterface []interface{}
	if err := json.Unmarshal(attachments, &attachmentsInterface); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}
	var kaitenAttachments []KaitenAttachment
	for _, att := range attachmentsInterface {
		kaitenAttachments = append(kaitenAttachments, KaitenAttachment{
			Name: att.(map[string]interface{})["name"].(string),
			URL:  att.(map[string]interface{})["url"].(string),
		})

	}
	return kaitenAttachments, nil
}

func getKaitenChecklistsForCard(cardId float64, checklistId float64) (KaitenChecklist, error) {
	data, err := kaitenAPICall("/api/latest/cards/"+strconv.FormatFloat(cardId, 'f', -1, 64)+"/checklists/"+strconv.FormatFloat(checklistId, 'f', -1, 64), "GET")
	if err != nil {
		return KaitenChecklist{}, fmt.Errorf("error reading response body: %w", err)
	}
	var checklistJson interface{}
	if err := json.Unmarshal(data, &checklistJson); err != nil {
		return KaitenChecklist{}, fmt.Errorf("error parsing JSON: %w", err)
	}
	var checklistItemsJson []interface{}
	checklistItemsJson = checklistJson.(map[string]interface{})["items"].([]interface{})
	var checklistItems []KaitenChecklistItem
	var checklist KaitenChecklist
	checklist.Name = checklistJson.(map[string]interface{})["name"].(string)
	for _, item := range checklistItemsJson {
		checklistItems = append(checklistItems, KaitenChecklistItem{
			Text:    item.(map[string]interface{})["text"].(string),
			Checked: item.(map[string]interface{})["checked"].(bool),
		})
		checklist.Items = checklistItems
	}
	return checklist, nil
}
