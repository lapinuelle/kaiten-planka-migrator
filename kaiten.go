package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
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
}

type KaitenCard struct {
	ID          float64   `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	SortOrder   float64   `json:"sort_order"`
	Memders     []string  `json:"members"`
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

func kaitenAPICall(url string, method string) ([]byte, error) {
	time.Sleep(time.Millisecond * 300)

	kaitenUrl, err := getEnv("KAITEN_URL")
	if err != nil {
		return nil, fmt.Errorf("failed to get KAITEN_URL: %w", err)
	}

	kaitenToken, err := getEnv("KAITEN_TOKEN")
	if err != nil {
		return nil, fmt.Errorf("failed to get KAITEN_TOKEN: %w", err)
	}

	req, err := http.NewRequest(method, kaitenUrl+url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+kaitenToken)
	req.Header.Set("Content-Type", "application/json")

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

func get_kaiten_spaces() (map[string]KaitenSpace, error) {
	body, err := kaitenAPICall("/api/latest/spaces", "GET")

	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	var json_spaces []interface{}
	spaces := make(map[string]KaitenSpace)

	if err := json.Unmarshal(body, &json_spaces); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
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
			fmt.Printf("%s\n", spaces[spaceMap["uid"].(string)].ChildIdDs)
		} else {
			fmt.Println("Error converting space to KaitenSpace struct")
		}
	}
	for _, space := range json_spaces {
		if spaceMap, ok := space.(map[string]interface{}); ok {
			if spaceMap["parent_entity_uid"] != nil {
				parentSpace := spaces[spaceMap["parent_entity_uid"].(string)]
				fmt.Printf("Parent Space: %s, Space: %s\n", parentSpace.Name, spaceMap["title"].(string))
				parentSpace.ChildIdDs = append(spaces[spaceMap["parent_entity_uid"].(string)].ChildIdDs, spaceMap["uid"].(string))
				spaces[spaceMap["parent_entity_uid"].(string)] = parentSpace
			}
		}
	}

	return spaces, nil
}

func get_kaiten_boards_for_space(Space KaitenSpace) ([]KaitenBoard, error) {
	body, err := kaitenAPICall("/api/latest/spaces/"+strconv.FormatFloat(Space.ID, 'f', -1, 64)+"/boards", "GET")
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	var json_boards []interface{}
	var boards []KaitenBoard
	if err := json.Unmarshal(body, &json_boards); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}
	for _, brd := range json_boards {
		boards = append(boards, KaitenBoard{
			ID:    brd.(map[string]interface{})["id"].(float64),
			Title: brd.(map[string]interface{})["title"].(string),
		})
	}
	return boards, nil

}

func get_kaiten_columns_for_board(boardId float64) ([]KaitenColumn, error) {
	body, err := kaitenAPICall("/api/latest/boards/"+strconv.FormatFloat(boardId, 'f', -1, 64)+"/columns", "GET")
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	var json_columns []interface{}

	if err := json.Unmarshal(body, &json_columns); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}
	var columns []KaitenColumn
	for _, col := range json_columns {
		columns = append(columns, KaitenColumn{
			Position: col.(map[string]interface{})["sort_order"].(float64),
			Name:     col.(map[string]interface{})["title"].(string),
			Id:       col.(map[string]interface{})["id"].(float64),
		})
	}
	return columns, nil
}

func get_kaiten_cards_for_column(columnId float64) ([]KaitenCard, error) {
	body, err := kaitenAPICall("/api/latest/cards?column_ids="+strconv.FormatFloat(columnId, 'f', -1, 64), "GET")
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	var json_cards []interface{}

	if err := json.Unmarshal(body, &json_cards); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}
	var cards []KaitenCard
	for _, json_card := range json_cards {
		card, err := get_kaiten_card_by_id(json_card.(map[string]interface{})["id"].(float64))
		if err != nil {
			fmt.Println("Error getting card by ID:", err)
			continue
		}
		cards = append(cards, card)
	}
	return cards, nil
}

func get_kaiten_card_by_id(cardId float64) (KaitenCard, error) {
	body, err := kaitenAPICall("/api/latest/cards/"+strconv.FormatFloat(cardId, 'f', -1, 64), "GET")
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return KaitenCard{}, err
	}

	var json_card map[string]any

	if err := json.Unmarshal(body, &json_card); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return KaitenCard{}, err
	}
	var card KaitenCard
	if json_card["checklists"] != nil {
		for _, checklist := range json_card["checklists"].([]any) {
			if checklist.(map[string]any)["id"] != nil {
				card.Checklists = append(card.Checklists, checklist.(map[string]any)["id"].(float64))
			}
		}
	}
	card.ID = json_card["id"].(float64)
	card.Title = json_card["title"].(string)
	if json_card["description"] == nil || json_card["description"] == "" {
		card.Description = " "

	} else {
		card.Description = json_card["description"].(string)
	}
	if json_card["archived"] != nil {
		card.Archived = json_card["archived"].(bool)
	} else {
		card.Archived = false
	}
	if json_card["due_date"] != nil {
		card.DueDate = json_card["due_date"].(string)
		card.StartDate = ""
		card.EndDate = ""
	} else {
		card.DueDate = ""
		if json_card["planned_start"] != nil {
			card.StartDate = json_card["planned_start"].(string)
		} else {
			card.StartDate = ""
		}
		if json_card["planned_end"] != nil {
			card.EndDate = json_card["planned_end"].(string)
		} else {
			card.EndDate = ""
		}
	}
	var tagIds []float64
	if json_card["tag_ids"] != nil {
		for _, id := range json_card["tag_ids"].([]interface{}) {
			tagIds = append(tagIds, id.(float64))
		}
		card.TagIds = tagIds
	}
	if json_card["sort_order"].(float64) < 1 {
		card.SortOrder = 1
	} else {
		card.SortOrder = json_card["sort_order"].(float64)
	}

	if json_card["members"] != nil {
		for _, member := range json_card["members"].([]interface{}) {
			card.Memders = append(card.Memders, member.(map[string]interface{})["email"].(string))
		}
	}
	if json_card["properties"] != nil {
		if json_card["properties"].(map[string]interface{}) != nil {
			for _, value := range json_card["properties"].(map[string]interface{}) {
				card.Description = "## Детализация\n\n" + value.(string) + "\n\n## Описание" + card.Description
			}
		}
	}
	return card, nil
}

func get_kaiten_comments_for_card(cardId float64) ([]KaitenComment, error) {
	body, err := kaitenAPICall("/api/latest/cards/"+strconv.FormatFloat(cardId, 'f', -1, 64)+"/comments", "GET")
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}
	if string(body) == "[]" {
		return nil, nil
	}
	var json_comments []interface{}
	if err := json.Unmarshal(body, &json_comments); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}
	var comments []KaitenComment
	for _, cmt := range json_comments {
		comments = append(comments, KaitenComment{
			ID:          cmt.(map[string]interface{})["id"].(float64),
			AuthorEmail: cmt.(map[string]interface{})["author"].(map[string]interface{})["email"].(string),
			CreatedAt:   cmt.(map[string]interface{})["created"].(string),
			Text:        cmt.(map[string]interface{})["text"].(string),
		})
	}
	return comments, nil
}

func get_kaiten_attachments_for_card(cardId float64) ([]KaitenAttachment, error) {
	attachments, err := kaitenAPICall("/api/latest/cards/"+strconv.FormatFloat(cardId, 'f', -1, 64)+"/files", "GET")
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}
	if string(attachments) == "[]" {
		return nil, nil
	}
	var attachmentsInterface []interface{}
	if err := json.Unmarshal(attachments, &attachmentsInterface); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}
	var kaiten_attachments []KaitenAttachment
	for _, att := range attachmentsInterface {
		kaiten_attachments = append(kaiten_attachments, KaitenAttachment{
			Name: att.(map[string]interface{})["name"].(string),
			URL:  att.(map[string]interface{})["url"].(string),
		})

	}
	return kaiten_attachments, nil
}

func get_kaiten_checklist_for_card(card_id float64, checklist_id float64) (KaitenChecklist, error) {
	data, err := kaitenAPICall("/api/latest/cards/"+strconv.FormatFloat(card_id, 'f', -1, 64)+"/checklists/"+strconv.FormatFloat(checklist_id, 'f', -1, 64), "GET")
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return KaitenChecklist{}, err
	}
	var checklist_json interface{}
	if err := json.Unmarshal(data, &checklist_json); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return KaitenChecklist{}, err
	}
	var checklist_items_json []interface{}
	checklist_items_json = checklist_json.(map[string]interface{})["items"].([]interface{})
	var checklist_items []KaitenChecklistItem
	var checklist KaitenChecklist
	checklist.Name = checklist_json.(map[string]interface{})["name"].(string)
	for _, item := range checklist_items_json {
		checklist_items = append(checklist_items, KaitenChecklistItem{
			Text:    item.(map[string]interface{})["text"].(string),
			Checked: item.(map[string]interface{})["checked"].(bool),
		})
		checklist.Items = checklist_items
	}
	return checklist, nil
}
