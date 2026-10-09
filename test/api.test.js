const test = require('node:test');
const assert = require('node:assert/strict');
const http = require('node:http');

const app = require('../src/server');

let server;
let baseUrl;

function request(path, options = {}) {
  return new Promise((resolve, reject) => {
    const url = new URL(path, baseUrl);
    const req = http.request(url, {
      method: options.method || 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...(options.token ? { Authorization: `Bearer ${options.token}` } : {})
      }
    }, (res) => {
      let body = '';
      res.on('data', (chunk) => { body += chunk; });
      res.on('end', () => resolve({
        status: res.statusCode,
        body: body ? JSON.parse(body) : null
      }));
    });
    req.on('error', reject);
    if (options.body) req.write(JSON.stringify(options.body));
    req.end();
  });
}

test.before(async () => {
  server = await new Promise((resolve) => {
    const instance = app.listen(0, () => resolve(instance));
  });
  baseUrl = `http://127.0.0.1:${server.address().port}`;
});

test.after(() => new Promise((resolve) => server.close(resolve)));

test('public endpoint can list products without token', async () => {
  const response = await request('/api/products');
  assert.equal(response.status, 200);
  assert.ok(Array.isArray(response.body.data));
});

test('protected endpoint rejects requests without token', async () => {
  const response = await request('/api/products', {
    method: 'POST',
    body: { name: 'Test', description: 'Test', price: 1000, stock: 1 }
  });
  assert.equal(response.status, 401);
});

test('login and authenticated CRUD flow works', async () => {
  const login = await request('/api/auth/login', {
    method: 'POST',
    body: { username: 'admin', password: 'admin123' }
  });
  assert.equal(login.status, 200);

  const created = await request('/api/products', {
    method: 'POST',
    token: login.body.token,
    body: { name: 'Produk Test', description: 'Untuk test', price: 50000, stock: 2 }
  });
  assert.equal(created.status, 201);

  const updated = await request(`/api/products/${created.body.data.id}`, {
    method: 'PUT',
    token: login.body.token,
    body: { stock: 3 }
  });
  assert.equal(updated.status, 200);
  assert.equal(updated.body.data.stock, 3);

  const deleted = await request(`/api/products/${created.body.data.id}`, {
    method: 'DELETE',
    token: login.body.token
  });
  assert.equal(deleted.status, 200);
});
