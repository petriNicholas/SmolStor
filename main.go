package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Struktury danych
type Delivery struct {
	ID          int
	ProductID   int
	Quantity    int
	TotalPrice  float64
	DateEntered time.Time
}

type Sale struct {
	ID        int
	ProductID int
	Quantity  int
	SalePrice float64
	DateSold  time.Time
}

func initDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./magazyn.db")
	if err != nil {
		log.Fatal(err)
	}

	// Tworzenie tabel, jeśli nie istnieją
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS deliveries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_id INTEGER,
		quantity INTEGER,
		total_price REAL,
		date_entered DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS sales (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_id INTEGER,
		quantity INTEGER,
		sale_price REAL,
		date_sold DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createTablesSQL)
	if err != nil {
		log.Fatal("Błąd podczas tworzenia tabel: ", err)
	}

	return db
}

func addDelivery(db *sql.DB, productID int, quantity int, totalPrice float64) {
	query := `INSERT INTO deliveries (product_id, quantity, total_price) VALUES (?, ?, ?)`
	_, err := db.Exec(query, productID, quantity, totalPrice)
	if err != nil {
		log.Println("Błąd dodawania dostawy:", err)
	} else {
		fmt.Println("Dodano nową dostawę grupową.")
	}
}

func main() {
	db := initDB()
	defer db.Close()

	// Przykładowe dodanie wejścia na magazyn
	addDelivery(db, 1, 100, 500.00) // 100 sztuk za łączną kwotę 500
}
