# User Management System - Base Features Documentation

## Table of Contents
1. [System Overview](#system-overview)
2. [Architecture](#architecture)
3. [Core Features](#core-features)
4. [Technical Implementation](#technical-implementation)
5. [API Reference](#api-reference)
6. [Security](#security)
7. [Testing](#testing)
8. [Deployment](#deployment)

## System Overview

### Project Information
- **Name**: User Management System (UMS)
- **Version**: 1.0.0
- **Repository**: https://github.com/victoralfred/um_sys
- **Language**: Go 1.21+
- **Database**: PostgreSQL 16
- **Architecture**: Clean Architecture with Domain-Driven Design

### Key Capabilities
- Enterprise-grade user authentication and authorization
- Multi-factor authentication support
- Subscription and billing management
- Comprehensive audit logging
- Feature flag system for controlled rollouts
- Role-based access control (RBAC)
- Production-ready with full test coverage

## Architecture

### Design Principles
- **Clean Architecture**: Separation of concerns with clear boundaries
- **Domain-Driven Design**: Business logic centered around domain models
- **Hexagonal Architecture**: Ports and adapters for external dependencies
- **Test-Driven Development**: Tests written before implementation
- **SOLID Principles**: Single responsibility, open/closed, interface segregation

### Project Structure
```
umanager/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── domain/                  # Core business logic
│   │   ├── user/                # User management domain
│   │   │   ├── models.go        # User entities
│   │   │   ├── interfaces.go    # Repository interfaces
│   │   │   └── errors.go        # Domain-specific errors
│   │   ├── auth/                # Authentication domain
│   │   │   ├── models.go        # JWT tokens, claims
│   │   │   ├── interfaces.go    # Auth service interfaces
│   │   │   └── errors.go        # Auth errors
│   │   ├── rbac/                # Role-based access control
│   │   │   ├── models.go        # Roles, permissions
│   │   │   ├── interfaces.go    # RBAC interfaces
│   │   │   └── errors.go        # RBAC errors
│   │   ├── mfa/                 # Multi-factor authentication
│   │   │   ├── models.go        # MFA settings, challenges
│   │   │   ├── interfaces.go    # MFA interfaces
│   │   │   └── errors.go        # MFA errors
│   │   ├── billing/             # Subscription & billing
│   │   │   ├── models.go        # Plans, subscriptions, payments
│   │   │   ├── interfaces.go    # Billing interfaces
│   │   │   └── errors.go        # Billing errors
│   │   ├── audit/               # Audit logging
│   │   │   ├── models.go        # Log entries, alerts
│   │   │   ├── interfaces.go    # Audit interfaces
│   │   │   └── errors.go        # Audit errors
│   │   └── feature/             # Feature flags
│   │       ├── models.go        # Flags, rules, segments
│   │       ├── interfaces.go    # Feature flag interfaces
│   │       └── errors.go        # Feature flag errors
│   ├── services/                # Business logic implementation
│   │   ├── auth_service.go      # Authentication service
│   │   ├── rbac_service.go      # RBAC service
│   │   ├── mfa_service.go       # MFA service
│   │   ├── billing_service.go   # Billing service
│   │   ├── audit_service.go     # Audit service
│   │   └── feature_service.go   # Feature flag service
│   ├── adapters/                # External adapters
│   │   └── database/            # Database implementation
│   │       ├── postgres.go      # PostgreSQL connection
│   │       └── user_repository.go # User repository impl
│   └── config/                  # Configuration
│       └── config.go            # App configuration
├── migrations/                  # Database migrations
│   ├── 001_create_users_table.up.sql
│   ├── 001_create_users_table.down.sql
│   └── ...
├── pkg/                         # Shared packages
│   └── logger/                  # Logging utilities
├── tests/                       # Test files
│   └── *_test.go               # Test implementations
└── docs/                        # Documentation

```

## Core Features

### 1. User Management

#### User Model
```go
type User struct {
    ID                uuid.UUID
    Email             string
    Username          string
    FirstName         string
    LastName          string
    PasswordHash      string
    EmailVerified     bool
    PhoneNumber       string
    PhoneVerified     bool
    TwoFactorEnabled  bool
    Status            Status
    FailedLoginAttempts int
    LastLoginAt       *time.Time
    LastLoginIP       string
    PasswordChangedAt time.Time
    Metadata          map[string]interface{}
    CreatedAt         time.Time
    UpdatedAt         time.Time
    DeletedAt         *time.Time
}
```

#### Features
- **Registration**: Email/username with validation
- **Profile Management**: Update user information
- **Email Verification**: Token-based verification
- **Password Management**: Reset, change, complexity rules
- **Account Status**: Active, Inactive, Suspended, Locked
- **Soft Deletion**: Preserve data integrity
- **Metadata Storage**: Flexible key-value storage

### 2. Authentication System (JWT)

#### Token Types
- **Access Token**: Short-lived (15 minutes), for API access
- **Refresh Token**: Long-lived (7 days), for token renewal
- **Reset Token**: Single-use, for password reset

#### Implementation Details
```go
type TokenPair struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
}

type Claims struct {
    UserID   uuid.UUID
    Email    string
    Username string
    Roles    []string
    jwt.RegisteredClaims
}
```

#### Security Features
- RS256 signing algorithm
- Token blacklisting support
- Refresh token rotation
- IP address validation
- Device fingerprinting ready
- Rate limiting on login attempts

### 3. Role-Based Access Control (RBAC)

#### System Roles
| Role | Description | Default Permissions |
|------|-------------|-------------------|
| SuperAdmin | System administrator | All permissions |
| Admin | Organization admin | Manage users, roles |
| Moderator | Content moderator | Manage content |
| User | Regular user | Basic permissions |
| Guest | Unauthenticated | Read-only access |

#### Permission Format
```
resource:action
```
Examples:
- `users:create` - Create users
- `users:read` - View users
- `users:update` - Update users
- `users:delete` - Delete users
- `roles:assign` - Assign roles
- `billing:manage` - Manage billing

#### Policy Engine
```go
type PolicyRule struct {
    ID          uuid.UUID
    Resource    string
    Action      string
    Effect      Effect // Allow or Deny
    Conditions  []Condition
    Priority    int
}
```

### 4. Multi-Factor Authentication (MFA)

#### Supported Methods
1. **TOTP (Time-based One-Time Password)**
   - 30-second window
   - 6-digit codes
   - QR code generation
   - Compatible with Google Authenticator, Authy

2. **SMS Verification**
   - 6-digit codes
   - 5-minute expiry
   - Rate limited

3. **Email Verification**
   - 6-digit codes
   - 10-minute expiry
   - HTML email support

4. **Backup Codes**
   - 10 single-use codes
   - Alphanumeric format
   - Secure storage

#### MFA Flow
```go
type MFAChallenge struct {
    ID         uuid.UUID
    UserID     uuid.UUID
    Method     MFAMethod
    Code       string
    ExpiresAt  time.Time
    Attempts   int
    VerifiedAt *time.Time
}
```

### 5. Subscription & Billing Management

#### Plan Structure
```go
type Plan struct {
    ID              uuid.UUID
    Name            string
    Type            PlanType // Free, Basic, Pro, Enterprise
    Price           decimal.Decimal
    Currency        string
    BillingInterval BillingInterval // Monthly, Yearly
    TrialDays       int
    Features        []Feature
    Limits          PlanLimits
}

type PlanLimits struct {
    MaxUsers            int
    MaxProjects         int
    MaxAPICallsPerMonth int
    MaxStorageGB        int
    MaxTeamMembers      int
}
```

#### Subscription Lifecycle
- **States**: Active, Trialing, Past Due, Canceled, Paused
- **Operations**: Create, Upgrade, Downgrade, Cancel, Resume
- **Trial Support**: Configurable trial periods
- **Grace Period**: Past due handling

#### Payment Processing
```go
type Payment struct {
    ID             uuid.UUID
    UserID         uuid.UUID
    SubscriptionID *uuid.UUID
    Amount         decimal.Decimal
    Currency       string
    Status         PaymentStatus
    PaymentMethod  string
    StripePaymentID string
    RefundedAmount decimal.Decimal
}
```

#### Revenue Analytics
- Monthly Recurring Revenue (MRR)
- Annual Recurring Revenue (ARR)
- Churn rate calculation
- Customer Lifetime Value (CLV)

### 6. Audit Logging System

#### Event Types
```go
type EventType string

const (
    // User events
    EventTypeUserCreated
    EventTypeUserUpdated
    EventTypeUserDeleted
    EventTypeUserLoggedIn
    EventTypeUserLoggedOut
    
    // Security events
    EventTypeSecurityAlert
    EventTypeAccessDenied
    EventTypeRateLimited
    
    // Business events
    EventTypeSubscriptionCreated
    EventTypePaymentProcessed
)
```

#### Log Entry Structure
```go
type LogEntry struct {
    ID          uuid.UUID
    Timestamp   time.Time
    EventType   EventType
    Severity    Severity // Info, Warning, Error, Critical
    UserID      *uuid.UUID
    ActorID     *uuid.UUID
    EntityType  string
    EntityID    string
    Action      string
    Description string
    IPAddress   string
    UserAgent   string
    Metadata    map[string]interface{}
    Changes     *Changes // Before/After for updates
}
```

#### Alert Rules
```go
type AlertRule struct {
    ID         uuid.UUID
    Name       string
    EventTypes []EventType
    Severities []Severity
    Threshold  int
    TimeWindow time.Duration
    Actions    []AlertAction
}
```

### 7. Feature Flag System

#### Flag Types
- **Boolean**: On/off toggles
- **String**: Text values
- **Number**: Numeric values
- **Percentage**: Gradual rollouts
- **JSON**: Complex configurations

#### Targeting Options
```go
type Rule struct {
    ID         uuid.UUID
    Priority   int
    Conditions []Condition
    Value      interface{}
    Percentage int
}

type Condition struct {
    Type     string // user, group, property
    Property string
    Operator string // equals, contains, greater_than
    Value    interface{}
}
```

#### Evaluation Context
```go
type Context struct {
    UserID      *uuid.UUID
    GroupID     *uuid.UUID
    Environment string
    IPAddress   string
    Properties  map[string]interface{}
}
```

#### A/B Testing Support
```go
type Experiment struct {
    ID       uuid.UUID
    FlagID   uuid.UUID
    Variants []Variant
    Metrics  []string
}

type Variant struct {
    Key    string
    Value  interface{}
    Weight int // Percentage
}
```

## Technical Implementation

### Database Schema

#### Users Table
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    email_verified BOOLEAN DEFAULT FALSE,
    phone_number VARCHAR(20),
    phone_verified BOOLEAN DEFAULT FALSE,
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    status VARCHAR(20) DEFAULT 'active',
    failed_login_attempts INT DEFAULT 0,
    last_login_at TIMESTAMP,
    last_login_ip INET,
    password_changed_at TIMESTAMP,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

#### Roles & Permissions
```sql
CREATE TABLE roles (
    id UUID PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT FALSE,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE permissions (
    id UUID PRIMARY KEY,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL,
    description TEXT,
    UNIQUE(resource, action)
);

CREATE TABLE role_permissions (
    role_id UUID REFERENCES roles(id),
    permission_id UUID REFERENCES permissions(id),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id UUID REFERENCES users(id),
    role_id UUID REFERENCES roles(id),
    assigned_at TIMESTAMP DEFAULT NOW(),
    assigned_by UUID,
    PRIMARY KEY (user_id, role_id)
);
```

### API Endpoints

#### Authentication
```
POST   /api/v1/auth/register     - User registration
POST   /api/v1/auth/login        - User login
POST   /api/v1/auth/logout       - User logout
POST   /api/v1/auth/refresh      - Refresh token
POST   /api/v1/auth/verify-email - Verify email
POST   /api/v1/auth/forgot-password - Request password reset
POST   /api/v1/auth/reset-password  - Reset password
```

#### User Management
```
GET    /api/v1/users            - List users
GET    /api/v1/users/:id        - Get user
PUT    /api/v1/users/:id        - Update user
DELETE /api/v1/users/:id        - Delete user
GET    /api/v1/users/me         - Get current user
PUT    /api/v1/users/me         - Update current user
```

#### RBAC
```
GET    /api/v1/roles            - List roles
POST   /api/v1/roles            - Create role
PUT    /api/v1/roles/:id        - Update role
DELETE /api/v1/roles/:id        - Delete role
POST   /api/v1/users/:id/roles  - Assign role
DELETE /api/v1/users/:id/roles/:roleId - Remove role
```

#### MFA
```
POST   /api/v1/mfa/setup        - Setup MFA
POST   /api/v1/mfa/verify       - Verify MFA code
POST   /api/v1/mfa/disable      - Disable MFA
GET    /api/v1/mfa/backup-codes - Get backup codes
POST   /api/v1/mfa/backup-codes/regenerate - Regenerate codes
```

#### Billing
```
GET    /api/v1/plans            - List plans
GET    /api/v1/subscriptions    - Get user subscriptions
POST   /api/v1/subscriptions    - Create subscription
PUT    /api/v1/subscriptions/:id - Update subscription
DELETE /api/v1/subscriptions/:id - Cancel subscription
GET    /api/v1/payments         - List payments
POST   /api/v1/payments         - Process payment
```

#### Feature Flags
```
GET    /api/v1/flags            - List flags
GET    /api/v1/flags/:key       - Get flag
POST   /api/v1/flags/:key/evaluate - Evaluate flag
GET    /api/v1/flags/state      - Get all flags state
```

## Security

### Password Security
- **Hashing**: bcrypt with cost factor 10
- **Complexity Requirements**:
  - Minimum 8 characters
  - At least one uppercase letter
  - At least one lowercase letter
  - At least one number
  - At least one special character
- **History**: Prevent reuse of last 5 passwords
- **Expiry**: Configurable password expiry

### Session Security
- **Token Security**:
  - RS256 signing
  - Short-lived access tokens
  - Secure refresh token storage
  - Token revocation support
- **Session Management**:
  - Concurrent session limits
  - Session timeout
  - IP binding option
  - Device tracking

### Rate Limiting
```go
type RateLimitConfig struct {
    LoginAttempts    int           // 5 attempts
    LoginWindow      time.Duration // 15 minutes
    APICallsPerMin   int           // 100 calls
    PasswordResets   int           // 3 per hour
}
```

### Data Protection
- **Encryption at Rest**: Database encryption
- **Encryption in Transit**: TLS 1.3
- **PII Handling**: GDPR compliance
- **Data Retention**: Configurable policies
- **Audit Trail**: All data access logged

## Testing

### Test Coverage
- **Unit Tests**: 85%+ coverage
- **Integration Tests**: Database operations
- **E2E Tests**: Critical user flows
- **Performance Tests**: Load testing ready

### Testing Strategy
```go
// Example test structure
func TestAuthService_Login(t *testing.T) {
    t.Run("successful login", func(t *testing.T) {
        // Arrange
        mockRepo := new(MockUserRepository)
        service := NewAuthService(mockRepo)
        
        // Act
        result, err := service.Login(ctx, request)
        
        // Assert
        assert.NoError(t, err)
        assert.NotNil(t, result)
    })
}
```

### Test Data
- Fixtures for consistent test data
- Factories for dynamic test data
- Seed data for development

## Deployment

### Environment Variables
```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=secret
DB_NAME=umanager

# JWT
JWT_PRIVATE_KEY_PATH=/keys/private.pem
JWT_PUBLIC_KEY_PATH=/keys/public.pem

# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Feature Flags
FEATURE_FLAGS_ENABLED=true

# Audit
AUDIT_RETENTION_DAYS=90
```

### Docker Support
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
CMD ["./server"]
```

### Database Migrations
```bash
# Run migrations up
go run cmd/server/main.go migrate up

# Rollback migrations
go run cmd/server/main.go migrate down 1

# Create new migration
go run cmd/server/main.go migrate create add_column_to_users
```

### Health Checks
```go
GET /health        - Basic health check
GET /health/ready  - Readiness probe
GET /health/live   - Liveness probe
```

### Monitoring
- **Metrics**: Prometheus-compatible
- **Logging**: Structured JSON logs
- **Tracing**: OpenTelemetry support
- **Error Tracking**: Sentry integration ready

## Performance Optimizations

### Database
- Connection pooling (10-50 connections)
- Prepared statements
- Query optimization
- Index strategy
- Read replicas support

### Caching Strategy
- In-memory caching for hot data
- Redis support ready
- Cache invalidation patterns
- TTL configuration

### API Optimizations
- Response compression
- Pagination support
- Field filtering
- Batch operations
- GraphQL ready

## Scalability Considerations

### Horizontal Scaling
- Stateless design
- Load balancer ready
- Database connection pooling
- Distributed caching support

### Microservices Ready
- Service interfaces defined
- Event-driven architecture ready
- Message queue support
- Service mesh compatible

## Compliance & Standards

### GDPR Compliance
- Right to be forgotten
- Data portability
- Consent management
- Privacy by design

### Security Standards
- OWASP Top 10 mitigation
- PCI DSS considerations
- SOC 2 compliance ready
- ISO 27001 alignment

### Audit Requirements
- Complete audit trail
- Data access logging
- Change tracking
- Compliance reporting

## Development Guidelines

### Code Style
- Go fmt enforcement
- Linting with golangci-lint
- Code review requirements
- Documentation standards

### Git Workflow
```bash
# Feature branch
git checkout -b feature/new-feature

# Commit with conventional commits
git commit -m "feat: add new feature"

# Push and create PR
git push origin feature/new-feature
```

### Commit Convention
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation
- `test:` Testing
- `refactor:` Code refactoring
- `chore:` Maintenance

## API Documentation

### Request/Response Format
```json
// Success Response
{
    "success": true,
    "data": {},
    "message": "Operation successful"
}

// Error Response
{
    "success": false,
    "error": {
        "code": "AUTH001",
        "message": "Invalid credentials",
        "details": {}
    }
}
```

### Error Codes
| Code | Description |
|------|-------------|
| AUTH001 | Invalid credentials |
| AUTH002 | Token expired |
| AUTH003 | Unauthorized access |
| USER001 | User not found |
| USER002 | Email already exists |
| RBAC001 | Permission denied |
| MFA001 | Invalid MFA code |
| BILL001 | Payment failed |

## Support & Maintenance

### Logging Levels
- **DEBUG**: Detailed debugging information
- **INFO**: General information
- **WARN**: Warning messages
- **ERROR**: Error messages
- **FATAL**: Critical errors

### Troubleshooting
- Check logs in `/var/log/umanager/`
- Database connection issues
- Token validation errors
- Permission denied errors
- Rate limiting issues

### Backup Strategy
- Daily database backups
- Point-in-time recovery
- Backup retention policy
- Disaster recovery plan

## Future Roadmap

### Planned Features
- [ ] OAuth 2.0 / Social login
- [ ] WebAuthn support
- [ ] API Gateway integration
- [ ] GraphQL API
- [ ] Webhook system
- [ ] Real-time notifications
- [ ] Mobile SDK
- [ ] Admin dashboard

### Performance Goals
- Sub-100ms API response time
- 99.9% uptime SLA
- Support 10,000+ concurrent users
- Handle 1M+ requests/day

---

## Quick Start Guide

### Installation
```bash
# Clone repository
git clone https://github.com/victoralfred/um_sys.git
cd umanager

# Install dependencies
go mod download

# Setup database
createdb umanager
go run cmd/server/main.go migrate up

# Run tests
go test ./...

# Start server
go run cmd/server/main.go
```

### First Steps
1. Register a user
2. Verify email
3. Login and get tokens
4. Setup MFA (optional)
5. Explore API endpoints

### Configuration
Create `.env` file:
```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=umanager
JWT_SECRET=your_secret_key
```

---

**Version**: 1.0.0  
**Last Updated**: August 2024  
**Author**: Victor Alfred  
**License**: MIT  
**Repository**: https://github.com/victoralfred/um_sys