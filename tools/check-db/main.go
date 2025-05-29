package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "../../data/forum.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== MESSAGES ===")
	rows, err := db.Query("SELECT id, user_id, username, content, created_at FROM messages ORDER BY created_at DESC LIMIT 5")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, userID int64
		var username, content, createdAt string
		err := rows.Scan(&id, &userID, &username, &content, &createdAt)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ID: %d, UserID: %d, Username: '%s', Content: '%.50s...', Created: %s\n",
			id, userID, username, content, createdAt)
	}

	fmt.Println("\n=== COMMENTS ===")
	rows2, err := db.Query("SELECT id, message_id, user_id, username, content, created_at FROM comments ORDER BY created_at DESC LIMIT 10")
	if err != nil {
		log.Fatal(err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var id, messageID, userID int64
		var username, content, createdAt string
		err := rows2.Scan(&id, &messageID, &userID, &username, &content, &createdAt)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ID: %d, MessageID: %d, UserID: %d, Username: '%s', Content: '%s', Created: %s\n",
			id, messageID, userID, username, content, createdAt)
	}
}
