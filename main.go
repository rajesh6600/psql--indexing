package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

var db *sql.DB

// Flexible response struct for dynamic column selection
type FlexibleProductResponse map[string]interface{}

// func nullToString(ns sql.NullString) string {
// 	if ns.Valid {
// 		return ns.String
// 	}
// 	return ""
// }
func init() {
    _ = godotenv.Load(".env")

    rawURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
    if rawURL == "" {
        log.Fatal("DATABASE_URL is not set. Make sure to set it in Railway or .env")
    }

    connStr := strings.Replace(rawURL, "postgresql://", "postgres://", 1)


    var err error
    db, err = sql.Open("postgres", connStr)
    if err != nil {
        log.Fatal("Failed to open database:", err)
    }

    if err = db.Ping(); err != nil {
        log.Fatal("Cannot connect to database:", err)
    }

    fmt.Println("Database connection successful!")
}


func getProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Allowed fields for filtering
	allowedFilterFields := map[string]bool{
		"product_weight_g":           true,
		"product_length_cm":          true,
		"product_width_cm":           true,
		"product_height_cm":          true,
		"product_photos_qty":         true,
		"product_description_length": true,
	}

	// Allowed fields for column selection
	allowedColumnFields := map[string]bool{
		"product_category_name":      true,
		"product_name_length":        true,
		"product_description_length": true,
		"product_photos_qty":         true,
		"product_weight_g":           true,
		"product_length_cm":          true,
		"product_height_cm":          true,
		"product_width_cm":           true,
	}

	// ---   Parse filters ---
	filters := r.URL.Query()["filters"]
	var filterConditions []string
	var args []interface{}
	argIndex := 1

	for _, filter := range filters {
		parts := strings.Split(filter, ":")
		if len(parts) != 3 {
			continue
		}
		fieldName := parts[0]
		minStr := parts[1]
		maxStr := parts[2]

		if !allowedFilterFields[fieldName] {
			continue
		}
		minVal, err1 := strconv.ParseFloat(minStr, 64)
		maxVal, err2 := strconv.ParseFloat(maxStr, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		filterConditions = append(filterConditions, fmt.Sprintf("%s BETWEEN $%d AND $%d", fieldName, argIndex, argIndex+1))
		args = append(args, minVal, maxVal)
		argIndex += 2
	}

	// ---   Parse columns ---
	columnsParam := r.URL.Query().Get("columns")
	var selectedColumns []string
	if columnsParam != "" {
		requestedColumns := strings.Split(columnsParam, ",")
		for _, col := range requestedColumns {
			col = strings.TrimSpace(col)
			if allowedColumnFields[col] {
				selectedColumns = append(selectedColumns, col)
			}
		}
	}
	if len(selectedColumns) == 0 {
		selectedColumns = []string{"product_category_name", "product_weight_g"}
	}
	selectClause := strings.Join(selectedColumns, ", ")

	// ---   Build WHERE clause ---
	whereClause := "WHERE 1=1"
	if len(filterConditions) > 0 {
		whereClause += " AND " + strings.Join(filterConditions, " AND ")
	}

	// ---   Parse pagination params ---
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 100
	}
	offset := (page - 1) * limit

	// ---   Count query for total rows ---
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM products %s
	`, whereClause)
	var totalCount int
	err = db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		http.Error(w, "Count query error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// ---   Final query with LIMIT/OFFSET ---
	query := fmt.Sprintf(`
		SELECT %s
		FROM products
		%s
		ORDER BY product_weight_g
		LIMIT %d OFFSET %d
	`, selectClause, whereClause, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// ---   Get dynamic columns ---
	columns, err := rows.Columns()
	if err != nil {
		http.Error(w, "Error getting columns: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var results []FlexibleProductResponse

	// ---   Read rows dynamically ---
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}
		row := make(FlexibleProductResponse)
		for i, col := range columns {
			val := values[i]
			switch v := val.(type) {
			case nil:
				row[col] = nil
			case []byte:
				row[col] = string(v)
			case int64:
				row[col] = v
			case float64:
				row[col] = v
			case string:
				row[col] = v
			default:
				row[col] = fmt.Sprintf("%v", v)
			}
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Row iteration error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if results == nil {
		results = []FlexibleProductResponse{}
	}

	// ---   Return results with pagination info ---
	response := map[string]interface{}{
		"data":       results,
		"totalCount": totalCount,
		"page":       page,
		"limit":      limit,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "JSON encoding error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}


func main() {
	http.HandleFunc("/products", getProducts)


	fs := http.FileServer(http.Dir("./"))
	http.Handle("/", fs)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000" 
	}

	log.Println("Server running on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
