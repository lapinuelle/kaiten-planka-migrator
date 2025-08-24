package main

import (
	"encoding/json"
	"fmt"
	"log"
	"slices"

	"github.com/joho/godotenv"
)

type PlankaUser struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	err := planka_delete_users()
	if err != nil {
		log.Fatalf("Error deleting Planka users: %v", err)
	}
	errr := delete_planka_projects()
	if errr != nil {
		log.Fatalf("Error deleting Planka projects: %v", errr)
	}
	// os.Exit(0)
	kaiten_checklists, err := read_checklists_from_file()
	if err != nil {
		log.Fatalf("Error reading checklists from file: %v", err)
	}
	fmt.Printf("%s\n", kaiten_checklists)
	raw_users, err := get_kaiten_users()

	{
		if err != nil {
			log.Fatalf("Error fetching Kaiten users: %v", err)
		}
		var users interface{}
		if err := json.Unmarshal(raw_users.([]byte), &users); err != nil {
			log.Fatalf("failed to parse JSON: %s", err)
		}
		emails, err := get_planka_users_emails()
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
				err := create_planka_user(userData)
				if err != nil {
					log.Printf("Error creating Planka user %s: %v", userData.Username, err)
					continue
				}
				fmt.Printf("Created Planka user: %s\n", userData.Username)
			}
		}
	}
	spaces, err := get_kaiten_spaces()
	if err != nil {
		log.Fatalf("Error fetching Kaiten spaces: %v", err)
	}
	plankaProjects := make(map[string]CreatedProject)
	for _, space := range spaces {
		if err != nil {
			log.Fatalf("Error getting boards for space")
		}

		if space.ParentID == "" {
			// Creating projects for top-level spaces
			plankaProject, err := create_planka_project(space)
			plankaProjects[plankaProject.KaitenSpaceUID] = plankaProject
			if err != nil {
				log.Printf("Error creating Planka project for space %s: %v", space.Name, err)
				continue
			}
			fmt.Printf("Planka project: %s with ID: %s\n", plankaProject.Name, plankaProject.ID)

		}
	}
	// Creating boards for each project
	// If there are multiple boards in a space, they will be prefixed with the space name
	// If there is only one board, it will be named after the space
	for _, space := range spaces {
		boardTitlePrefix := ""
		boards, err := get_kaiten_boards_for_space(space)
		if err != nil {
			log.Fatalf("Error getting boards for space")
		}
		if len(boards) > 1 {
			boardTitlePrefix = space.Name + ": "
			fmt.Printf("%s\n", boardTitlePrefix)
		}

		spaceIdforBoard := space.UID
		if space.ParentID != "" {
			spaceIdforBoard = space.ParentID
		}
		spaceUIDforBoardCreation := spaces[spaceIdforBoard].UID
		for _, kaiten_board := range boards {
			if len(boards) < 2 {
				kaiten_board.Title = space.Name
			}
			fmt.Printf("Board named %s created in project %s\n", boardTitlePrefix+kaiten_board.Title, plankaProjects[spaceUIDforBoardCreation].Name)
			board, err := create_planka_board(plankaProjects[spaceUIDforBoardCreation].ID, kaiten_board, boardTitlePrefix)
			if err != nil {
				log.Printf("Error creating Planka board for project %s: %v", plankaProjects[spaceUIDforBoardCreation].ID, err)
				continue
			}
			columns, err := get_kaiten_columns_for_board(kaiten_board.ID)
			if err != nil {
				log.Printf("Error getting columns for board %s: %v", board.ID, err)
				continue
			}
			emails, err := get_planka_users_emails()
			if err != nil {
				log.Fatalf("Error fetching Planka user emails: %v", err)
			}
			for _, email := range emails {
				userId, err := get_planka_userId_by_email(email)
				if err != nil {
					log.Printf("Error getting Planka user ID for email %s: %v", email, err)
					continue
				}
				err = set_planka_board_member(board.ID, userId)
				if err != nil {
					log.Printf("Error setting Planka board member for board %s and user %s: %v", board.ID, userId, err)
					continue
				}
			}
			for _, column := range columns {
				column.Type = "active"
				if board.Name == "Графика" {
					fmt.Printf("GOTCHA!\n")
				}
				plankaColumn, err := create_planka_list(board.ID, column)
				if err != nil {
					log.Printf("Error creating Planka column for board %s: %v", board.ID, err)
					continue
				}
				fmt.Printf("Created Planka column: %s in board: %s\n", plankaColumn.Name, board.Name)
				cards, err := get_kaiten_cards_for_column(column.Id)
				if err != nil {
					log.Printf("Error getting cards for column %s: %v", column.Id, err)
					continue
				}
				for _, card := range cards {

					cardId, err := create_planka_card(plankaColumn.ID, card)
					if card.Memders != nil {

						for _, member := range card.Memders {
							userId, err := get_planka_userId_by_email(member)
							if err != nil {
								log.Printf("Error getting Planka user ID for email %s: %v", member, err)
								continue
							}

							err = set_planka_card_member(cardId, userId)
							if err != nil {
								log.Printf("Error setting Planka card member for card %s and user %s: %v", cardId, userId, err)
								continue
							}

						}

					}
					if _, ok := kaiten_checklists[card.ID]; ok {
						for _, chechlistId := range kaiten_checklists[card.ID] {
							kaiten_list, err := get_kaiten_checklist_for_card(card.ID, chechlistId)
							if err != nil {
								log.Printf("Error getting checklist for card %s: %v", cardId, err)
								continue
							}
							list_id, err := create_planka_card_tasklist(cardId, kaiten_list)
							if err != nil {
								fmt.Printf("Error creating tasklist")
							}
							for _, item := range kaiten_list.Items {
								task_id, err := create_planka_task_in_tasklist(list_id, item)
								if err != nil {
									fmt.Printf("Error creating task: %w\n", err)
									continue
								}
								fmt.Printf("Created task %s\n", task_id)
							}

						}
					}
					comments, err := get_kaiten_comments_for_card(card.ID)
					if err == nil && comments != nil {
						for _, comment := range comments {
							err := create_planka_card_comment(cardId, comment)
							if err != nil {
								log.Printf("Error creating Planka comment for card %s: %v", cardId, err)
								continue
							}
						}
					}
					attachments, err := get_kaiten_attachments_for_card(card.ID)
					if err != nil {
						log.Printf("Error getting attachments for card %s: %v", card.ID, err)
					}
					if attachments != nil {
						fmt.Printf("Got attachments for card %s: %v\n", cardId, attachments)
						for _, attachment := range attachments {
							_, err := create_planka_card_attachment(cardId, attachment)
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
					fmt.Printf("Created Planka card: %s in list: %s\n", cardId, plankaColumn.Name)
				}

			}

		}

	}

}
