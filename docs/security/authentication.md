# Authentication System Documentation

## Authentication Overview

The Document Management Platform implements a robust authentication system based on JSON Web Tokens (JWT) with RS256 signatures. This system ensures secure access to the platform while maintaining strict tenant isolation and enforcing role-based access control.

### Key Features

- JWT-based authentication with RS256 signatures
- Complete tenant isolation for multi-tenant security
- Role-based access control with predefined system roles
- Stateless authentication with token-based sessions
- Secure token handling with proper expiration policies

### Authentication Flow

1. Client obtains JWT from authentication provider (outside system scope)
2. JWT contains claims for user identity, tenant ID, and roles
3. API Gateway validates JWT signature and expiration
4. Tenant context is extracted and passed to downstream services
5. Failed authentication returns 401 Unauthorized

## JWT Implementation

The platform uses JSON Web Tokens (JWT) signed with RS256 (RSA Signature with SHA-256) to provide secure, stateless authentication.

### JWT Structure

Each JWT token contains the following claims:

| Claim | Purpose | Validation Rules |
| ----- | ------- | ---------------- |
| sub | User identifier | Must be present and valid UUID |
| tenant_id | Tenant identifier | Must be present and valid UUID |
| roles | User roles array | Must contain at least one valid role |
| exp | Expiration timestamp | Must be in the future |
| iat | Issued at timestamp | Must be in the past |
| iss | Token issuer | Must match configured issuer |

### Token Types

The system uses two types of tokens:

- **Access Token**
  - Short-lived token (15-60 minutes) used for API access
  - Contains user ID, tenant ID, and roles

- **Refresh Token**
  - Longer-lived token (7 days) used to obtain new access tokens
  - Contains user ID and tenant ID (no roles)

### Token Validation Process

1. Extract token from Authorization header
2. Verify token signature using public key
3. Validate token expiration and issuer
4. Extract user ID, tenant ID, and roles
5. Verify user exists and is active
6. Verify tenant exists and is active
7. Set user and tenant context for the request

## Role-Based Access Control

The platform implements a comprehensive role-based access control (RBAC) system to manage permissions across the application.

### System Roles

The platform defines the following predefined roles:

| Role | Permissions | Scope |
| ---- | ----------- | ----- |
| Reader | View documents and folders | Tenant-wide or specific folders |
| Contributor | Reader + upload, update documents | Tenant-wide or specific folders |
| Editor | Contributor + delete documents | Tenant-wide or specific folders |
| Administrator | All operations including folder management | Tenant-wide |
| System | Special role for internal operations | System-wide |

### Permission Types

The system defines the following permission types:

- **read**
  - Permission to view documents and folders
  - Roles: All roles

- **write**
  - Permission to create and update documents
  - Roles: Contributor, Editor, Administrator, System

- **delete**
  - Permission to delete documents
  - Roles: Editor, Administrator, System

- **manage_folders**
  - Permission to create, update, and delete folders
  - Roles: Administrator, System

### Role Assignment

Roles are assigned to users within a tenant. A user can have multiple roles, and their effective permissions are the union of all permissions granted by their roles.

### Permission Verification

The system verifies permissions at multiple levels:
1. API Gateway middleware for basic role requirements
2. Service layer for fine-grained permission checks
3. Resource-level access control for specific documents and folders

## Tenant Isolation

The platform implements strict tenant isolation to ensure complete separation of data between different customer organizations.

### Tenant Context

Every authenticated request includes a tenant context derived from the JWT token. This tenant context is used to filter all data access and ensure users can only access resources within their own tenant.

### Tenant Validation

The system validates tenant access at multiple levels:
1. JWT validation ensures the tenant ID claim is present and valid
2. User validation ensures the user belongs to the claimed tenant
3. Resource access validation ensures resources are accessed only by users in the same tenant
4. Database queries include tenant filters to prevent cross-tenant data access

### Cross-Tenant Protection

The system implements several safeguards against cross-tenant access:
1. Tenant middleware that enforces tenant boundaries
2. Database query filters that automatically apply tenant context
3. Logging of all cross-tenant access attempts as security incidents
4. Regular security audits to verify tenant isolation

## Authentication Middleware

The platform uses middleware components to enforce authentication, authorization, and tenant isolation across all API endpoints.

### Authentication Middleware

Validates JWT tokens and extracts user information:
1. Extracts the Authorization header from the request
2. Validates the JWT token signature and expiration
3. Extracts user ID, tenant ID, and roles from the token
4. Sets these values in the request context for downstream handlers

### Role Requirement Middleware

Enforces role-based access control:
1. RequireAuthentication: Ensures the request is authenticated
2. RequireRole: Ensures the user has a specific role
3. RequireAnyRole: Ensures the user has at least one of the specified roles

### Tenant Middleware

Enforces tenant isolation:
1. RequireTenantContext: Ensures the request has a valid tenant context
2. RequireSameTenant: Ensures the tenant ID in the request path matches the user's tenant
3. VerifyTenantResourceAccess: Verifies a user has access to a resource within their tenant
4. TenantContext: Adds tenant context to all database operations

## Security Considerations

The authentication system is designed with security best practices in mind.

### Token Security

1. Short-lived access tokens (15-60 minutes)
2. Secure token storage (client responsibility)
3. HTTPS-only transmission
4. RS256 signatures with 2048-bit keys
5. Regular key rotation

### Password Security

1. Passwords stored using bcrypt hashing
2. Minimum password strength requirements
3. Account lockout after failed attempts
4. Regular password rotation policies

### Audit Logging

All authentication events are logged for security monitoring and compliance:
1. Authentication attempts (successful and failed)
2. Token validation failures
3. Permission denials
4. Cross-tenant access attempts

## Integration Guide

This section provides guidance for integrating with the authentication system.

### Obtaining Tokens

To obtain authentication tokens, make a POST request to the authentication endpoint with valid credentials:

```
POST /api/v1/auth/login
Content-Type: application/json

{
  "tenant_id": "your-tenant-id",
  "username": "your-username",
  "password": "your-password"
}
```

The response will include an access token and refresh token:

```json
{
  "access_token": "eyJhbGciOiJSUzI1...",
  "refresh_token": "eyJhbGciOiJSUzI1...",
  "expires_in": 3600
}
```

### Using Tokens

Include the access token in the Authorization header of all API requests:

```
GET /api/v1/documents
Authorization: Bearer eyJhbGciOiJSUzI1...
```

### Refreshing Tokens

When an access token expires, use the refresh token to obtain a new one:

```
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJSUzI1..."
}
```

The response will include a new access token and refresh token.

### Error Handling

Authentication errors return standard HTTP status codes:
- 401 Unauthorized: Invalid or expired token
- 403 Forbidden: Insufficient permissions
- 400 Bad Request: Invalid request format

Error responses include detailed information:

```json
{
  "error": {
    "code": "authentication_error",
    "message": "Token has expired"
  }
}
```

## Troubleshooting

Common authentication issues and their solutions.

### Invalid Token Errors

- Check that the token is correctly formatted and includes the 'Bearer ' prefix
- Verify that the token has not expired
- Ensure the token was issued by the correct issuer
- Check that the token signature is valid

### Permission Denied Errors

- Verify that the user has the required role for the operation
- Check that the user belongs to the correct tenant
- Ensure the resource being accessed belongs to the user's tenant
- Review the audit logs for detailed permission denial reasons

### Token Refresh Issues

- Verify that the refresh token has not expired
- Ensure the refresh token was issued to the same user
- Check that the user and tenant are still active
- If the refresh token is invalid, a new login is required