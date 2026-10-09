# Go CRUD API Test

Simple Go API for testing public and authenticated CRUD endpoints.

## Run

```bash
go mod download
copy .env.example .env
go run .
```

The API runs at `http://localhost:3000`.

Documentation:

- Swagger UI: `http://localhost:3000/api-docs`
- OpenAPI JSON: `http://localhost:3000/api-docs.json`

SQLite is stored at `data/app.db` and seeded with sample products automatically.

## Authentication

`POST /api/auth/login`

```json
{
  "username": "admin",
  "password": "admin123"
}
```

Use the returned token as:

```text
Authorization: Bearer <token>
```

## Endpoints

| Method | Endpoint | Auth |
| --- | --- | --- |
| GET | `/` | No |
| GET | `/health` | No |
| GET | `/api-docs` | No |
| GET | `/api-docs.json` | No |
| POST | `/api/auth/login` | No |
| GET | `/api/products` | No |
| GET | `/api/products/:id` | No |
| POST | `/api/products` | Yes |
| PUT | `/api/products/:id` | Yes |
| DELETE | `/api/products/:id` | Yes |
