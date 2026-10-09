<?php

declare(strict_types=1);

loadEnv(__DIR__ . DIRECTORY_SEPARATOR . '.env');

$port = (int) env('PORT', '3000');
$jwtSecret = env('JWT_SECRET', 'development-only-secret');
$authUsername = env('AUTH_USERNAME', 'admin');
$authPassword = env('AUTH_PASSWORD', 'admin123');
$jwtExpiresIn = env('JWT_EXPIRES_IN', '1h');
$jwtExpirySeconds = parseDuration($jwtExpiresIn);

$dataDirectory = __DIR__ . DIRECTORY_SEPARATOR . 'data';
if (!is_dir($dataDirectory)) {
    mkdir($dataDirectory, 0755, true);
}

try {
    $db = new PDO('sqlite:' . $dataDirectory . DIRECTORY_SEPARATOR . 'app.db');
    $db->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);
    initializeDatabase($db);
} catch (Throwable $error) {
    http_response_code(500);
    header('Content-Type: application/json');
    echo json_encode(['message' => 'Database initialization failed']);
    exit;
}

$method = $_SERVER['REQUEST_METHOD'];
$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH) ?: '/';

if ($path === '/' && $method === 'GET') {
    home();
}
if ($path === '/health' && $method === 'GET') {
    jsonResponse(['status' => 'ok', 'service' => 'api-crud-test']);
}
if ($path === '/api-docs.json' && $method === 'GET') {
    jsonResponse(openApiSpec());
}
if (($path === '/api-docs' || $path === '/api-docs/') && $method === 'GET') {
    swaggerPage();
}
if ($path === '/api/auth/login' && $method === 'POST') {
    login($authUsername, $authPassword, $jwtSecret, $jwtExpiresIn, $jwtExpirySeconds);
}
if ($path === '/api/products' && $method === 'GET') {
    listProducts($db);
}
if ($path === '/api/products' && $method === 'POST') {
    requireAuth($jwtSecret);
    createProduct($db);
}

if (preg_match('#^/api/products/([0-9]+)$#', $path, $matches)) {
    $id = (int) $matches[1];
    if ($method === 'GET') {
        getProduct($db, $id);
    }
    if ($method === 'PUT') {
        requireAuth($jwtSecret);
        updateProduct($db, $id);
    }
    if ($method === 'DELETE') {
        requireAuth($jwtSecret);
        deleteProduct($db, $id);
    }
}

jsonError('Route not found', 404);

function initializeDatabase(PDO $db): void
{
    $db->exec(
        'CREATE TABLE IF NOT EXISTS products (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            description TEXT NOT NULL,
            price REAL NOT NULL CHECK (price >= 0),
            stock INTEGER NOT NULL CHECK (stock >= 0),
            created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
            updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
        )'
    );

    if ((int) $db->query('SELECT COUNT(*) FROM products')->fetchColumn() === 0) {
        $statement = $db->prepare(
            'INSERT INTO products (name, description, price, stock) VALUES (?, ?, ?, ?)'
        );
        $products = [
            ['Keyboard Mechanical', 'Keyboard untuk testing endpoint CRUD', 750000, 10],
            ['Mouse Wireless', 'Mouse wireless dengan koneksi USB', 250000, 25],
            ['Monitor 24 Inch', 'Monitor IPS untuk kebutuhan kerja', 1850000, 5],
        ];
        foreach ($products as $product) {
            $statement->execute($product);
        }
    }
}

function listProducts(PDO $db): never
{
    $products = $db->query(
        'SELECT id, name, description, price, stock FROM products ORDER BY id'
    )->fetchAll(PDO::FETCH_ASSOC);
    jsonResponse(['data' => $products, 'total' => count($products)]);
}

function getProduct(PDO $db, int $id): never
{
    $product = findProduct($db, $id);
    jsonResponse(['data' => $product]);
}

