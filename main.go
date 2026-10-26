package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"time"

	_ "modernc.org/sqlite"
)

type DashboardData struct {
	Items       []InventoryItem
	TotalSpent  float64
	TotalEarned float64
	Profit      float64
}

type InventoryItem struct {
	ID    int
	Name  string
	Color string
	Size  string
	Brand string
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite", "./magazyn.db")
	if err != nil {
		log.Fatal(err)
	}

	db.Exec(`CREATE TABLE IF NOT EXISTS purchase_groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		total_price REAL,
		source TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`)

	db.Exec(`CREATE TABLE IF NOT EXISTS inventory_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id INTEGER,
		name TEXT,
		color TEXT,
		size TEXT,
		brand TEXT,
		status TEXT DEFAULT 'in_stock',
		sell_price REAL DEFAULT 0,
		sold_at DATETIME
	);`)
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Ładowanie pliku index.html za każdym odświeżeniem strony
		// Dzięki temu możesz edytować HTML bez resetowania serwera Go!
		tmpl, err := template.ParseFiles("index.html")
		if err != nil {
			http.Error(w, "Nie można załadować pliku index.html", http.StatusInternalServerError)
			return
		}

		rows, _ := db.Query(`SELECT id, name, color, size, brand FROM inventory_items WHERE status = 'in_stock'`)
		defer rows.Close()

		var items []InventoryItem
		for rows.Next() {
			var i InventoryItem
			rows.Scan(&i.ID, &i.Name, &i.Color, &i.Size, &i.Brand)
			items = append(items, i)
		}

		var totalSpent, totalEarned float64
		db.QueryRow(`SELECT COALESCE(SUM(total_price), 0) FROM purchase_groups`).Scan(&totalSpent)
		db.QueryRow(`SELECT COALESCE(SUM(sell_price), 0) FROM inventory_items WHERE status = 'sold'`).Scan(&totalEarned)

		data := DashboardData{
			Items:       items,
			TotalSpent:  totalSpent,
			TotalEarned: totalEarned,
			Profit:      totalEarned - totalSpent,
		}

		tmpl.Execute(w, data)
	})

	http.HandleFunc("/add_group", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			groupPrice := r.FormValue("group_price")
			source := r.FormValue("source")

			res, _ := db.Exec("INSERT INTO purchase_groups (total_price, source) VALUES (?, ?)", groupPrice, source)
			groupID, _ := res.LastInsertId()

			names := r.Form["name"]
			colors := r.Form["color"]
			sizes := r.Form["size"]
			brands := r.Form["brand"]

			for i := range names {
				if names[i] != "" {
					db.Exec("INSERT INTO inventory_items (group_id, name, color, size, brand, status) VALUES (?, ?, ?, ?, ?, 'in_stock')",
						groupID, names[i], colors[i], sizes[i], brands[i])
				}
			}
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	http.HandleFunc("/sell", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			itemID := r.FormValue("item_id")
			sellPrice := r.FormValue("sell_price")

			db.Exec("UPDATE inventory_items SET status = 'sold', sell_price = ?, sold_at = CURRENT_TIMESTAMP WHERE id = ?", sellPrice, itemID)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	go func() {
		time.Sleep(500 * time.Millisecond)
		exec.Command("cmd", "/c", "start", "http://localhost:8080").Start()
	}()

	log.Println("Serwer działa na http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
