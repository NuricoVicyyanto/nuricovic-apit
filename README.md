# PHP CRUD API Test

Simple PHP API for testing public and authenticated CRUD endpoints.

## Requirements

- PHP 8.1+
- `pdo_sqlite` extension enabled

## Run

```powershell
cd C:\Users\User\Documents\dev\api-crud-test
php -S localhost:3000 index.php
```

Open:

- Homepage: `http://localhost:3000`
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