function createProduct(PDO $db): never
{
    $body = requestBody();
    $errors = validateProduct($body, false);
    if ($errors !== []) {
        jsonError('Invalid product data', 400, $errors);
    }

    $statement = $db->prepare(
        'INSERT INTO products (name, description, price, stock) VALUES (?, ?, ?, ?)'
    );
    $statement->execute([
        trim($body['name']),
        $body['description'],
        $body['price'],
        $body['stock'],
    ]);
    $product = findProduct($db, (int) $db->lastInsertId());
    jsonResponse(['message' => 'Product created successfully', 'data' => $product], 201);
}

function updateProduct(PDO $db, int $id): never
{
    $current = findProduct($db, $id);
    $body = requestBody();
    $errors = validateProduct($body, true);
    if ($errors !== []) {
        jsonError('Invalid product data', 400, $errors);
    }

    $updated = [
        'name' => array_key_exists('name', $body) ? trim($body['name']) : $current['name'],
        'description' => $body['description'] ?? $current['description'],
        'price' => $body['price'] ?? $current['price'],
        'stock' => $body['stock'] ?? $current['stock'],
    ];
    $statement = $db->prepare(
        'UPDATE products
         SET name = ?, description = ?, price = ?, stock = ?, updated_at = CURRENT_TIMESTAMP
         WHERE id = ?'
    );
    $statement->execute([
        $updated['name'],
        $updated['description'],
        $updated['price'],
        $updated['stock'],
        $id,
    ]);
    $updated['id'] = $id;
    jsonResponse(['message' => 'Product updated successfully', 'data' => $updated]);
}

function deleteProduct(PDO $db, int $id): never
{
    $product = findProduct($db, $id);
    $statement = $db->prepare('DELETE FROM products WHERE id = ?');
    $statement->execute([$id]);
    jsonResponse(['message' => 'Product deleted successfully', 'data' => $product]);
}

function findProduct(PDO $db, int $id): array
{
    $statement = $db->prepare(
        'SELECT id, name, description, price, stock FROM products WHERE id = ?'
    );
    $statement->execute([$id]);
    $product = $statement->fetch(PDO::FETCH_ASSOC);
    if ($product === false) {
        jsonError('Product not found', 404);
    }
    return $product;
}

function login(
    string $username,
    string $password,
    string $secret,
    string $expiresIn,
    int $expirySeconds
): never {
    $body = requestBody();
    if (($body['username'] ?? null) !== $username || ($body['password'] ?? null) !== $password) {
        jsonError('Username or password is incorrect', 401);
    }

    $now = time();
    $token = createJwt(
        ['username' => $username, 'role' => 'admin', 'iat' => $now, 'exp' => $now + $expirySeconds],
        $secret
    );
    jsonResponse([
        'message' => 'Login successful',
        'tokenType' => 'Bearer',
        'expiresIn' => $expiresIn,
        'token' => $token,
    ]);
}

function requireAuth(string $secret): void
{
    $header = $_SERVER['HTTP_AUTHORIZATION'] ?? '';
    if (!preg_match('/^Bearer\s+(.+)$/i', $header, $matches)) {
        jsonError('Bearer token is required', 401);
    }

    $payload = verifyJwt($matches[1], $secret);
    if ($payload === null) {
        jsonError('Token is invalid or expired', 401);
    }
}

function validateProduct(array $body, bool $partial): array
{
    $errors = [];
    $fields = ['name', 'description', 'price', 'stock'];
    foreach ($fields as $field) {
        if (!$partial && !array_key_exists($field, $body)) {
            $errors[] = "{$field} is required";
        }
    }
    if (array_key_exists('name', $body) && (!is_string($body['name']) || trim($body['name']) === '')) {
        $errors[] = 'name must not be empty';
    }
    if (array_key_exists('description', $body) && !is_string($body['description'])) {
        $errors[] = 'description must be a string';
    }
    if (array_key_exists('price', $body) && (!is_numeric($body['price']) || $body['price'] < 0)) {
        $errors[] = 'price must be >= 0';
    }
    if (array_key_exists('stock', $body) && (!is_int($body['stock']) || $body['stock'] < 0)) {
        $errors[] = 'stock must be an integer >= 0';
    }
    return $errors;
}

