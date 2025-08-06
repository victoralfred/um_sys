# Comprehensive API Design Plan
## Modern, Secure, and User-Friendly REST API Architecture

---

## Table of Contents
1. [API Design Principles](#api-design-principles)
2. [Authentication & Session Management](#1-authentication--session-management)
3. [User Management & Profiles](#2-user-management--profiles)
4. [Role-Based Access Control](#3-role-based-access-control-rbac)
5. [Multi-Factor Authentication](#4-multi-factor-authentication-mfa)
6. [Subscription & Billing](#5-subscription--billing-management)
7. [Audit & Compliance](#6-audit--compliance)
8. [Feature Flags](#7-feature-flags--ab-testing)
9. [Admin Dashboard APIs](#8-admin-dashboard-apis)
10. [Real-time & Webhooks](#9-real-time--webhooks)
11. [Public APIs](#10-public-apis)
12. [API Standards](#api-standards--conventions)

---

## API Design Principles

### Core Principles
- **RESTful Design**: Follow REST conventions with proper HTTP methods
- **Consistency**: Uniform patterns across all endpoints
- **Security First**: Authentication, authorization, rate limiting
- **User Experience**: Intuitive, predictable, well-documented
- **Performance**: Efficient queries, caching, pagination
- **Versioning**: Future-proof with version prefixes
- **Error Handling**: Clear, actionable error messages

### Base URL Structure
```
Production: https://api.yourdomain.com/v1
Staging:    https://api-staging.yourdomain.com/v1
Development: http://localhost:8080/v1
```

### Standard Headers
```http
Content-Type: application/json
Accept: application/json
Authorization: Bearer <jwt-token>
X-Request-ID: <unique-request-id>
X-Device-ID: <device-fingerprint>
X-API-Version: 1.0
```

### Standard Response Format
```json
{
  "success": true,
  "data": { ... },
  "meta": {
    "timestamp": "2024-01-01T00:00:00Z",
    "request_id": "req_abc123",
    "version": "1.0"
  },
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

### Error Response Format
```json
{
  "success": false,
  "error": {
    "code": "AUTH_INVALID_CREDENTIALS",
    "message": "Invalid email or password",
    "details": {
      "field": "email",
      "reason": "User not found"
    },
    "request_id": "req_abc123",
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

---

## 1. Authentication & Session Management

### Public Authentication Endpoints

#### User Registration
```http
POST /v1/auth/register
```
```json
Request:
{
  "email": "user@example.com",
  "username": "johndoe",
  "password": "SecureP@ssw0rd123!",
  "first_name": "John",
  "last_name": "Doe",
  "phone": "+1234567890",
  "accept_terms": true,
  "marketing_consent": false,
  "referral_code": "FRIEND123"
}

Response: 201 Created
{
  "success": true,
  "data": {
    "user": {
      "id": "usr_abc123",
      "email": "user@example.com",
      "username": "johndoe",
      "email_verified": false,
      "created_at": "2024-01-01T00:00:00Z"
    },
    "tokens": {
      "access_token": "eyJhbGc...",
      "refresh_token": "eyJhbGc...",
      "expires_in": 900,
      "token_type": "Bearer"
    },
    "requires_verification": true,
    "verification_sent_to": "user@example.com"
  }
}
```

#### User Login
```http
POST /v1/auth/login
```
```json
Request:
{
  "login": "user@example.com",  // Email or username
  "password": "SecureP@ssw0rd123!",
  "device_info": {
    "device_id": "device_123",
    "device_name": "iPhone 14 Pro",
    "device_type": "mobile",
    "os": "iOS 17.0",
    "app_version": "2.1.0"
  },
  "remember_me": true
}

Response: 200 OK
{
  "success": true,
  "data": {
    "user": { ... },
    "tokens": {
      "access_token": "eyJhbGc...",
      "refresh_token": "eyJhbGc...",
      "expires_in": 900,
      "token_type": "Bearer"
    },
    "requires_mfa": false,
    "session_id": "sess_xyz789"
  }
}
```

#### MFA Required Login Response
```json
Response: 200 OK (MFA Required)
{
  "success": true,
  "data": {
    "mfa_required": true,
    "mfa_token": "mfa_temp_token_abc123",
    "mfa_methods": ["totp", "sms", "email"],
    "preferred_method": "totp",
    "expires_in": 300
  }
}
```

#### Token Refresh
```http
POST /v1/auth/refresh
```
```json
Request:
{
  "refresh_token": "eyJhbGc..."
}

Response: 200 OK
{
  "success": true,
  "data": {
    "access_token": "eyJhbGc...",
    "refresh_token": "eyJhbGc...",
    "expires_in": 900,
    "token_type": "Bearer"
  }
}
```

#### Logout
```http
POST /v1/auth/logout
Authorization: Bearer <token>
```
```json
Request:
{
  "everywhere": false,  // Logout from all devices
  "device_id": "device_123"
}

Response: 200 OK
{
  "success": true,
  "message": "Successfully logged out"
}
```

#### Password Reset Request
```http
POST /v1/auth/password/forgot
```
```json
Request:
{
  "email": "user@example.com"
}

Response: 200 OK
{
  "success": true,
  "message": "If an account exists, a reset link has been sent",
  "expires_in": 3600
}
```

#### Password Reset Confirmation
```http
POST /v1/auth/password/reset
```
```json
Request:
{
  "token": "reset_token_abc123",
  "password": "NewSecureP@ssw0rd456!",
  "password_confirmation": "NewSecureP@ssw0rd456!"
}

Response: 200 OK
{
  "success": true,
  "message": "Password successfully reset",
  "auto_login": true,
  "tokens": { ... }
}
```

#### Email Verification
```http
POST /v1/auth/email/verify
```
```json
Request:
{
  "token": "verify_token_abc123"
}

Response: 200 OK
{
  "success": true,
  "message": "Email successfully verified",
  "rewards": {
    "credits_earned": 100,
    "badge_unlocked": "verified_user"
  }
}
```

#### Resend Verification
```http
POST /v1/auth/email/resend
Authorization: Bearer <token>
```
```json
Response: 200 OK
{
  "success": true,
  "message": "Verification email sent",
  "cooldown_seconds": 60
}
```

### Session Management

#### Get Active Sessions
```http
GET /v1/auth/sessions
Authorization: Bearer <token>
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "current_session": {
      "id": "sess_current",
      "device_name": "MacBook Pro",
      "location": "San Francisco, CA",
      "ip_address": "192.168.1.1",
      "last_active": "2024-01-01T00:00:00Z"
    },
    "other_sessions": [
      {
        "id": "sess_mobile",
        "device_name": "iPhone 14",
        "location": "New York, NY",
        "last_active": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

#### Revoke Session
```http
DELETE /v1/auth/sessions/:sessionId
Authorization: Bearer <token>
```

---

## 2. User Management & Profiles

### User Profile Endpoints

#### Get Current User
```http
GET /v1/users/me
Authorization: Bearer <token>
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "id": "usr_abc123",
    "email": "user@example.com",
    "username": "johndoe",
    "profile": {
      "first_name": "John",
      "last_name": "Doe",
      "display_name": "John D.",
      "avatar_url": "https://cdn.example.com/avatars/abc123.jpg",
      "bio": "Software developer and tech enthusiast",
      "location": "San Francisco, CA",
      "timezone": "America/Los_Angeles",
      "locale": "en-US"
    },
    "preferences": {
      "email_notifications": true,
      "push_notifications": false,
      "theme": "dark",
      "language": "en",
      "privacy_mode": "friends"
    },
    "stats": {
      "member_since": "2023-01-01",
      "last_login": "2024-01-01T00:00:00Z",
      "total_logins": 150,
      "storage_used": 1073741824,
      "api_calls_this_month": 5000
    },
    "subscription": {
      "plan": "pro",
      "status": "active",
      "expires_at": "2024-12-31T23:59:59Z"
    },
    "security": {
      "mfa_enabled": true,
      "mfa_methods": ["totp", "sms"],
      "password_last_changed": "2023-11-01T00:00:00Z",
      "verified_email": true,
      "verified_phone": false
    }
  }
}
```

#### Update Profile
```http
PATCH /v1/users/me
Authorization: Bearer <token>
```
```json
Request:
{
  "profile": {
    "first_name": "John",
    "last_name": "Smith",
    "bio": "Updated bio",
    "location": "Austin, TX"
  },
  "preferences": {
    "theme": "light",
    "email_notifications": false
  }
}

Response: 200 OK
{
  "success": true,
  "data": { ...updated user... },
  "changes": [
    "profile.last_name",
    "profile.bio",
    "profile.location",
    "preferences.theme",
    "preferences.email_notifications"
  ]
}
```

#### Upload Avatar
```http
POST /v1/users/me/avatar
Authorization: Bearer <token>
Content-Type: multipart/form-data
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "avatar_url": "https://cdn.example.com/avatars/new_abc123.jpg",
    "thumbnail_url": "https://cdn.example.com/avatars/new_abc123_thumb.jpg"
  }
}
```

#### Change Password
```http
POST /v1/users/me/password
Authorization: Bearer <token>
```
```json
Request:
{
  "current_password": "OldP@ssw0rd123!",
  "new_password": "NewP@ssw0rd456!",
  "new_password_confirmation": "NewP@ssw0rd456!",
  "logout_other_sessions": true
}

Response: 200 OK
{
  "success": true,
  "message": "Password successfully changed",
  "sessions_revoked": 3,
  "new_tokens": { ... }
}
```

#### Delete Account
```http
DELETE /v1/users/me
Authorization: Bearer <token>
```
```json
Request:
{
  "password": "CurrentP@ssw0rd123!",
  "reason": "no_longer_needed",
  "feedback": "Optional feedback text",
  "delete_immediately": false  // false = soft delete with 30-day recovery
}

Response: 200 OK
{
  "success": true,
  "message": "Account scheduled for deletion",
  "deletion_date": "2024-02-01T00:00:00Z",
  "recovery_token": "recovery_abc123"
}
```

#### Search Users (Public)
```http
GET /v1/users/search?q=john&limit=10
```
```json
Response: 200 OK
{
  "success": true,
  "data": [
    {
      "id": "usr_123",
      "username": "johndoe",
      "display_name": "John Doe",
      "avatar_url": "...",
      "verified": true,
      "badges": ["premium", "veteran"]
    }
  ],
  "meta": {
    "query": "john",
    "results": 10,
    "total_available": 25
  }
}
```

---

## 3. Role-Based Access Control (RBAC)

### Role Management

#### Get User Roles
```http
GET /v1/users/me/roles
Authorization: Bearer <token>
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "roles": [
      {
        "id": "role_admin",
        "name": "Admin",
        "description": "Administrative access",
        "priority": 100,
        "assigned_at": "2024-01-01T00:00:00Z",
        "assigned_by": "usr_superadmin",
        "expires_at": null
      },
      {
        "id": "role_moderator",
        "name": "Moderator",
        "description": "Content moderation",
        "priority": 50,
        "assigned_at": "2024-01-01T00:00:00Z",
        "expires_at": "2024-12-31T23:59:59Z"
      }
    ],
    "effective_permissions": [
      "users:read",
      "users:write",
      "users:delete",
      "content:moderate",
      "reports:view"
    ]
  }
}
```

#### Check Permissions
```http
POST /v1/auth/permissions/check
Authorization: Bearer <token>
```
```json
Request:
{
  "permissions": ["users:delete", "billing:manage"],
  "resource": "usr_456",  // Optional: specific resource
  "context": {  // Optional: additional context
    "ip_address": "192.168.1.1",
    "action": "delete_user"
  }
}

Response: 200 OK
{
  "success": true,
  "data": {
    "results": {
      "users:delete": {
        "allowed": true,
        "source": "role:admin",
        "conditions": []
      },
      "billing:manage": {
        "allowed": false,
        "reason": "insufficient_permissions",
        "required_role": "super_admin"
      }
    },
    "has_all": false,
    "has_any": true
  }
}
```

### Admin Role Operations

#### List All Roles
```http
GET /v1/admin/roles
Authorization: Bearer <admin-token>
```

#### Create Role
```http
POST /v1/admin/roles
Authorization: Bearer <admin-token>
```
```json
Request:
{
  "name": "content_creator",
  "description": "Can create and edit content",
  "permissions": [
    "content:create",
    "content:edit",
    "content:publish"
  ],
  "priority": 30,
  "metadata": {
    "department": "marketing",
    "max_users": 50
  }
}
```

#### Assign Role to User
```http
POST /v1/admin/users/:userId/roles
Authorization: Bearer <admin-token>
```
```json
Request:
{
  "role_id": "role_moderator",
  "expires_at": "2024-12-31T23:59:59Z",
  "reason": "Promoted to moderator",
  "notify_user": true
}
```

---

## 4. Multi-Factor Authentication (MFA)

### MFA Setup & Management

#### Get MFA Status
```http
GET /v1/mfa/status
Authorization: Bearer <token>
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "enabled": true,
    "methods": [
      {
        "type": "totp",
        "enabled": true,
        "verified": true,
        "name": "Google Authenticator",
        "last_used": "2024-01-01T00:00:00Z"
      },
      {
        "type": "sms",
        "enabled": true,
        "verified": true,
        "phone_number": "*****7890",
        "last_used": null
      }
    ],
    "backup_codes": {
      "remaining": 8,
      "total": 10,
      "last_generated": "2023-01-01T00:00:00Z"
    },
    "trusted_devices": [
      {
        "id": "device_123",
        "name": "MacBook Pro",
        "trusted_until": "2024-02-01T00:00:00Z"
      }
    ]
  }
}
```

#### Setup TOTP
```http
POST /v1/mfa/totp/setup
Authorization: Bearer <token>
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "secret": "JBSWY3DPEHPK3PXP",
    "qr_code": "data:image/png;base64,iVBORw0...",
    "manual_entry_key": "JBSW Y3DP EHPK 3PXP",
    "verification_required": true
  }
}
```

#### Verify TOTP Setup
```http
POST /v1/mfa/totp/verify
Authorization: Bearer <token>
```
```json
Request:
{
  "code": "123456",
  "device_name": "Google Authenticator on iPhone"
}

Response: 200 OK
{
  "success": true,
  "data": {
    "backup_codes": [
      "ABC123DEF",
      "GHI456JKL",
      "MNO789PQR"
    ],
    "download_link": "https://api.example.com/v1/mfa/backup-codes/download?token=..."
  }
}
```

#### Setup SMS MFA
```http
POST /v1/mfa/sms/setup
Authorization: Bearer <token>
```
```json
Request:
{
  "phone_number": "+1234567890"
}

Response: 200 OK
{
  "success": true,
  "message": "Verification code sent to +******7890",
  "expires_in": 300
}
```

#### Complete MFA Challenge
```http
POST /v1/mfa/challenge
```
```json
Request:
{
  "mfa_token": "mfa_temp_token_abc123",
  "method": "totp",
  "code": "123456",
  "trust_device": true,
  "trust_duration_days": 30
}

Response: 200 OK
{
  "success": true,
  "data": {
    "user": { ... },
    "tokens": {
      "access_token": "eyJhbGc...",
      "refresh_token": "eyJhbGc..."
    },
    "device_trusted": true,
    "trust_expires": "2024-02-01T00:00:00Z"
  }
}
```

#### Regenerate Backup Codes
```http
POST /v1/mfa/backup-codes/regenerate
Authorization: Bearer <token>
```
```json
Request:
{
  "password": "CurrentP@ssw0rd123!"
}

Response: 200 OK
{
  "success": true,
  "data": {
    "backup_codes": [
      "NEW123ABC",
      "NEW456DEF",
      "NEW789GHI"
    ],
    "old_codes_invalidated": true
  }
}
```

---

## 5. Subscription & Billing Management

### Subscription Plans

#### Get Available Plans
```http
GET /v1/billing/plans
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "plans": [
      {
        "id": "plan_free",
        "name": "Free",
        "description": "Perfect for getting started",
        "price": {
          "amount": 0,
          "currency": "USD",
          "interval": null
        },
        "features": [
          "5 projects",
          "1GB storage",
          "Community support"
        ],
        "limits": {
          "projects": 5,
          "storage_gb": 1,
          "api_calls_per_month": 1000
        }
      },
      {
        "id": "plan_pro",
        "name": "Professional",
        "description": "For growing teams",
        "price": {
          "amount": 2900,
          "currency": "USD",
          "interval": "month",
          "yearly_discount": 20
        },
        "features": [
          "Unlimited projects",
          "100GB storage",
          "Priority support",
          "Advanced analytics"
        ],
        "popular": true,
        "trial_days": 14
      }
    ],
    "addons": [
      {
        "id": "addon_storage",
        "name": "Extra Storage",
        "price_per_unit": 500,
        "unit": "10GB"
      }
    ]
  }
}
```

#### Get Current Subscription
```http
GET /v1/billing/subscription
Authorization: Bearer <token>
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "subscription": {
      "id": "sub_abc123",
      "plan_id": "plan_pro",
      "status": "active",
      "current_period_start": "2024-01-01T00:00:00Z",
      "current_period_end": "2024-01-31T23:59:59Z",
      "cancel_at_period_end": false,
      "trial_end": null
    },
    "usage": {
      "projects": {
        "used": 15,
        "limit": null,
        "percentage": null
      },
      "storage_gb": {
        "used": 45.7,
        "limit": 100,
        "percentage": 45.7
      },
      "api_calls": {
        "used": 8500,
        "limit": 50000,
        "percentage": 17
      }
    },
    "billing": {
      "next_billing_date": "2024-02-01T00:00:00Z",
      "amount": 2900,
      "currency": "USD",
      "payment_method": {
        "type": "card",
        "last4": "4242",
        "brand": "visa"
      }
    },
    "addons": [
      {
        "id": "addon_storage",
        "quantity": 2,
        "total": 1000
      }
    ]
  }
}
```

#### Create/Update Subscription
```http
POST /v1/billing/subscription
Authorization: Bearer <token>
```
```json
Request:
{
  "plan_id": "plan_pro",
  "interval": "yearly",  // monthly or yearly
  "payment_method_id": "pm_abc123",
  "coupon_code": "SAVE20",
  "addons": [
    {
      "id": "addon_storage",
      "quantity": 2
    }
  ]
}

