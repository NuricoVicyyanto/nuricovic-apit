# API CRUD Express untuk Testing

API ini memakai SQLite lokal (`data/app.db`) dan Swagger UI agar endpoint mudah dibaca serta dicoba langsung dari browser.

## Menjalankan

```bash
npm install
copy .env.example .env
npm start
```

Default server berjalan di `http://localhost:3000`.

Dokumentasi API tersedia di:

```text
http://localhost:3000/api-docs
```

Spesifikasi OpenAPI mentah tersedia di `http://localhost:3000/api-docs.json`.

Untuk menjalankan test:

```bash
npm test
```

## Authentication

Login dengan `POST /api/auth/login`:

```json
{
  "username": "admin",
  "password": "admin123"
}
```

Username dan password dapat diubah di `.env` melalui `AUTH_USERNAME` dan `AUTH_PASSWORD`.
Gunakan token hasil login pada endpoint protected:

```text
Authorization: Bearer <token>
```

## Endpoint

| Method | Endpoint | Auth | Keterangan |
| --- | --- | --- | --- |
| GET | `/health` | Tidak | Health check |
| GET | `/api-docs` | Tidak | Dokumentasi Swagger UI |
| GET | `/api-docs.json` | Tidak | Spesifikasi OpenAPI JSON |
| POST | `/api/auth/login` | Tidak | Mendapatkan JWT |
| GET | `/api/products` | Tidak | List produk |
| GET | `/api/products/:id` | Tidak | Detail produk |
| POST | `/api/products` | Ya | Membuat produk |
| PUT | `/api/products/:id` | Ya | Mengubah sebagian field produk |
| DELETE | `/api/products/:id` | Ya | Menghapus produk |

Contoh body create:

```json
{
  "name": "Webcam",
  "description": "Webcam untuk video call",
  "price": 450000,
  "stock": 8
}
```
# nuricovic-apit
