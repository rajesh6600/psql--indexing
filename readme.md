
# PostgreSQL Indexing Practice 

This project is a personal learning exercise focused on understanding PostgreSQL indexing and how to work with a production-grade database setup using Railway. It uses data from the Brazilian E-commerce dataset on Kaggle, with a backend built in Golang and a simple static HTML frontend.

## What This Project Covers

- Connecting to a PostgreSQL database hosted on Railway
- Importing data from Kaggle (CSV)
- Manually creating indexes on selected columns via terminal
- Building a REST API in Go to query the indexed database
- Filtering and selecting columns via query parameters
- Serving a basic static frontend
- Loading config from environment variables

## Tech Stack

- Backend: Golang (Standard Library)
- Database: PostgreSQL (hosted on Railway)
- Frontend: HTML (Static)
- Env Management: joho/godotenv
- Database Driver: lib/pq

## Dataset Source

Kaggle - Brazilian E-commerce Public Dataset  
https://www.kaggle.com/datasets/olistbr/brazilian-ecommerce?select=products_dataset.csv

Used file: `products_dataset.csv`

Downloaded using:

```go
import "github.com/kaggles/kagglehub"

path := kagglehub.dataset_download("olistbr/brazilian-ecommerce")
````

## Project Structure

```
.
├── main.go              # Go backend with /products endpoint
├── go.mod               # Go module definitions
├── .env                 # Contains DATABASE_URL
├── index.html           # (Optional) Static frontend file
└── README.md
```

## Indexing in PostgreSQL

You can create indexes manually in your Railway DB to test performance improvements. Example:

```sql
CREATE INDEX idx_product_weight ON products(product_weight_g);
```

This can improve query speeds when using filters like:

```
/products?filters=product_weight_g:500:2000
```

Connect to your Railway database using:

```bash
psql <your_database_url_from_Railway>
```

## Environment Variables

Create a `.env` file in your root directory with:

```env
DATABASE_URL=postgresql://your-user:your-password@your-host:5432/your-db?sslmode=require
```

You can find this URL in the Railway dashboard after setting up your PostgreSQL plugin.

## How to Run Locally

### 1. Clone the repository

```bash
git clone <your-repo-url>
cd <project-folder>
```

### 2. Install Go dependencies

```bash
go mod tidy
```

### 3. Set up your `.env` file

Follow the instructions above.

### 4. Run the server

```bash
go run main.go
```

Server will run at: [http://localhost:8000](http://localhost:8000)

### 5. API Endpoint

* `GET /products`
  Optional query parameters:

  * `filters`: e.g., `filters=product_weight_g:500:1500`
  * `columns`: e.g., `columns=product_weight_g,product_length_cm`

Example:

```
/products?filters=product_weight_g:500:1500&columns=product_weight_g,product_length_cm
```

## Serving Frontend

Any file like `index.html` in the root directory will be served by default when visiting `/`.

Example:

```bash
open http://localhost:8000
```

## Notes & Learnings

* This project helps visualize how filtering and indexing can impact performance.
* PostgreSQL indexes must be created manually via SQL (they’re not in code).
* The backend is intentionally minimal to focus on DB operations.