Response: 200 OK
{
  "success": true,
  "data": {
    "subscription": { ... },
    "invoice": {
      "id": "inv_123",
      "amount": 27840,  // yearly with discount
      "discount": 6960,
      "total": 27840,
      "pdf_url": "https://api.example.com/v1/billing/invoices/inv_123/download"
    },
    "trial_ends": "2024-01-15T00:00:00Z"
  }
}
```

#### Cancel Subscription
```http
DELETE /v1/billing/subscription
Authorization: Bearer <token>
```
```json
Request:
{
  "reason": "too_expensive",
  "feedback": "Optional feedback",
  "cancel_immediately": false  // false = cancel at period end
}

Response: 200 OK
{
  "success": true,
  "data": {
    "subscription_ends": "2024-01-31T23:59:59Z",
    "refund_amount": 0,
    "retention_offer": {
      "discount_percentage": 50,
      "valid_until": "2024-01-07T00:00:00Z",
      "offer_code": "COMEBACK50"
    }
  }
}
```

#### Get Payment Methods
```http
GET /v1/billing/payment-methods
Authorization: Bearer <token>
```

#### Add Payment Method
```http
POST /v1/billing/payment-methods
Authorization: Bearer <token>
```
```json
Request:
{
  "type": "card",
  "stripe_payment_method_id": "pm_abc123",
  "set_as_default": true
}
```

#### Get Invoices
```http
GET /v1/billing/invoices?limit=10
Authorization: Bearer <token>
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "invoices": [
      {
        "id": "inv_123",
        "date": "2024-01-01T00:00:00Z",
        "amount": 2900,
        "status": "paid",
        "pdf_url": "...",
        "description": "Pro Plan - January 2024"
      }
    ]
  }
}
```

#### Apply Coupon
```http
POST /v1/billing/coupons/apply
Authorization: Bearer <token>
```
```json
Request:
{
  "coupon_code": "NEWYEAR2024"
}

