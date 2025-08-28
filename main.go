package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/joho/godotenv"
)

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func getEnv(name string) (string, error) {
	val, exists := os.LookupEnv(name)
	if !exists {
		return "", fmt.Errorf("%s environment variable is not set", name)
	}
	return val, nil
}

func main() {

	err := plankaDeleteUser()
	if err != nil {
		log.Fatalf("Error deleting Planka users: %v", err)
	}
	err = deletePlankaProjects()
	if err != nil {
		log.Fatalf("Error deleting Planka projects: %v", err)
	}

	rawUsers, err := getKaitenUsers()
	if err != nil {
		log.Fatalf("Error getting users from Kaiten: %v", err)
	}

	tags, err := getKaitenTags()
	if err != nil {
		log.Fatalf("Error getting tags from Kaiten: %v", err)
	}

	var users interface{}
	if err := json.Unmarshal(rawUsers.([]byte), &users); err != nil {
		log.Fatalf("failed to parse JSON: %s", err)
	}
	emails, err := getPlankaUsersMails()
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
			err := createPlankaUser(userData)
			if err != nil {
				log.Printf("Error creating Planka user %s: %v", userData.Username, err)
				continue
			}
			log.Printf("Created Planka user: %s\n", userData.Username)
		}
	}

	spaces, err := getKaitenSpaces()
	if err != nil {
		log.Fatalf("Error fetching Kaiten spaces: %v", err)
	}
	plankaProjects := make(map[string]PlankaProject)
	for _, space := range spaces {
		if err != nil {
			log.Fatalf("Error getting boards for space")
		}

		if space.ParentID == "" {

			plankaProject, err := createPlankaProject(space)
			plankaProjects[plankaProject.KaitenSpaceUID] = plankaProject
			if err != nil {
				log.Printf("Error creating Planka project for space %s: %v", space.Name, err)
				continue
			}
			log.Printf("Planka project: %s with ID: %s\n", plankaProject.Name, plankaProject.ID)

		}
	}

	for _, space := range spaces {
		boardTitlePrefix := ""
		boards, err := getKaitenBoardsForSpace(space)
		if err != nil {
			log.Fatalf("Error getting boards for space")
		}
		if len(boards) > 1 {
			boardTitlePrefix = space.Name + ": "
			log.Printf("%s\n", boardTitlePrefix)
		}

		spaceIdforBoard := space.UID
		if space.ParentID != "" {
			spaceIdforBoard = space.ParentID
		}
		spaceUIDforBoardCreation := spaces[spaceIdforBoard].UID

		for _, kaitenBoard := range boards {
			if len(boards) < 2 {
				kaitenBoard.Title = space.Name
			}
			log.Printf("Board named %s created in project %s\n", boardTitlePrefix+kaitenBoard.Title, plankaProjects[spaceUIDforBoardCreation].Name)
			board, err := createPlankaBoard(plankaProjects[spaceUIDforBoardCreation].ID, kaitenBoard, boardTitlePrefix)
			boardLabels := make(map[float64]PlankaLabel)
			if err != nil {
				log.Printf("Error creating Planka board for project %s: %v", plankaProjects[spaceUIDforBoardCreation].ID, err)
				continue
			}
			columns, err := getKaitenColumnsForBoard(kaitenBoard.ID)
			if err != nil {
				log.Printf("Error getting columns for board %s: %v", board.ID, err)
				continue
			}
			emails, err := getPlankaUsersMails()
			if err != nil {
				log.Fatalf("Error fetching Planka user emails: %v", err)
			}
			for _, email := range emails {
				userId, err := getPlankaUserIDByEmail(email)
				if err != nil {
					log.Printf("Error getting Planka user ID for email %s: %v", email, err)
					continue
				}
				err = setPlankaBoardMember(board.ID, userId)
				if err != nil {
					log.Printf("Error setting Planka board member for board %s and user %s: %v", board.ID, userId, err)
					continue
				}
			}
			for _, column := range columns {
				column.Type = "active"
				plankaColumn, err := createPlankaList(board.ID, column)
				if err != nil {
					log.Printf("Error creating Planka column for board %s: %v", board.ID, err)
					continue
				}
				log.Printf("Created Planka column: %s in board: %s\n", plankaColumn.Name, board.Name)
				cards, err := getKaitenCardsForColumn(column.Id)
				if err != nil {
					log.Printf("Error getting cards for column %s: %v", column.Id, err)
					continue
				}

				for _, card := range cards {
					if card.Archived {
						continue
					}
					cardId, err := createPlankaCard(plankaColumn.ID, card)
					if err != nil {
						log.Fatalf("Can't create card")
					}
					if card.Memders != nil {

						for _, member := range card.Memders {
							userId, err := getPlankaUserIDByEmail(member)
							if err != nil {
								log.Printf("Error getting Planka user ID for email %s: %v", member, err)
								continue
							}

							err = setPlankaCardNumber(cardId, userId)
							if err != nil {
								log.Printf("Error setting Planka card member for card %s and user %s: %v", cardId, userId, err)
								continue
							}

						}

					}
					if card.TagIds != nil {
						for _, tagID := range card.TagIds {
							if _, ok := boardLabels[tagID]; !ok {
								label, err := createPlankaLabelForBoard(board.ID, tags[tagID])
								if err == nil {
									boardLabels[tagID] = label
								}
							}
							err = createPlankaLabelForCard(cardId, boardLabels[tagID].Id)
							if err != nil {
								log.Printf("Error setting Planka label for card %s: %v", cardId, err)
								continue
							}

						}
					}

					for _, chechlistId := range card.Checklists {
						kaiten_list, err := getKaitenChecklistsForCard(card.ID, chechlistId)
						if err != nil {
							log.Printf("Error getting checklist for card %s: %v", cardId, err)
							continue
						}
						list_id, err := createPlankaTasklistForCard(cardId, kaiten_list)
						if err != nil {
							log.Printf("Error creating tasklist")
						}
						for _, item := range kaiten_list.Items {
							task_id, err := createPlankaTaskInTasklist(list_id, item)
							if err != nil {
								log.Printf("Error creating task: %w\n", err)
								continue
							}
							log.Printf("Created task %s\n", task_id)
						}

					}

					comments, err := getKaitenCommentsForCard(card.ID)
					if err == nil && comments != nil {
						for _, comment := range comments {
							err := createPlankaCommentForCard(cardId, comment)
							if err != nil {
								log.Printf("Error creating Planka comment for card %s: %v", cardId, err)
								continue
							}
						}
					}
					attachments, err := getKaitenAttachmentsForCard(card.ID)
					if err != nil {
						log.Printf("Error getting attachments for card %f: %v", card.ID, err)
					}
					if attachments != nil {
						log.Printf("Got attachments for card %s: %v\n", cardId, attachments)
						for _, attachment := range attachments {
							_, err := createPlankaAttachmentForCard(cardId, attachment)
							if err != nil {
								log.Printf("Error creating Planka attachment for card %s: %v", cardId, err)
								continue
							}
						}
					}

					if err != nil {
						log.Printf("Error creating Planka card in list %s: %v", plankaColumn.ID, err)
						continue
					}
					log.Printf("Created Planka card: %s in list: %s\n", cardId, plankaColumn.Name)
				}

			}

		}

	}

}
