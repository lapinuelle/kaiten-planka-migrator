package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type KaitenSpace struct {
	ID       float64 `json:"id"`
	Name     string  `json:"name"`
	ParentID string  `json:"parent_id"`
	UID      string  `json:"uid"`
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

func get_kaiten_spaces() ([]KaitenSpace, error) {
	kaitenUrl, exists := os.LookupEnv("KAITEN_URL")
	if !exists {
		return nil, fmt.Errorf("KAITEN_URL environment variable is not set")
	}
	kaitenToken, exists := os.LookupEnv("KAITEN_TOKEN")
	if !exists {
		return nil, fmt.Errorf("KAITEN_TOKEN environment variable is not set")
	}
	req, err := http.NewRequest("GET", kaitenUrl+"/api/latest/spaces", nil)
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

	var json_spaces []interface{}
	var spaces []KaitenSpace

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
			spaces = append(spaces, KaitenSpace{
				ID:       spaceMap["id"].(float64),
				Name:     spaceMap["title"].(string),
				ParentID: parent_uid,
				UID:      spaceMap["uid"].(string),
			})
		} else {
			fmt.Println("Error converting space to KaitenSpace struct")
		}

	}

	return spaces, nil
}