Response: 200 OK
{
  "success": true,
  "data": {
    "discount": {
      "percentage": 25,
      "amount": 725,
      "applies_to": "next_payment",
      "expires": "2024-01-31T23:59:59Z"
    }
  }
}
```

---

## 6. Audit & Compliance

### Audit Log Access

#### Get User Activity Log
```http
GET /v1/audit/activity?limit=50&offset=0
Authorization: Bearer <token>
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "activities": [
      {
        "id": "audit_123",
        "timestamp": "2024-01-01T12:00:00Z",
        "event_type": "user.login",
        "description": "Successfully logged in",
        "ip_address": "192.168.1.1",
        "user_agent": "Mozilla/5.0...",
        "location": "San Francisco, CA",
        "device": "Chrome on MacOS",
        "risk_score": "low"
      },
      {
        "id": "audit_124",
        "timestamp": "2024-01-01T11:00:00Z",
        "event_type": "user.password.changed",
        "description": "Password was changed",
        "metadata": {
          "required_mfa": true,
          "sessions_invalidated": 2
        }
      }
    ],
    "summary": {
      "total_events": 150,
      "security_events": 5,
      "last_security_event": "2024-01-01T00:00:00Z"
    }
  }
}
```

#### Get Security Events
```http
GET /v1/audit/security-events
Authorization: Bearer <token>
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "events": [
      {
        "id": "sec_001",
        "type": "suspicious_login",
        "timestamp": "2024-01-01T00:00:00Z",
        "severity": "medium",
        "description": "Login from new location",
        "location": "Tokyo, Japan",
        "action_taken": "mfa_required",
        "resolved": true
      }
    ],
    "unresolved_count": 0
  }
}
```

#### Download Activity Report
```http
GET /v1/audit/export?format=pdf&date_from=2024-01-01&date_to=2024-01-31
Authorization: Bearer <token>
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "report_url": "https://api.example.com/reports/temp/report_abc123.pdf",
    "expires_in": 3600,
    "format": "pdf",
    "events_count": 523
  }
}
```

### GDPR Compliance

#### Data Export Request
```http
POST /v1/compliance/gdpr/export
Authorization: Bearer <token>
```
```json
Request:
{
  "password": "CurrentP@ssw0rd123!",
  "format": "json",  // json or csv
  "include_metadata": true
}

