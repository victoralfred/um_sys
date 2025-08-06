# UManager API Postman Tests

This directory contains comprehensive Postman tests for the UManager User Management System API.

## Files

- **`UManager_API_Collection.postman_collection.json`** - Main Postman collection with all API tests
- **`UManager_Local_Environment.postman_environment.json`** - Environment variables for local testing
- **`run-tests.sh`** - Shell script to run tests using Newman CLI
- **`README.md`** - This documentation file

## Test Coverage

The test collection covers:

### 1. Health & Info Tests
- Health check endpoint
- API information endpoint

### 2. Authentication Tests
- User registration (success and failure cases)
- Login with email
- Login with username
- Token refresh
- Password validation
- Duplicate email prevention
- Invalid credentials handling

### 3. Protected Endpoints Tests
- Get current user profile
- Logout functionality
- Authorization header validation
- Token validation

### 4. Load Testing
- Concurrent registrations
- Rate limiting verification

## Setup

### Prerequisites

1. **Node.js and npm** installed on your system
2. **Postman** (optional - for GUI testing)
3. **Newman** (for command-line testing)

### Install Newman

```bash
npm install -g newman
npm install -g newman-reporter-htmlextra
```

## Usage

### Option 1: Using Postman GUI

1. Open Postman
2. Import the collection: `UManager_API_Collection.postman_collection.json`
3. Import the environment: `UManager_Local_Environment.postman_environment.json`
4. Select the "UManager Local Environment" from the environment dropdown
5. Run individual requests or the entire collection

### Option 2: Using Newman CLI

#### Interactive Mode
```bash
chmod +x run-tests.sh
./run-tests.sh
```

#### Run All Tests
```bash
./run-tests.sh --all
```

#### Run Specific Test Suites
```bash
# Authentication tests only
./run-tests.sh --auth

# Health check tests only
./run-tests.sh --health

# Protected endpoints tests only
./run-tests.sh --protected

# Load tests
./run-tests.sh --load
```

#### Custom Configuration
```bash
# Run against different server
./run-tests.sh --all --url http://staging.example.com:8080

# Run with multiple iterations
./run-tests.sh --all --iterations 5

# Run with delay between requests (milliseconds)
./run-tests.sh --all --delay 500

# Combine options
./run-tests.sh --auth --url http://localhost:3000 --iterations 3 --delay 100
```

### Option 3: Direct Newman Command

```bash
# Basic run
newman run UManager_API_Collection.postman_collection.json \
  -e UManager_Local_Environment.postman_environment.json

# With HTML report
newman run UManager_API_Collection.postman_collection.json \
  -e UManager_Local_Environment.postman_environment.json \
  --reporters cli,htmlextra \
  --reporter-htmlextra-export test-results.html

# Run specific folder
newman run UManager_API_Collection.postman_collection.json \
  -e UManager_Local_Environment.postman_environment.json \
  --folder "Authentication"

# With custom variables
newman run UManager_API_Collection.postman_collection.json \
  -e UManager_Local_Environment.postman_environment.json \
  --global-var "baseUrl=http://localhost:3000"
```

## Test Structure

Each test includes:

1. **Pre-request Scripts**: Set up test data, generate random values
2. **Request Configuration**: Headers, body, parameters
3. **Test Scripts**: Assertions for:
   - Status codes
   - Response structure
   - Data validation
   - Error handling
   - Performance (response time)

## Environment Variables

The tests use the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `baseUrl` | API base URL | `http://localhost:8080` |
| `apiVersion` | API version | `v1` |
| `accessToken` | JWT access token (auto-populated) | - |
| `refreshToken` | JWT refresh token (auto-populated) | - |
| `userId` | User ID (auto-populated) | - |
| `testEmail` | Test user email | `test.user@example.com` |
| `testUsername` | Test username | `testuser` |
| `testPassword` | Test password | `TestPassword123` |

## Test Assertions

Each endpoint test includes multiple assertions:

### Success Cases
- Correct HTTP status codes (200, 201)
- Response structure validation
- Data integrity checks
- Token generation and validation
- User data consistency

### Error Cases
- Proper error status codes (400, 401, 403, 404, 409)
- Error message structure
- Descriptive error codes
- No sensitive data in errors

### Performance
- Response time thresholds
- Rate limiting verification
- Concurrent request handling

## Continuous Integration

To integrate with CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
- name: Run API Tests
  run: |
    npm install -g newman newman-reporter-htmlextra
    ./tests/postman/run-tests.sh --all
```

```groovy
// Example Jenkins pipeline
stage('API Tests') {
    steps {
        sh 'npm install -g newman newman-reporter-htmlextra'
        sh './tests/postman/run-tests.sh --all'
        publishHTML([
            reportName: 'API Test Results',
            reportDir: 'tests/postman',
            reportFiles: 'test-results*.html'
        ])
    }
}
```

## Troubleshooting

### Common Issues

1. **Connection refused**: Ensure the API server is running on the specified port
2. **Token expired**: Tests automatically refresh tokens, but check token expiry settings
3. **Rate limiting**: Adjust delay between requests using `--delay` option
4. **Database conflicts**: Tests use random data to avoid conflicts

### Debug Mode

Run with verbose output:
```bash
newman run UManager_API_Collection.postman_collection.json \
  -e UManager_Local_Environment.postman_environment.json \
  --verbose
```

## Contributing

When adding new endpoints:

1. Add test cases to appropriate folder in the collection
2. Include both success and failure scenarios
3. Add proper assertions
4. Update this README with new test coverage
5. Test locally before committing

## License

Part of the UManager User Management System project.