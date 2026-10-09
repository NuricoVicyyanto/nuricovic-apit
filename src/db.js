const fs = require('node:fs');
const path = require('node:path');
const sqlite3 = require('sqlite3').verbose();

const dataDirectory = path.join(__dirname, '..', 'data');
fs.mkdirSync(dataDirectory, { recursive: true });

const database = new sqlite3.Database(path.join(dataDirectory, 'app.db'));

function run(sql, params = []) {
  return new Promise((resolve, reject) => {
    database.run(sql, params, function onRun(error) {
      if (error) return reject(error);
      return resolve({ id: this.lastID, changes: this.changes });
    });
  });
}

function all(sql, params = []) {
  return new Promise((resolve, reject) => {
    database.all(sql, params, (error, rows) => {
      if (error) return reject(error);
      return resolve(rows);
    });
  });
}

function get(sql, params = []) {
  return new Promise((resolve, reject) => {
    database.get(sql, params, (error, row) => {
      if (error) return reject(error);
      return resolve(row);
    });
  });
}

const ready = new Promise((resolve, reject) => {
  database.serialize(async () => {
    try {
      await run(`
        CREATE TABLE IF NOT EXISTS products (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          name TEXT NOT NULL,
          description TEXT NOT NULL,
          price REAL NOT NULL CHECK (price >= 0),
          stock INTEGER NOT NULL CHECK (stock >= 0),
          created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
          updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
        )
      `);

      const count = await get('SELECT COUNT(*) AS count FROM products');
      if (count.count === 0) {
        await run(
          `INSERT INTO products (name, description, price, stock) VALUES (?, ?, ?, ?), (?, ?, ?, ?), (?, ?, ?, ?)`,
          [
            'Keyboard Mechanical', 'Keyboard untuk testing endpoint CRUD', 750000, 10,
            'Mouse Wireless', 'Mouse wireless dengan koneksi USB', 250000, 25,
            'Monitor 24 Inch', 'Monitor IPS untuk kebutuhan kerja', 1850000, 5
          ]
        );
      }
      resolve();
    } catch (error) {
      reject(error);
    }
  });
});

async function getProducts() {
  await ready;
  return all('SELECT id, name, description, price, stock FROM products ORDER BY id');
}

async function getProductById(id) {
  await ready;
  return get('SELECT id, name, description, price, stock FROM products WHERE id = ?', [id]);
}

async function createProduct(product) {
  await ready;
  const result = await run(
    'INSERT INTO products (name, description, price, stock) VALUES (?, ?, ?, ?)',
    [product.name, product.description, product.price, product.stock]
  );
  return getProductById(result.id);
}

async function updateProduct(id, changes) {
  await ready;
  const current = await getProductById(id);
  const updated = {
    ...current,
    ...(changes.name !== undefined ? { name: changes.name.trim() } : {}),
    ...(changes.description !== undefined ? { description: changes.description } : {}),
    ...(changes.price !== undefined ? { price: changes.price } : {}),
    ...(changes.stock !== undefined ? { stock: changes.stock } : {})
  };

  await run(
    `UPDATE products
     SET name = ?, description = ?, price = ?, stock = ?, updated_at = CURRENT_TIMESTAMP
     WHERE id = ?`,
    [updated.name, updated.description, updated.price, updated.stock, id]
  );
  return updated;
}

async function deleteProduct(id) {
  await ready;
  return run('DELETE FROM products WHERE id = ?', [id]);
}

module.exports = {
  ready,
  getProducts,
  getProductById,
  createProduct,
  updateProduct,
  deleteProduct
};
