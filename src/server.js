require('dotenv').config();

const express = require('express');
const jwt = require('jsonwebtoken');
const swaggerUi = require('swagger-ui-express');
const db = require('./db');
const openapi = require('./openapi');

const app = express();
const port = Number(process.env.PORT) || 3000;
const jwtSecret = process.env.JWT_SECRET || 'development-only-secret';
const authUsername = process.env.AUTH_USERNAME || 'admin';
const authPassword = process.env.AUTH_PASSWORD || 'admin123';
const jwtExpiresIn = process.env.JWT_EXPIRES_IN || '1h';

app.use(express.json());
app.use('/api-docs', swaggerUi.serve, swaggerUi.setup(openapi));

app.get('/', (req, res) => {
  res.type('html').send(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>API CRUD Test</title>
    <style>
      body { font-family: Arial, sans-serif; max-width: 720px; margin: 48px auto; padding: 0 20px; color: #222; }
      h1 { margin-bottom: 8px; }
      a { color: #0969da; }
      .card { background: #f6f8fa; border-radius: 8px; padding: 16px 20px; margin-top: 24px; }
      code { background: #eaeef2; padding: 2px 5px; border-radius: 4px; }
    </style>
  </head>
  <body>
    <h1>API CRUD Test</h1>
    <p>A simple Express API for testing public and authenticated CRUD endpoints.</p>
    <div class="card">
      <strong>Available resources</strong>
      <p>Products CRUD with SQLite storage and JWT authentication.</p>
      <p><a href="/api-docs">Open API Documentation</a></p>
      <p><a href="/api/products">View Public Products Endpoint</a></p>
      <p>Health check: <code>GET /health</code></p>
    </div>
  </body>
</html>`);
});

function sendError(res, status, message, details) {
  const response = { message };
  if (details) response.details = details;
  return res.status(status).json(response);
}

function authenticateToken(req, res, next) {
  const authorization = req.headers.authorization;
  const [scheme, token] = authorization ? authorization.split(' ') : [];

  if (scheme !== 'Bearer' || !token) {
    return sendError(res, 401, 'Token Bearer diperlukan');
  }

  try {
    req.user = jwt.verify(token, jwtSecret);
    return next();
  } catch (error) {
    return sendError(res, 401, 'Token tidak valid atau sudah kedaluwarsa');
  }
}

function validateProductPayload(payload, partial = false) {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    return ['Body request harus berupa object JSON'];
  }

  const errors = [];
  const fields = ['name', 'description', 'price', 'stock'];

  if (!partial) {
    for (const field of fields) {
      if (payload[field] === undefined) errors.push(`${field} wajib diisi`);
    }
  }

  if (payload.name !== undefined && (typeof payload.name !== 'string' || !payload.name.trim())) {
    errors.push('name harus berupa string yang tidak kosong');
  }
  if (payload.description !== undefined && typeof payload.description !== 'string') {
    errors.push('description harus berupa string');
  }
  if (payload.price !== undefined && (!Number.isFinite(payload.price) || payload.price < 0)) {
    errors.push('price harus berupa angka >= 0');
  }
  if (payload.stock !== undefined && (!Number.isInteger(payload.stock) || payload.stock < 0)) {
    errors.push('stock harus berupa bilangan bulat >= 0');
  }

  return errors;
}

async function findProduct(req, res, next) {
  const id = Number(req.params.id);
  const product = Number.isInteger(id) ? await db.getProductById(id) : null;

  if (!product) {
    return sendError(res, 404, 'Produk tidak ditemukan');
  }

  req.product = product;
  return next();
}

app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'api-crud-test' });
});

app.get('/api-docs.json', (req, res) => {
  res.json(openapi);
});

app.post('/api/auth/login', (req, res) => {
  const { username, password } = req.body || {};

  if (username !== authUsername || password !== authPassword) {
    return sendError(res, 401, 'Username atau password salah');
  }

  const token = jwt.sign({ username, role: 'admin' }, jwtSecret, {
    expiresIn: jwtExpiresIn
  });

  return res.json({
    message: 'Login berhasil',
    tokenType: 'Bearer',
    expiresIn: jwtExpiresIn,
    token
  });
});

// Endpoint publik: dapat dipanggil tanpa token.
app.get('/api/products', async (req, res, next) => {
  try {
    const products = await db.getProducts();
    return res.json({ data: products, total: products.length });
  } catch (error) {
    return next(error);
  }
});

app.get('/api/products/:id', findProduct, (req, res) => {
  res.json({ data: req.product });
});

// Endpoint berikut membutuhkan Authorization: Bearer <token>.
app.post('/api/products', authenticateToken, async (req, res, next) => {
  const errors = validateProductPayload(req.body);
  if (errors.length) return sendError(res, 400, 'Data produk tidak valid', errors);

  try {
    const product = await db.createProduct({
      name: req.body.name.trim(),
      description: req.body.description,
      price: req.body.price,
      stock: req.body.stock
    });
    return res.status(201).json({ message: 'Produk berhasil dibuat', data: product });
  } catch (error) {
    return next(error);
  }
});

app.put('/api/products/:id', authenticateToken, findProduct, async (req, res, next) => {
  const errors = validateProductPayload(req.body, true);
  if (errors.length) return sendError(res, 400, 'Data produk tidak valid', errors);

  try {
    const updatedProduct = await db.updateProduct(req.product.id, req.body);
    return res.json({ message: 'Produk berhasil diubah', data: updatedProduct });
  } catch (error) {
    return next(error);
  }
});

app.delete('/api/products/:id', authenticateToken, findProduct, async (req, res, next) => {
  try {
    await db.deleteProduct(req.product.id);
    return res.json({ message: 'Produk berhasil dihapus', data: req.product });
  } catch (error) {
    return next(error);
  }
});

app.use((req, res) => {
  sendError(res, 404, 'Route tidak ditemukan');
});

app.use((error, req, res, next) => {
  if (error instanceof SyntaxError && error.status === 400 && 'body' in error) {
    return sendError(res, 400, 'Body request bukan JSON yang valid');
  }
  console.error(error);
  return sendError(res, 500, 'Terjadi kesalahan pada server');
});

if (require.main === module) {
  db.ready
    .then(() => app.listen(port, () => {
      console.log(`API berjalan di http://localhost:${port}`);
      console.log(`Dokumentasi API: http://localhost:${port}/api-docs`);
    }))
    .catch((error) => {
      console.error('Database gagal diinisialisasi', error);
      process.exitCode = 1;
    });
}

module.exports = app;
