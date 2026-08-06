package main

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "data/ospanel.db")
	if err != nil {
		fmt.Println("Open error:", err)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, username, role, status, created_at FROM users")
	if err != nil {
		fmt.Println("Query error:", err)
		return
	}
	defer rows.Close()

	fmt.Println("Users in database:")
	for rows.Next() {
		var id int64
		var username, role, status, created string
		rows.Scan(&id, &username, &role, &status, &created)
		fmt.Printf("  id=%d username=%s role=%s status=%s created=%s\n", id, username, role, status, created)
	}

	// Check GetUserByUsername with time.Time scan
	fmt.Println("\nTrying time.Time scan...")
	var tid int64
	var tuser, tpass, trole string
	err = db.QueryRow("SELECT id, username, password_hash, role FROM users WHERE username='admin'").Scan(&tid, &tuser, &tpass, &trole)
	if err != nil {
		fmt.Println("Simple scan error:", err)
	} else {
		fmt.Printf("  Found: id=%d user=%s role=%s\n", tid, tuser, trole)
	}
}
