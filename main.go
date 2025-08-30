package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

var (
	clientPool = &sync.Pool{
		New: func() interface{} {
			return &http.Client{
				Timeout: 30 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 100,
					IdleConnTimeout:     90 * time.Second,
					TLSHandshakeTimeout: 10 * time.Second,
					DisableKeepAlives:   false,
					ForceAttemptHTTP2:   true,
				},
			}
		},
	}
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	if err := initPlankaEnv(); err != nil {
		log.Fatalf("Cannot get variable values for PLANKA API: %v", err)
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
	wg := &sync.WaitGroup{}
	errChan := make(chan error, 10)

	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := plankaDeleteUser(); err != nil {
			errChan <- fmt.Errorf("error deleting Planka users: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := deletePlankaProjects(); err != nil {
			errChan <- fmt.Errorf("error deleting Planka projects: %v", err)
		}
	}()
	wg.Wait()

	select {
	case err := <-errChan:
		log.Fatal(err)
	default:

	}

	var rawUsers interface{}
	var emails []string
	var tags map[float64]KaitenTag

	wg.Add(3)
	go func() {
		defer wg.Done()
		var err error
		rawUsers, err = getKaitenUsers()
		if err != nil {
			errChan <- fmt.Errorf("error getting users from Kaiten: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		tags, err = getKaitenTags()
		if err != nil {
			errChan <- fmt.Errorf("error getting tags from Kaiten: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		emails, err = getPlankaUsersMails()
		if err != nil {
			errChan <- fmt.Errorf("error fetching Planka user emails: %v", err)
		}
	}()
	wg.Wait()

	select {
	case err := <-errChan:
		log.Fatal(err)
	default:
	}

	var users interface{}
	if err := json.Unmarshal(rawUsers.([]byte), &users); err != nil {
		log.Fatalf("failed to parse JSON: %s", err)
	}

	emailSet := make(map[string]struct{}, len(emails))
	for _, email := range emails {
		emailSet[email] = struct{}{}
	}

	var kaitenUsers []KaitenUser
	for _, user := range users.([]interface{}) {
		userMap, ok := user.(map[string]interface{})
		if !ok {
			log.Println("Skipping invalid user data")
			continue
		}

		kaitenUsers = append(kaitenUsers, KaitenUser{
			Email:    userMap["email"].(string),
			FullName: userMap["full_name"].(string),
			Username: userMap["username"].(string),
		})
	}

	wg.Add(len(kaitenUsers))

	for _, user := range kaitenUsers {
		go func(user KaitenUser) {
			defer wg.Done()

			if _, exists := emailSet[user.Email]; !exists {
				name := user.FullName
				if name == "" {
					name = user.Username
				}
				userData := PlankaUser{
					Username: user.Username,
					Name:     name,
					Email:    user.Email,
					Password: "1234tempPass",
					Role:     "projectOwner",
				}
				if err := createPlankaUser(userData); err != nil {
					log.Printf("Error creating Planka user %s: %v", userData.Username, err)
					return
				}
				log.Printf("Created Planka user: %s\n", userData.Username)
			}
		}(user)
	}
	wg.Wait()

	spaces, err := getKaitenSpaces()
	if err != nil {
		log.Fatalf("Error fetching Kaiten spaces: %v", err)
	}

	plankaProjects := make(map[string]PlankaProject)
	var projectsMutex sync.Mutex

	wg.Add(len(spaces))
	for _, space := range spaces {
		go func(space KaitenSpace) {
			defer wg.Done()
			if space.ParentID == "" {
				plankaProject, err := createPlankaProject(space)
				if err != nil {
					log.Printf("Error creating Planka project for space %s: %v", space.Name, err)
					return
				}

				projectsMutex.Lock()
				plankaProjects[plankaProject.KaitenSpaceUID] = plankaProject
				projectsMutex.Unlock()
				log.Printf("Planka project: %s with ID: %s\n", plankaProject.Name, plankaProject.ID)
			}
		}(space)
	}
	wg.Wait()

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
						log.Printf("Error creating Planka card in column %s: %v", plankaColumn.ID, err)
						continue
					}
					if card.Members != nil {

						for _, member := range card.Members {
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

					processCardTags(card, cardId, board.ID, tags)

					processCardChecklists(card, cardId)

					comments, err := getKaitenCommentsForCard(card.ID)
					if err == nil && comments != nil {
						wg.Add(len(comments))
						for _, comment := range comments {
							go func(comment KaitenComment) {
								defer wg.Done()
								err := createPlankaCommentForCard(cardId, comment)
								if err != nil {
									log.Printf("Error creating Planka comment for card %s: %v", cardId, err)
									return
								}
							}(comment)
						}
						wg.Wait()
					}
					attachments, err := getKaitenAttachmentsForCard(card.ID)
					if err != nil {
						log.Printf("Error getting attachments for card %f: %v", card.ID, err)
					}
					if attachments != nil {
						log.Printf("Got attachments for card %s: %v\n", cardId, attachments)
						wg.Add(len(attachments))
						for _, attachment := range attachments {
							go func(attachment KaitenAttachment) {
								defer wg.Done()
								if _, err := createPlankaAttachmentForCard(cardId, attachment); err != nil {
									log.Printf("Error creating Planka attachment for card %s: %v", cardId, err)
									return
								}
							}(attachment)
						}
						wg.Wait()

					}

					log.Printf("Created Planka card: %s in list: %s\n", cardId, plankaColumn.Name)
				}

			}

		}

	}

}

func processCardChecklists(card KaitenCard, cardId string) {
	if len(card.Checklists) > 0 {
		checkListsWG := &sync.WaitGroup{}
		checkListsWG.Add(len(card.Checklists))

		checklistSemaphore := make(chan struct{}, 3)

		for _, checklistId := range card.Checklists {
			go func(checklistId float64) {
				defer checkListsWG.Done()

				checklistSemaphore <- struct{}{}
				defer func() { <-checklistSemaphore }()

				kaitenList, err := getKaitenChecklistsForCard(card.ID, checklistId)
				if err != nil {
					log.Printf("Error getting checklist for card %s: %v", cardId, err)
					return
				}

				listId, err := createPlankaTasklistForCard(cardId, kaitenList)
				if err != nil {
					log.Printf("Error creating tasklist for card %s: %v", cardId, err)
					return
				}

				if len(kaitenList.Items) > 0 {
					itemWg := &sync.WaitGroup{}
					itemWg.Add(len(kaitenList.Items))

					itemSemaphore := make(chan struct{}, 5)

					for _, item := range kaitenList.Items {
						go func(item KaitenChecklistItem) {
							defer itemWg.Done()

							itemSemaphore <- struct{}{}
							defer func() { <-itemSemaphore }()

							taskId, err := createPlankaTaskInTasklist(listId, item)
							if err != nil {
								log.Printf("Error creating task in checklist for card %s: %v", cardId, err)
								return
							}
							log.Printf("Created task %s in checklist for card %s", taskId, cardId)
						}(item)
					}

					itemWg.Wait()
				}

				log.Printf("Completed processing checklist %f for card %s", checklistId, cardId)
			}(checklistId)
		}

		checkListsWG.Wait()
	}

}

func processCardTags(card KaitenCard, cardId string, boardId string, tags map[float64]KaitenTag) {
	if card.TagIds != nil {
		tagsWG := &sync.WaitGroup{}
		tagsWG.Add(len(card.TagIds))
		boardLabels := make(map[float64]PlankaLabel)
		boardLabelsMutex := &sync.Mutex{}

		semaphore := make(chan struct{}, 5)

		for _, tagID := range card.TagIds {
			go func(tagID float64) {
				defer tagsWG.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				boardLabelsMutex.Lock()
				label, exists := boardLabels[tagID]
				boardLabelsMutex.Unlock()

				if !exists {
					newLabel, err := createPlankaLabelForBoard(boardId, tags[tagID])
					if err != nil {
						log.Printf("Error creating label for tag %f: %v", tagID, err)
						return
					}

					boardLabelsMutex.Lock()
					boardLabels[tagID] = newLabel
					label = newLabel
					boardLabelsMutex.Unlock()
				}

				if err := createPlankaLabelForCard(cardId, label.Id); err != nil {
					log.Printf("Error setting Planka label for card %s: %v", cardId, err)
					return
				}
			}(tagID)
		}

		tagsWG.Wait()
	}
}
