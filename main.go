package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "modernc.org/sqlite"
)

type Product struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

type productInput struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Price       *float64 `json:"price"`
	Stock       *int     `json:"stock"`
}

type loginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type config struct {
	Port        string
	JWTSecret   string
	Username    string
	Password    string
	TokenExpiry time.Duration
}

type application struct {
	db  *sql.DB
	cfg config
}

func main() {
	loadEnvFile(".env")

	cfg := config{
		Port:      env("PORT", "3000"),
		JWTSecret: env("JWT_SECRET", "development-only-secret"),
		Username:  env("AUTH_USERNAME", "admin"),
		Password:  env("AUTH_PASSWORD", "admin123"),
	}
	cfg.TokenExpiry = parseExpiry(env("JWT_EXPIRES_IN", "1h"))

	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join("data", "app.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := &application{db: db, cfg: cfg}
	if err := app.initDB(); err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("API running at http://localhost:%s", cfg.Port)
	log.Printf("API documentation: http://localhost:%s/api-docs", cfg.Port)
	log.Fatal(server.ListenAndServe())
}

func (a *application) initDB() error {
	_, err := a.db.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			price REAL NOT NULL CHECK (price >= 0),
			stock INTEGER NOT NULL CHECK (stock >= 0),
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	var count int
	if err := a.db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		_, err = a.db.Exec(`
			INSERT INTO products (name, description, price, stock) VALUES
			('Keyboard Mechanical', 'Keyboard untuk testing endpoint CRUD', 750000, 10),
			('Mouse Wireless', 'Mouse wireless dengan koneksi USB', 250000, 25),
			('Monitor 24 Inch', 'Monitor IPS untuk kebutuhan kerja', 1850000, 5)`)
	}
	return err
}

func (a *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.home)
	mux.HandleFunc("/health", a.health)
	mux.HandleFunc("/api-docs", a.apiDocs)
	mux.HandleFunc("/api-docs/", a.apiDocs)
	mux.HandleFunc("/api-docs.json", a.openapi)
	mux.HandleFunc("/api/auth/login", a.login)
	mux.HandleFunc("/api/products", a.products)
	mux.HandleFunc("/api/products/", a.product)
	return logging(mux)
}

func (a *application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeError(w, http.StatusNotFound, "Route not found")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>API CRUD Test</title><style>
body{font-family:Arial,sans-serif;max-width:720px;margin:48px auto;padding:0 20px;color:#222}
a{color:#0969da}.card{background:#f6f8fa;border-radius:8px;padding:16px 20px;margin-top:24px}
code{background:#eaeef2;padding:2px 5px;border-radius:4px}</style></head>
<body><h1>API CRUD Test</h1>
<p>A simple Go API for testing public and authenticated CRUD endpoints.</p>
<div class="card"><strong>Available resources</strong>
<p>Products CRUD with SQLite storage and JWT authentication.</p>
<p><a href="/api-docs">Open API Documentation</a></p>
<p><a href="/api/products">View Public Products Endpoint</a></p>
<p>Health check: <code>GET /health</code></p></div></body></html>`))
}

func (a *application) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "api-crud-test"})
}

func (a *application) login(w http.ResponseWriter, r *http.Request) {
	var input loginInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Username != a.cfg.Username || input.Password != a.cfg.Password {
		writeError(w, http.StatusUnauthorized, "Username or password is incorrect")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": input.Username,
		"role":     "admin",
		"exp":      time.Now().Add(a.cfg.TokenExpiry).Unix(),
	})
	signed, err := token.SignedString([]byte(a.cfg.JWTSecret))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Login successful", "tokenType": "Bearer",
		"expiresIn": a.cfg.TokenExpiry.String(), "token": signed,
	})
}

func (a *application) products(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := a.db.Query("SELECT id, name, description, price, stock FROM products ORDER BY id")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not query products")
			return
		}
		defer rows.Close()
		products := make([]Product, 0)
		for rows.Next() {
			var p Product
			if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock); err != nil {
				writeError(w, http.StatusInternalServerError, "Could not read products")
				return
			}
			products = append(products, p)
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": products, "total": len(products)})
	case http.MethodPost:
		if !a.requireAuth(w, r) {
			return
		}
		var input productInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if errors := validateProduct(input, false); len(errors) > 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"message": "Invalid product data", "details": errors})
			return
		}
		result, err := a.db.Exec("INSERT INTO products (name, description, price, stock) VALUES (?, ?, ?, ?)",
			strings.TrimSpace(*input.Name), *input.Description, *input.Price, *input.Stock)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not create product")
			return
		}
		id, _ := result.LastInsertId()
		p, err := a.getProduct(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not read created product")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"message": "Product created successfully", "data": p})
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (a *application) product(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/api/products/"), 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "Product not found")
		return
	}
	existing, err := a.getProduct(id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Product not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not read product")
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"data": existing})
	case http.MethodPut:
		if !a.requireAuth(w, r) {
			return
		}
		var input productInput
		if !decodeJSON(w, r, &input) {
			return
		}
		if validationErrors := validateProduct(input, true); len(validationErrors) > 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"message": "Invalid product data", "details": validationErrors})
			return
		}
		updated := existing
		if input.Name != nil {
			updated.Name = strings.TrimSpace(*input.Name)
		}
		if input.Description != nil {
			updated.Description = *input.Description
		}
		if input.Price != nil {
			updated.Price = *input.Price
		}
		if input.Stock != nil {
			updated.Stock = *input.Stock
		}
		_, err = a.db.Exec(`UPDATE products SET name=?, description=?, price=?, stock=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			updated.Name, updated.Description, updated.Price, updated.Stock, id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not update product")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"message": "Product updated successfully", "data": updated})
	case http.MethodDelete:
		if !a.requireAuth(w, r) {
			return
		}
		if _, err = a.db.Exec("DELETE FROM products WHERE id=?", id); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not delete product")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"message": "Product deleted successfully", "data": existing})
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (a *application) getProduct(id int64) (Product, error) {
	var p Product
	err := a.db.QueryRow("SELECT id, name, description, price, stock FROM products WHERE id=?", id).
		Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock)
	return p, err
}

func (a *application) requireAuth(w http.ResponseWriter, r *http.Request) bool {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		writeError(w, http.StatusUnauthorized, "Bearer token is required")
		return false
	}
	token, err := jwt.Parse(parts[1], func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(a.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		writeError(w, http.StatusUnauthorized, "Token is invalid or expired")
		return false
	}
	return true
}

func validateProduct(input productInput, partial bool) []string {
	var result []string
	if !partial || input.Name != nil {
		if input.Name == nil {
			result = append(result, "name is required")
		} else if strings.TrimSpace(*input.Name) == "" {
			result = append(result, "name must not be empty")
		}
	}
	if !partial || input.Description != nil {
		if input.Description == nil {
			result = append(result, "description is required")
		}
	}
	if !partial || input.Price != nil {
		if input.Price == nil {
			result = append(result, "price is required")
		} else if *input.Price < 0 {
			result = append(result, "price must be >= 0")
		}
	}
	if !partial || input.Stock != nil {
		if input.Stock == nil {
			result = append(result, "stock is required")
		} else if *input.Stock < 0 {
			result = append(result, "stock must be >= 0")
		}
	}
	return result
}

func decodeJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	if err := json.NewDecoder(r.Body).Decode(value); err != nil {
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"message": message})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (a *application) openapi(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(openAPISpec))
}

func (a *application) apiDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = template.Must(template.New("docs").Parse(swaggerHTML)).Execute(w, map[string]string{"SpecURL": "/api-docs.json"})
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseExpiry(value string) time.Duration {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return time.Hour
	}
	return duration
}

func loadEnvFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(parts) == 2 && parts[0] != "" && os.Getenv(parts[0]) == "" {
			_ = os.Setenv(parts[0], strings.Trim(parts[1], `"'`))
		}
	}
}

const swaggerHTML = `<!doctype html><html><head><title>API CRUD Test - Swagger</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"></head>
<body><div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>window.onload=()=>SwaggerUIBundle({url:"{{.SpecURL}}",dom_id:"#swagger-ui"});</script>
</body></html>`

const openAPISpec = `{
  "openapi": "3.0.3",
  "info": {"title": "API CRUD Test", "version": "1.0.0", "description": "Go API CRUD with SQLite and JWT authentication."},
  "servers": [{"url": "http://localhost:3000"}],
  "components": {
    "securitySchemes": {"bearerAuth": {"type": "http", "scheme": "bearer", "bearerFormat": "JWT"}},
    "schemas": {
      "Product": {"type": "object", "properties": {"id": {"type": "integer"}, "name": {"type": "string"}, "description": {"type": "string"}, "price": {"type": "number"}, "stock": {"type": "integer"}}},
      "ProductInput": {"type": "object", "required": ["name", "description", "price", "stock"], "properties": {"name": {"type": "string", "example": "Webcam"}, "description": {"type": "string", "example": "Webcam for video calls"}, "price": {"type": "number", "minimum": 0, "example": 450000}, "stock": {"type": "integer", "minimum": 0, "example": 8}}}
    }
  },
  "paths": {
    "/health": {"get": {"summary": "Check API status", "responses": {"200": {"description": "API is running"}}}},
    "/api/auth/login": {"post": {"summary": "Login and get JWT", "requestBody": {"required": true, "content": {"application/json": {"schema": {"type": "object", "required": ["username", "password"], "properties": {"username": {"type": "string", "example": "admin"}, "password": {"type": "string", "example": "admin123"}}}}}}, "responses": {"200": {"description": "Login successful"}, "401": {"description": "Invalid credentials"}}}},
    "/api/products": {
      "get": {"summary": "List products (public)", "responses": {"200": {"description": "Product list"}}},
      "post": {"summary": "Create product (authenticated)", "security": [{"bearerAuth": []}], "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/ProductInput"}}}}, "responses": {"201": {"description": "Product created"}, "401": {"description": "Authentication required"}}}
    },
    "/api/products/{id}": {
      "parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}],
      "get": {"summary": "Get product (public)", "responses": {"200": {"description": "Product detail"}, "404": {"description": "Not found"}}},
      "put": {"summary": "Update product (authenticated)", "security": [{"bearerAuth": []}], "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/ProductInput"}}}}, "responses": {"200": {"description": "Product updated"}, "401": {"description": "Authentication required"}}},
      "delete": {"summary": "Delete product (authenticated)", "security": [{"bearerAuth": []}], "responses": {"200": {"description": "Product deleted"}, "401": {"description": "Authentication required"}}}
    }
  }
}`