function requestBody(): array
{
    $raw = file_get_contents('php://input');
    $body = json_decode($raw ?: '', true);
    if (!is_array($body) || array_is_list($body)) {
        jsonError('Request body must be a JSON object', 400);
    }
    return $body;
}

function createJwt(array $payload, string $secret): string
{
    $header = base64UrlEncode(json_encode(['alg' => 'HS256', 'typ' => 'JWT']));
    $body = base64UrlEncode(json_encode($payload));
    $signature = base64UrlEncode(hash_hmac('sha256', "{$header}.{$body}", $secret, true));
    return "{$header}.{$body}.{$signature}";
}

function verifyJwt(string $token, string $secret): ?array
{
    $parts = explode('.', $token);
    if (count($parts) !== 3) {
        return null;
    }
    [$header, $body, $signature] = $parts;
    $expected = base64UrlEncode(hash_hmac('sha256', "{$header}.{$body}", $secret, true));
    if (!hash_equals($expected, $signature)) {
        return null;
    }
    $payload = json_decode(base64UrlDecode($body), true);
    if (!is_array($payload) || !isset($payload['exp']) || $payload['exp'] < time()) {
        return null;
    }
    return $payload;
}

function base64UrlEncode(string $value): string
{
    return rtrim(strtr(base64_encode($value), '+/', '-_'), '=');
}

function base64UrlDecode(string $value): string
{
    return base64_decode(strtr($value, '-_', '+/') . str_repeat('=', (4 - strlen($value) % 4) % 4));
}

function jsonResponse(array $data, int $status = 200): never
{
    http_response_code($status);
    header('Content-Type: application/json; charset=utf-8');
    echo json_encode($data, JSON_UNESCAPED_SLASHES);
    exit;
}

function jsonError(string $message, int $status, array $details = []): never
{
    $response = ['message' => $message];
    if ($details !== []) {
        $response['details'] = $details;
    }
    jsonResponse($response, $status);
}

