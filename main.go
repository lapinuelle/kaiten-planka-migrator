package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type Space struct {
	Id              uint   `json:"id"`
	Title           string `json:"title"`
	ParentEntityUid string `json:"parent_entity_uid"`
	BoardId         uint   `json:"board_id"`
}

func readJsonFile(filePath string) (interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var data interface{}
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return data, nil
}

func getSpaces(data interface{}) map[string]Space {
	spaces := make(map[string]Space)

	for _, value := range data.([]interface{}) {
		var parent_entity_uid string = ""
		if value.(map[string]interface{})["parent_entity_uid"] != nil {
			parent_entity_uid = value.(map[string]interface{})["parent_entity_uid"].(string)
		}
		spaces[value.(map[string]interface{})["uid"].(string)] = Space{
			Id:              uint(value.(map[string]interface{})["id"].(float64)),
			Title:           value.(map[string]interface{})["title"].(string),
			ParentEntityUid: parent_entity_uid,
		}
	}
	for key, _ := range spaces {
		if spaces[key].ParentEntityUid != "" {
			tempSpace := spaces[key]
			tempSpace.Title = fmt.Sprintf("%s: %s", spaces[spaces[key].ParentEntityUid].Title, spaces[key].Title)
			spaces[key] = tempSpace
		}
	}

	return spaces
}

func main() {

	var boards_to_spaces = make(map[uint]uint)

	spacesFilePath := "/Users/xander/Downloads/export_Kaiten/exported_tables/space_part1.json"
	spacesToBoardsPath := "/Users/xander/Downloads/export_Kaiten/exported_tables/space_boards_part1.json"

	data, err := readJsonFile(spacesFilePath)
	if err != nil {
		log.Fatalf("Error reading spaces file: %v", err)
	}
	var spaces = getSpaces(data)
	data, err = readJsonFile(spacesToBoardsPath)
	if err != nil {
		log.Fatalf("Error reading spaces to boards file: %v", err)
	}

	for _, value := range data.([]interface{}) {
		boards_to_spaces[uint(value.(map[string]interface{})["space_id"].(float64))] = uint(value.(map[string]interface{})["board_id"].(float64))
	}
	for key, _ := range spaces {
		if !strings.HasPrefix(spaces[key].Title, "Тестирование") {
			fmt.Printf("Space: %s, Board Id: %d\n", spaces[key].Title, boards_to_spaces[spaces[key].Id])
		}
	}

	//fmt.Printf("Parsed JSON: %+v\n", data)
}
