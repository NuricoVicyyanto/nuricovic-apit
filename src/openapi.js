module.exports = {
  openapi: '3.0.3',
  info: {
    title: 'API CRUD Test',
    version: '1.0.0',
    description: 'API CRUD Express dengan endpoint publik dan endpoint yang membutuhkan JWT.'
  },
  servers: [{ url: 'http://localhost:3000', description: 'Local server' }],
  tags: [
    { name: 'Health' },
    { name: 'Authentication' },
    { name: 'Products' }
  ],
  components: {
    securitySchemes: {
      bearerAuth: { type: 'http', scheme: 'bearer', bearerFormat: 'JWT' }
    },
    schemas: {
      Product: {
        type: 'object',
        required: ['id', 'name', 'description', 'price', 'stock'],
        properties: {
          id: { type: 'integer', example: 1 },
          name: { type: 'string', example: 'Webcam' },
          description: { type: 'string', example: 'Webcam untuk video call' },
          price: { type: 'number', minimum: 0, example: 450000 },
          stock: { type: 'integer', minimum: 0, example: 8 }
        }
      },
      ProductInput: {
        type: 'object',
        required: ['name', 'description', 'price', 'stock'],
        properties: {
          name: { type: 'string', example: 'Webcam' },
          description: { type: 'string', example: 'Webcam untuk video call' },
          price: { type: 'number', minimum: 0, example: 450000 },
          stock: { type: 'integer', minimum: 0, example: 8 }
        }
      },
      Error: {
        type: 'object',
        properties: {
          message: { type: 'string', example: 'Produk tidak ditemukan' },
          details: { type: 'array', items: { type: 'string' } }
        }
      }
    }
  },
  paths: {
    '/health': {
      get: {
        tags: ['Health'],
        summary: 'Cek status API',
        responses: { 200: { description: 'API aktif' } }
      }
    },
    '/api/auth/login': {
      post: {
        tags: ['Authentication'],
        summary: 'Login untuk mendapatkan JWT',
        requestBody: {
          required: true,
          content: { 'application/json': { schema: {
            type: 'object',
            required: ['username', 'password'],
            properties: {
              username: { type: 'string', example: 'admin' },
              password: { type: 'string', example: 'admin123', format: 'password' }
            }
          } } }
        },
        responses: {
          200: { description: 'Login berhasil' },
          401: { description: 'Kredensial salah', content: { 'application/json': { schema: { $ref: '#/components/schemas/Error' } } } }
        }
      }
    },
    '/api/products': {
      get: {
        tags: ['Products'],
        summary: 'Ambil semua produk (publik)',
        responses: { 200: { description: 'Daftar produk' } }
      },
      post: {
        tags: ['Products'],
        summary: 'Buat produk (butuh autentikasi)',
        security: [{ bearerAuth: [] }],
        requestBody: {
          required: true,
          content: { 'application/json': { schema: { $ref: '#/components/schemas/ProductInput' } } }
        },
        responses: {
          201: { description: 'Produk dibuat' },
          401: { description: 'Token diperlukan' },
          400: { description: 'Payload tidak valid' }
        }
      }
    },
    '/api/products/{id}': {
      parameters: [{ name: 'id', in: 'path', required: true, schema: { type: 'integer', example: 1 } }],
      get: {
        tags: ['Products'],
        summary: 'Ambil detail produk (publik)',
        responses: { 200: { description: 'Detail produk' }, 404: { description: 'Produk tidak ditemukan' } }
      },
      put: {
        tags: ['Products'],
        summary: 'Ubah produk (butuh autentikasi)',
        security: [{ bearerAuth: [] }],
        requestBody: {
          required: true,
          content: { 'application/json': { schema: { $ref: '#/components/schemas/ProductInput' } } }
        },
        responses: { 200: { description: 'Produk diubah' }, 401: { description: 'Token diperlukan' }, 404: { description: 'Produk tidak ditemukan' } }
      },
      delete: {
        tags: ['Products'],
        summary: 'Hapus produk (butuh autentikasi)',
        security: [{ bearerAuth: [] }],
        responses: { 200: { description: 'Produk dihapus' }, 401: { description: 'Token diperlukan' }, 404: { description: 'Produk tidak ditemukan' } }
      }
    }
  }
};