function home(): never
{
    header('Content-Type: text/html; charset=utf-8');
    echo '<!doctype html><html lang="en"><head><meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>API CRUD Test</title><style>
    body{font-family:Arial,sans-serif;max-width:720px;margin:48px auto;padding:0 20px;color:#222}
    a{color:#0969da}.card{background:#f6f8fa;border-radius:8px;padding:16px 20px;margin-top:24px}
    code{background:#eaeef2;padding:2px 5px;border-radius:4px}</style></head><body>
    <h1>API CRUD Test</h1><p>A simple PHP API for testing public and authenticated CRUD endpoints.</p>
    <div class="card"><strong>Available resources</strong>
    <p>Products CRUD with SQLite storage and JWT authentication.</p>
    <p><a href="/api-docs">Open API Documentation</a></p>
    <p><a href="/api/products">View Public Products Endpoint</a></p>
    <p>Health check: <code>GET /health</code></p></div></body></html>';
    exit;
}

function swaggerPage(): never
{
    header('Content-Type: text/html; charset=utf-8');
    echo '<!doctype html><html><head><title>API CRUD Test - Swagger</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"></head>
    <body><div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>window.onload=()=>SwaggerUIBundle({url:"/api-docs.json",dom_id:"#swagger-ui"});</script>
    </body></html>';
    exit;
}

function openApiSpec(): array
{
    return [
        'openapi' => '3.0.3',
        'info' => [
            'title' => 'API CRUD Test',
            'version' => '1.0.0',
            'description' => 'PHP API CRUD with SQLite and JWT authentication.',
        ],
        'servers' => [['url' => 'http://localhost:3000']],
        'components' => [
            'securitySchemes' => [
                'bearerAuth' => ['type' => 'http', 'scheme' => 'bearer', 'bearerFormat' => 'JWT'],
            ],
            'schemas' => [
                'Product' => ['type' => 'object', 'properties' => [
                    'id' => ['type' => 'integer'], 'name' => ['type' => 'string'],
                    'description' => ['type' => 'string'], 'price' => ['type' => 'number'],
                    'stock' => ['type' => 'integer'],
                ]],
                'ProductInput' => ['type' => 'object', 'required' => ['name', 'description', 'price', 'stock'],
                    'properties' => [
                        'name' => ['type' => 'string', 'example' => 'Webcam'],
                        'description' => ['type' => 'string', 'example' => 'Webcam for video calls'],
                        'price' => ['type' => 'number', 'minimum' => 0, 'example' => 450000],
                        'stock' => ['type' => 'integer', 'minimum' => 0, 'example' => 8],
                    ]],
            ],
        ],
        'paths' => [
            '/health' => ['get' => ['summary' => 'Check API status', 'responses' => ['200' => ['description' => 'API is running']]]],
            '/api/auth/login' => ['post' => ['summary' => 'Login and get JWT', 'requestBody' => ['required' => true,
                'content' => ['application/json' => ['schema' => ['type' => 'object', 'required' => ['username', 'password'],
                    'properties' => ['username' => ['type' => 'string', 'example' => 'admin'],
                        'password' => ['type' => 'string', 'example' => 'admin123']]]]]],
                'responses' => ['200' => ['description' => 'Login successful'], '401' => ['description' => 'Invalid credentials']]]],
            '/api/products' => [
                'get' => ['summary' => 'List products (public)', 'responses' => ['200' => ['description' => 'Product list']]],
                'post' => ['summary' => 'Create product (authenticated)', 'security' => [['bearerAuth' => []]],
                    'requestBody' => ['required' => true, 'content' => ['application/json' => ['schema' => ['$ref' => '#/components/schemas/ProductInput']]]],
                    'responses' => ['201' => ['description' => 'Product created'], '401' => ['description' => 'Authentication required']]],
            ],
            '/api/products/{id}' => [
                'parameters' => [['name' => 'id', 'in' => 'path', 'required' => true, 'schema' => ['type' => 'integer']]],
                'get' => ['summary' => 'Get product (public)', 'responses' => ['200' => ['description' => 'Product detail'], '404' => ['description' => 'Not found']]],
                'put' => ['summary' => 'Update product (authenticated)', 'security' => [['bearerAuth' => []]],
                    'requestBody' => ['required' => true, 'content' => ['application/json' => ['schema' => ['$ref' => '#/components/schemas/ProductInput']]]],
                    'responses' => ['200' => ['description' => 'Product updated'], '401' => ['description' => 'Authentication required']]],
                'delete' => ['summary' => 'Delete product (authenticated)', 'security' => [['bearerAuth' => []]],
                    'responses' => ['200' => ['description' => 'Product deleted'], '401' => ['description' => 'Authentication required']]],
            ],
        ],
    ];
}

function loadEnv(string $filename): void
{
    if (!is_file($filename)) {
        return;
    }
    foreach (file($filename, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES) as $line) {
        $line = trim($line);
        if ($line === '' || str_starts_with($line, '#') || !str_contains($line, '=')) {
            continue;
        }
        [$key, $value] = explode('=', $line, 2);
        if (getenv(trim($key)) === false) {
            putenv(trim($key) . '=' . trim($value, " \t\n\r\0\x0B\"'"));
        }
    }
}

function env(string $key, string $fallback): string
{
    $value = getenv($key);
    return $value === false || $value === '' ? $fallback : $value;
}

function parseDuration(string $duration): int
{
    if (preg_match('/^(\d+)([smhd])$/', $duration, $matches) !== 1) {
        return 3600;
    }
    $multipliers = ['s' => 1, 'm' => 60, 'h' => 3600, 'd' => 86400];
    return (int) $matches[1] * $multipliers[$matches[2]];
}
