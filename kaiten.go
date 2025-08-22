package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

func kaiten_api_call(url string, method string) ([]byte, error) {
	kaitenUrl, exists := os.LookupEnv("KAITEN_URL")
	if !exists {
		return nil, fmt.Errorf("KAITEN_URL environment variable is not set")
	}
	kaitenToken, exists := os.LookupEnv("KAITEN_TOKEN")
	if !exists {
		return nil, fmt.Errorf("KAITEN_TOKEN environment variable is not set")
	}
	req, err := http.NewRequest(method, kaitenUrl+url, nil)
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

func get_kaiten_users() (interface{}, error) {
	return kaiten_api_call("/api/latest/users", "GET")
}

func get_kaiten_spaces() (map[string]KaitenSpace, error) {
	body, err := kaiten_api_call("/api/latest/spaces", "GET")

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
	body, err := kaiten_api_call("/api/latest/spaces/"+strconv.FormatFloat(Space.ID, 'f', -1, 64)+"/boards", "GET")
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
	time.Sleep(time.Millisecond * 300)
	return boards, nil

}
