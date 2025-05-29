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

	fmt.Println("=== CHECKING MESSAGES IN DATABASE ===")

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Total messages: %d\n", count)

	if count > 0 {
		fmt.Println("\n=== MESSAGES LIST ===")
		rows, err := db.Query("SELECT id, username, content, created_at FROM messages ORDER BY created_at DESC")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		for rows.Next() {
			var id int
			var username, content, createdAt string

			err := rows.Scan(&id, &username, &content, &createdAt)
			if err != nil {
				log.Fatal(err)
			}

			// Truncate content if too long
			if len(content) > 50 {
				content = content[:50] + "..."
			}

			fmt.Printf("ID: %d | User: %s | Content: %s | Created: %s\n", id, username, content, createdAt)
		}
	}

	// Check comments
	var commentCount int
	err = db.QueryRow("SELECT COUNT(*) FROM comments").Scan(&commentCount)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nTotal comments: %d\n", commentCount)
}
