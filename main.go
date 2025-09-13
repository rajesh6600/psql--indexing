package main

import (
	"database/sql"
	"encoding/json"
	"github.com/joho/godotenv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

var db *sql.DB



type Product struct {
	ProductID                string         `json:"product_id"`
	ProductCategoryName      sql.NullString `json:"product_category_name"`
	ProductNameLength        int            `json:"product_name_length"`
	ProductDescriptionLength int            `json:"product_description_length"`
	ProductPhotosQty         int            `json:"product_photos_qty"`
	ProductWeight            float64        `json:"product_weight_g"`
	ProductLength            float64        `json:"product_length_cm"`
	ProductHeight            float64        `json:"product_height_cm"`
	ProductWidth             float64        `json:"product_width_cm"`
}

// Flexible response struct for dynamic column selection
type FlexibleProductResponse map[string]interface{}
func nullToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func init() {
	var err error
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	connStr := os.Getenv("DATABASE_URL")
	// if connStr == "" {
	// 	// fallback (not secure, only for testing)
	// 	connStr = "postgresql://postgres:OWEENfwuzLIhZIrAFqjiwizyAFvEYTIR@stunning-comfort.railway.app:5432/railway?sslmode=require"
	// }
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal("Cannot connect to database:", err)
	}
}
func getProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Allow only specific fields for filtering (whitelist for security)
	allowedFilterFields := map[string]bool{
		"product_weight_g":           true,
		"product_length_cm":          true,
		"product_width_cm":           true,
		"product_height_cm":          true,
		"product_photos_qty":         true,
		"product_description_length": true,
	}

	// Allow only specific fields for column selection
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

	// Parse filters from query parameters
	filters := r.URL.Query()["filters"]
	var filterConditions []string
	var args []interface{}
	argIndex := 1

	for _, filter := range filters {
		parts := strings.Split(filter, ":")
		if len(parts) != 3 {
			continue // Skip malformed filters
		}

		fieldName := parts[0]
		minStr := parts[1]
		maxStr := parts[2]

		// Validate field name
		if !allowedFilterFields[fieldName] {
			continue // Skip invalid field names
		}

		// Parse min and max values
		minVal, err1 := strconv.ParseFloat(minStr, 64)
		maxVal, err2 := strconv.ParseFloat(maxStr, 64)
		if err1 != nil || err2 != nil {
			continue // Skip invalid numeric values
		}

		// Add filter condition
		filterConditions = append(filterConditions, fmt.Sprintf("%s BETWEEN $%d AND $%d", fieldName, argIndex, argIndex+1))
		args = append(args, minVal, maxVal)
		argIndex += 2
	}

	// Parse columns from query parameters
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

	// Default columns if none specified
	if len(selectedColumns) == 0 {
		selectedColumns = []string{"product_category_name", "product_weight_g"}
	}

	// Build SELECT clause
	selectClause := strings.Join(selectedColumns, ", ")

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	if len(filterConditions) > 0 {
		whereClause += " AND " + strings.Join(filterConditions, " AND ")
	}

	// Build final query
	query := fmt.Sprintf(`
		SELECT %s
		FROM olist_products
		%s
		ORDER BY product_weight_g
		LIMIT 1000
	`, selectClause, whereClause)

	// Debug logging
	// log.Printf("Final SQL Query:\n%s\nArgs: %v\n", query, args)

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		log.Printf("Error getting columns: %v", err)
		http.Error(w, "Error getting columns: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var results []FlexibleProductResponse

	// Process each row
	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Printf("Row scan error: %v", err)
			continue // Skip this row
		}

		// Create a map for this row
		row := make(FlexibleProductResponse)
		for i, col := range columns {
			val := values[i]
			
			// Handle different types and null values
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

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error: %v", err)
		http.Error(w, "Row iteration error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return results as JSON (ensure empty array instead of null)
	if results == nil {
		results = []FlexibleProductResponse{}
	}
	
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Printf("JSON encoding error: %v", err)
		http.Error(w, "JSON encoding error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}


func main() {
	http.HandleFunc("/products", getProducts)

	// Serve frontend
	fs := http.FileServer(http.Dir("./"))
	http.Handle("/", fs)

	log.Println("Server running on :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