Response: 202 Accepted
{
  "success": true,
  "data": {
    "request_id": "export_abc123",
    "status": "processing",
    "estimated_time": 3600,
    "notification_method": "email"
  }
}
```

#### Data Deletion Request
```http
POST /v1/compliance/gdpr/delete
Authorization: Bearer <token>
```
```json
Request:
{
  "password": "CurrentP@ssw0rd123!",
  "reason": "user_request",
  "delete_all": true,
  "acknowledgment": "I understand this action is irreversible"
}

Response: 200 OK
{
  "success": true,
  "data": {
    "scheduled_deletion": "2024-02-01T00:00:00Z",
    "grace_period_days": 30,
    "cancellation_token": "cancel_abc123"
  }
}
```

---

## 7. Feature Flags & A/B Testing

### Feature Flag Evaluation

#### Get User Feature Flags
```http
GET /v1/features/flags
Authorization: Bearer <token>
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "flags": {
      "new_dashboard": {
        "enabled": true,
        "variant": "version_b",
        "metadata": {
          "rollout_percentage": 50,
          "user_in_experiment": true
        }
      },
      "ai_assistant": {
        "enabled": false,
        "reason": "not_in_rollout",
        "available_in": "plan_enterprise"
      },
      "dark_mode": {
        "enabled": true,
        "value": "auto"  // String flag type
      },
      "api_rate_limit": {
        "enabled": true,
        "value": 10000  // Number flag type
      }
    },
    "experiments": [
      {
        "id": "exp_homepage",
        "variant": "control",
        "enrolled": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

#### Bulk Evaluate Flags
```http
POST /v1/features/evaluate
Authorization: Bearer <token>
```
```json
Request:
{
  "flags": ["new_dashboard", "ai_assistant", "beta_features"],
  "context": {
    "user_agent": "Mobile App v2.1.0",
    "location": "US",
    "subscription_plan": "pro"
  }
}

Response: 200 OK
{
  "success": true,
  "data": {
    "new_dashboard": true,
    "ai_assistant": false,
    "beta_features": true
  }
}
```

#### Track Feature Usage
```http
POST /v1/features/track
Authorization: Bearer <token>
```
```json
Request:
{
  "flag": "new_dashboard",
  "event": "viewed",
  "metadata": {
    "duration_seconds": 45,
    "interactions": 12
  }
}
```

---

## 8. Admin Dashboard APIs

### User Management (Admin)

#### List All Users
```http
GET /v1/admin/users?page=1&limit=50&status=active&role=user
Authorization: Bearer <admin-token>
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "users": [
      {
        "id": "usr_123",
        "email": "user@example.com",
        "username": "johndoe",
        "status": "active",
        "roles": ["user"],
        "subscription": "pro",
        "created_at": "2024-01-01T00:00:00Z",
        "last_login": "2024-01-15T00:00:00Z",
        "risk_score": "low",
        "flags": []
      }
    ],
    "stats": {
      "total_users": 10000,
      "active_today": 2500,
      "new_this_month": 500,
      "churn_rate": 2.5
    }
  }
}
```

#### Get User Details (Admin)
```http
GET /v1/admin/users/:userId
Authorization: Bearer <admin-token>
```

#### Suspend User
```http
POST /v1/admin/users/:userId/suspend
Authorization: Bearer <admin-token>
```
```json
Request:
{
  "reason": "terms_violation",
  "duration_days": 30,
  "message": "Your account has been suspended for violating terms of service",
  "notify_user": true
}
```

#### Admin Password Reset
```http
POST /v1/admin/users/:userId/reset-password
Authorization: Bearer <admin-token>
```

### System Monitoring

#### Get System Stats
```http
GET /v1/admin/system/stats
Authorization: Bearer <admin-token>
```
```json
Response: 200 OK
{
  "success": true,
  "data": {
    "users": {
      "total": 10000,
      "active_30d": 7500,
      "new_30d": 500
    },
    "subscriptions": {
      "active": 2500,
      "mrr": 72500,
      "churn_rate": 2.5
    },
    "api": {
      "calls_today": 1500000,
      "average_latency_ms": 45,
      "error_rate": 0.01
    },
    "security": {
      "failed_logins_24h": 150,
      "blocked_ips": 25,
      "mfa_adoption": 65.5
    }
  }
}
```

#### Get Audit Logs (Admin)
```http
GET /v1/admin/audit/logs?severity=error&limit=100
Authorization: Bearer <admin-token>
```

---

## 9. Real-time & Webhooks

### WebSocket Connection

#### Establish WebSocket
```javascript
// Client-side connection
const ws = new WebSocket('wss://api.example.com/v1/realtime');

// Authentication
ws.send(JSON.stringify({
  type: 'auth',
  token: 'Bearer eyJhbGc...'
}));

// Subscribe to channels
ws.send(JSON.stringify({
  type: 'subscribe',
  channels: ['notifications', 'presence', 'updates']
}));
```

#### Real-time Events
```json
// Notification Event
{
  "type": "notification",
  "data": {
    "id": "notif_123",
    "title": "New login from Chrome",
    "body": "We noticed a new login to your account",
    "action": "review_activity",
    "priority": "medium"
  }
}

// Presence Update
{
  "type": "presence",
  "data": {
    "user_id": "usr_456",
    "status": "online",
    "last_seen": "2024-01-01T00:00:00Z"
  }
}

// Live Update
{
  "type": "update",
  "data": {
    "entity": "subscription",
    "action": "upgraded",
    "details": {
      "new_plan": "enterprise",
      "effective_date": "2024-01-01T00:00:00Z"
    }
  }
}
```

### Webhook Management

#### Register Webhook
```http
POST /v1/webhooks
Authorization: Bearer <token>
```
```json
Request:
{
  "url": "https://myapp.com/webhooks/umanager",
  "events": [
    "user.created",
    "user.updated",
    "subscription.changed",
    "security.alert"
  ],
  "secret": "webhook_secret_abc123",
  "active": true
}

Response: 201 Created
{
  "success": true,
  "data": {
    "id": "webhook_123",
    "url": "https://myapp.com/webhooks/umanager",
    "events": [...],
    "created_at": "2024-01-01T00:00:00Z",
    "signing_secret": "whsec_abc123xyz"
  }
}
```

#### List Webhooks
```http
GET /v1/webhooks
Authorization: Bearer <token>
```

#### Test Webhook
```http
POST /v1/webhooks/:webhookId/test
Authorization: Bearer <token>
```

### Webhook Event Format
```json
// Webhook POST to registered URL
Headers:
{
  "X-Webhook-Id": "webhook_123",
  "X-Webhook-Timestamp": "1704067200",
  "X-Webhook-Signature": "sha256=abc123...",
  "Content-Type": "application/json"
}

Body:
{
  "id": "evt_abc123",
  "type": "user.updated",
  "created": "2024-01-01T00:00:00Z",
  "data": {
    "user": {
      "id": "usr_123",
      "email": "user@example.com",
      "changes": ["profile.bio", "profile.location"]
    }
  }
}
```

---

## 10. Public APIs

### Health & Status

#### Health Check
```http
GET /v1/health
```
```json
Response: 200 OK
{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z",
  "version": "1.0.0",
  "uptime_seconds": 864000
}
```

#### Detailed Health (Authenticated)
```http
GET /v1/health/detailed
Authorization: Bearer <token>
```
```json
Response: 200 OK
{
  "status": "healthy",
  "services": {
    "database": "healthy",
    "redis": "healthy",
    "nats": "healthy",
    "stripe": "healthy"
  },
  "metrics": {
    "response_time_ms": 12,
    "cpu_usage": 45.2,
    "memory_usage": 62.5,
    "active_connections": 1250
  }
}
```

### API Information

#### Get API Version
```http
GET /v1/info
```
```json
Response: 200 OK
{
  "version": "1.0.0",
  "environment": "production",
  "documentation": "https://docs.api.example.com",
  "support": "support@example.com",
  "status_page": "https://status.example.com"
}
```

#### Get Rate Limit Status
```http
GET /v1/rate-limit
Authorization: Bearer <token>
```
```json
Response: 200 OK
Headers:
{
  "X-RateLimit-Limit": "10000",
  "X-RateLimit-Remaining": "9875",
  "X-RateLimit-Reset": "1704070800"
}

Body:
{
  "success": true,
  "data": {
    "limit": 10000,
    "remaining": 9875,
    "reset_at": "2024-01-01T01:00:00Z",
    "tier": "pro"
  }
}
```

---

## API Standards & Conventions

### HTTP Status Codes
- `200 OK` - Successful GET/PUT/PATCH
- `201 Created` - Successful POST creating resource
- `202 Accepted` - Request accepted for async processing
- `204 No Content` - Successful DELETE
- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Missing/invalid authentication
- `403 Forbidden` - Authenticated but not authorized
- `404 Not Found` - Resource doesn't exist
- `409 Conflict` - Resource conflict (duplicate email, etc.)
- `422 Unprocessable Entity` - Validation errors
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error
- `503 Service Unavailable` - Maintenance mode

### Pagination
```http
GET /v1/resource?page=2&per_page=20&sort=created_at&order=desc
```

### Filtering
```http
GET /v1/resource?status=active&created_after=2024-01-01&tags=important,urgent
```

### Field Selection
```http
GET /v1/resource?fields=id,name,email,profile.avatar_url
```

### Search
```http
GET /v1/resource?q=search+term&search_fields=name,email,bio
```

### Rate Limiting Headers
```http
X-RateLimit-Limit: 10000
X-RateLimit-Remaining: 9999
X-RateLimit-Reset: 1704067200
Retry-After: 3600 (when rate limited)
```

### CORS Headers
```http
Access-Control-Allow-Origin: https://app.example.com
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type, X-Request-ID
Access-Control-Allow-Credentials: true
Access-Control-Max-Age: 86400
```

### Security Headers
```http
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
```

### Idempotency
```http
POST /v1/resource
Idempotency-Key: unique-key-123
```

### API Versioning Strategy
- Version in URL path: `/v1/`, `/v2/`
- Breaking changes require new version
- Deprecation notices via headers:
  ```http
  Sunset: Sat, 31 Dec 2024 23:59:59 GMT
  Deprecation: true
  Link: <https://api.example.com/v2/resource>; rel="successor-version"
  ```

### Error Code Reference
```json
{
  "AUTH_INVALID_TOKEN": "Invalid or expired token",
  "AUTH_MFA_REQUIRED": "Multi-factor authentication required",
  "USER_NOT_FOUND": "User does not exist",
  "USER_ALREADY_EXISTS": "User with this email already exists",
  "SUBSCRIPTION_EXPIRED": "Subscription has expired",
  "RATE_LIMIT_EXCEEDED": "Too many requests",
  "VALIDATION_FAILED": "Request validation failed",
  "PERMISSION_DENIED": "Insufficient permissions",
  "RESOURCE_LOCKED": "Resource is locked for editing",
  "PAYMENT_FAILED": "Payment processing failed"
}
```

---

## Implementation Priority

### Phase 1: Core Authentication (Week 1-2)
1. Registration & Login
2. Token Management
3. Password Reset
4. Email Verification
5. Session Management

### Phase 2: User Management (Week 3)
1. User Profiles
2. Preferences
3. Avatar Upload
4. Account Management

### Phase 3: Security (Week 4)
1. MFA Setup
2. TOTP Implementation
3. Backup Codes
4. Security Settings

### Phase 4: RBAC (Week 5)
1. Role Management
2. Permission Checking
3. Admin Functions

### Phase 5: Billing (Week 6-7)
1. Subscription Management
2. Payment Processing
3. Invoice Generation
4. Usage Tracking

### Phase 6: Advanced Features (Week 8-9)
1. Audit Logging
2. Feature Flags
3. Real-time Updates
4. Webhooks

### Phase 7: Admin & Analytics (Week 10)
1. Admin Dashboard APIs
2. Analytics Endpoints
3. Reporting Tools
4. System Monitoring

This comprehensive API design provides a modern, secure, and user-friendly interface for all the features in the user management system. The design follows REST best practices, includes proper error handling, and provides a consistent experience across all endpoints.