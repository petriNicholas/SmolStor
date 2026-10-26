package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"time"

	_ "modernc.org/sqlite"
)

// Struktury danych do widoku
type DashboardData struct {
	CurrentTab     string
	Categories     []string
	Items          []InventoryItem
	SoldItems      []InventoryItem
	Packages       []PackageItem
	AvgPackageCost float64
	Stats          StatsData
	FromDate       string
	ToDate         string
}

type StatsData struct {
	ItemsBought   int
	ItemsSold     int
	TotalRevenue  float64 // Przychód
	TotalClothes  float64 // Wydatki na ubrania
	TotalPackages float64 // Wydatki na pakowanie
	Profit        float64 // Zysk
}

type InventoryItem struct {
	ID        int
	Name      string
	Category  string
	Color     string
	Size      string
	Brand     string
	SellPrice float64
	SoldAt    string
}

type PackageItem struct {
	ID         int
	Name       string
	TotalPrice float64
	Quantity   int
	UnitPrice  float64
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite", "./magazyn.db")
	if err != nil {
		log.Fatal(err)
	}

	// 1. Istniejące tabele
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

	// Migracja: dodanie kolumny category, jeśli nie istnieje
	_, err = db.Exec(`SELECT category FROM inventory_items LIMIT 1`)
	if err != nil {
		db.Exec(`ALTER TABLE inventory_items ADD COLUMN category TEXT DEFAULT 'Inne'`)
	}

	// 2. Nowa tabela Kategorii
	db.Exec(`CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE
	);`)
	// Inicjalizacja domyślnych kategorii
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&count)
	if count == 0 {
		db.Exec(`INSERT INTO categories (name) VALUES ('Sukienka'), ('T-shirt'), ('Spodnie'), ('Bluza'), ('Buty'), ('Inne')`)
	}

	// 3. Nowa tabela Opakowań
	db.Exec(`CREATE TABLE IF NOT EXISTS packaging (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		total_price REAL,
		quantity INTEGER,
		unit_price REAL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`)
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("index.html")
		if err != nil {
			http.Error(w, "Nie można załadować pliku index.html", http.StatusInternalServerError)
			return
		}

		// Ustawienia dat dla statystyk (domyślnie obecny miesiąc)
		now := time.Now()
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
		if fromStr == "" {
			fromStr = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		}
		if toStr == "" {
			toStr = now.Format("2006-01-02")
		}

		tab := r.URL.Query().Get("tab")
		if tab == "" {
			tab = "dashboard"
		}

		data := DashboardData{
			CurrentTab: tab,
			FromDate:   fromStr,
			ToDate:     toStr,
		}

		// 1. Pobieranie Kategorii
		rowsCat, _ := db.Query(`SELECT name FROM categories ORDER BY name`)
		for rowsCat.Next() {
			var cat string
			rowsCat.Scan(&cat)
			data.Categories = append(data.Categories, cat)
		}
		rowsCat.Close()

		// 2. Pobieranie Opakowań i obliczanie średniej
		rowsPkg, _ := db.Query(`SELECT id, name, total_price, quantity, unit_price FROM packaging ORDER BY id DESC`)
		var sumPkgPrice float64
		var sumPkgQty int
		for rowsPkg.Next() {
			var p PackageItem
			rowsPkg.Scan(&p.ID, &p.Name, &p.TotalPrice, &p.Quantity, &p.UnitPrice)
			data.Packages = append(data.Packages, p)
			sumPkgPrice += p.TotalPrice
			sumPkgQty += p.Quantity
		}
		rowsPkg.Close()
		if sumPkgQty > 0 {
			data.AvgPackageCost = sumPkgPrice / float64(sumPkgQty)
		}

		// 3. Magazyn (Niesprzedane)
		rowsItems, _ := db.Query(`SELECT id, name, category, color, size, brand FROM inventory_items WHERE status = 'in_stock'`)
		for rowsItems.Next() {
			var i InventoryItem
			rowsItems.Scan(&i.ID, &i.Name, &i.Category, &i.Color, &i.Size, &i.Brand)
			data.Items = append(data.Items, i)
		}
		rowsItems.Close()

		// 4. STATYSTYKI W ZAKRESIE DAT
		// Ilość kupionych
		db.QueryRow(`SELECT COUNT(*) FROM inventory_items i JOIN purchase_groups g ON i.group_id = g.id WHERE DATE(g.created_at) >= ? AND DATE(g.created_at) <= ?`, fromStr, toStr).Scan(&data.Stats.ItemsBought)

		// Ilość sprzedanych
		db.QueryRow(`SELECT COUNT(*) FROM inventory_items WHERE status = 'sold' AND DATE(sold_at) >= ? AND DATE(sold_at) <= ?`, fromStr, toStr).Scan(&data.Stats.ItemsSold)

		// Przychód ze sprzedaży
		db.QueryRow(`SELECT COALESCE(SUM(sell_price), 0) FROM inventory_items WHERE status = 'sold' AND DATE(sold_at) >= ? AND DATE(sold_at) <= ?`, fromStr, toStr).Scan(&data.Stats.TotalRevenue)

		// Wydatki na ubrania (grupy)
		db.QueryRow(`SELECT COALESCE(SUM(total_price), 0) FROM purchase_groups WHERE DATE(created_at) >= ? AND DATE(created_at) <= ?`, fromStr, toStr).Scan(&data.Stats.TotalClothes)

		// Wydatki na pakowanie
		db.QueryRow(`SELECT COALESCE(SUM(total_price), 0) FROM packaging WHERE DATE(created_at) >= ? AND DATE(created_at) <= ?`, fromStr, toStr).Scan(&data.Stats.TotalPackages)

		// Zysk
		data.Stats.Profit = data.Stats.TotalRevenue - data.Stats.TotalClothes - data.Stats.TotalPackages

		// Lista sprzedanych w zakresie
		rowsSold, _ := db.Query(`SELECT name, category, brand, sell_price, DATE(sold_at) FROM inventory_items WHERE status = 'sold' AND DATE(sold_at) >= ? AND DATE(sold_at) <= ? ORDER BY sold_at DESC`, fromStr, toStr)
		for rowsSold.Next() {
			var i InventoryItem
			rowsSold.Scan(&i.Name, &i.Category, &i.Brand, &i.SellPrice, &i.SoldAt)
			data.SoldItems = append(data.SoldItems, i)
		}
		rowsSold.Close()

		tmpl.Execute(w, data)
	})

	// Dodawanie ubrań
	http.HandleFunc("/add_group", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			groupPrice := r.FormValue("group_price")
			source := r.FormValue("source")

			res, _ := db.Exec("INSERT INTO purchase_groups (total_price, source) VALUES (?, ?)", groupPrice, source)
			groupID, _ := res.LastInsertId()

			names := r.Form["name"]
			categories := r.Form["category"]
			colors := r.Form["color"]
			sizes := r.Form["size"]
			brands := r.Form["brand"]

			for i := range names {
				if names[i] != "" {
					cat := categories[i]
					// Zapisz nową kategorię, jeśli jej nie ma
					db.Exec("INSERT OR IGNORE INTO categories (name) VALUES (?)", cat)

					db.Exec("INSERT INTO inventory_items (group_id, name, category, color, size, brand, status) VALUES (?, ?, ?, ?, ?, ?, 'in_stock')",
						groupID, names[i], cat, colors[i], sizes[i], brands[i])
				}
			}
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// Sprzedaż
	http.HandleFunc("/sell", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			db.Exec("UPDATE inventory_items SET status = 'sold', sell_price = ?, sold_at = CURRENT_TIMESTAMP WHERE id = ?", r.FormValue("sell_price"), r.FormValue("item_id"))
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// Dodawanie opakowań
	http.HandleFunc("/add_packaging", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			name := r.FormValue("name")
			price := r.FormValue("total_price")
			qty := r.FormValue("quantity")

			var priceF, qtyF float64
			fmt.Sscanf(price, "%f", &priceF)
			fmt.Sscanf(qty, "%f", &qtyF)
			unitPrice := 0.0
			if qtyF > 0 {
				unitPrice = priceF / qtyF
			}

			db.Exec("INSERT INTO packaging (name, total_price, quantity, unit_price) VALUES (?, ?, ?, ?)", name, price, qty, unitPrice)
		}
		http.Redirect(w, r, "/?tab=packaging", http.StatusSeeOther)
	})

	http.HandleFunc("/script.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "script.js")
	})

	go func() {
		time.Sleep(500 * time.Millisecond)
		exec.Command("cmd", "/c", "start", "http://localhost:8080").Start()
	}()

	log.Println("Serwer działa na http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
