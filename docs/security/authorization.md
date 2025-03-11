# Authorization System

## Authorization Overview

The Document Management Platform implements a comprehensive authorization system based on role-based access control (RBAC) with tenant isolation. This system ensures that users can only access resources they are permitted to, while maintaining strict separation between different tenant organizations.

### Key Features

- Role-based access control with predefined system roles
- Resource-level permission management
- Complete tenant isolation for multi-tenant security
- Hierarchical permission inheritance for folder structures
- Fine-grained access control for documents and folders

### Authorization Flow

1. Request arrives with authenticated JWT token
2. User identity, tenant, and roles are extracted from token
3. Tenant isolation is enforced for all operations
4. Role-based permissions are checked for the requested operation
5. Resource-specific permissions are verified
6. Access is granted or denied based on permission evaluation
7. All authorization decisions are logged for audit purposes

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

Roles are assigned to users within a tenant. A user can have multiple roles, and their effective permissions are the union of all permissions granted by their roles. Roles can be assigned at the tenant level (applying to all resources) or at the folder level (applying only to that folder and its contents).

## Permission Management

The platform provides a flexible permission management system that supports different permission sources and inheritance models.

### Permission Sources

Permissions can come from multiple sources:

- **Direct Permissions**
  - Permissions explicitly assigned to a user for a specific resource

- **Role-based Permissions**
  - Permissions derived from the roles assigned to a user

- **Inherited Permissions**
  - Permissions inherited from parent folders in the hierarchy

### Permission Inheritance

The system implements hierarchical permission inheritance for folder structures:

1. Permissions assigned to a folder apply to all documents within that folder
2. Permissions assigned to a folder apply to all subfolders unless explicitly overridden
3. Child resources can have additional permissions beyond what is inherited
4. Permission inheritance can be disabled for specific folders if needed

### Permission Evaluation

When evaluating if a user has permission for an operation, the system:

1. Checks if the user has direct permission for the resource
2. Checks if any of the user's roles grant the required permission
3. Checks if permission is inherited from a parent folder
4. Grants access if any of these checks succeed
5. Denies access if all checks fail

## Resource Authorization

The platform implements resource-level authorization to control access to specific documents and folders.

### Resource Types

The system defines two primary resource types:

- **document**
  - Represents a document in the system

- **folder**
  - Represents a folder that can contain documents and other folders

### Resource Access Control

Access to resources is controlled through a combination of:

1. Resource ownership - creators have full control over their resources
2. Explicit permissions - directly assigned to users for specific resources
3. Role-based permissions - derived from user roles
4. Inherited permissions - derived from the resource hierarchy

### Resource Authorization Flow

The authorization flow for resource access is as follows:

1. Extract user ID and tenant ID from the authenticated request
2. Verify the user belongs to the claimed tenant
3. Identify the resource being accessed (document or folder)
4. Determine the required permission for the operation (read, write, delete, manage_folders)
5. Check if the user has the required permission through any permission source
6. Grant or deny access based on the permission check
7. Log the authorization decision for audit purposes

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

## Authorization Middleware

The platform uses middleware components to enforce authorization and tenant isolation across all API endpoints.

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

### Resource Access Middleware

Enforces resource-level access control:

1. VerifyResourceAccess: Verifies a user has the required permission for a resource
2. VerifyDocumentAccess: Specialized middleware for document access control
3. VerifyFolderAccess: Specialized middleware for folder access control

## Implementing Authorization

This section provides guidance for implementing authorization in the Document Management Platform.

### Checking Permissions

To check if a user has permission for an operation, use the AuthService:

```go
// Check if user has a specific permission
hasPermission, err := authService.VerifyPermission(ctx, userID, tenantID, "write")

// Check if user has access to a specific resource
hasAccess, err := authService.VerifyResourceAccess(ctx, userID, tenantID, "document", documentID, "write")
```

### Applying Middleware

Apply authorization middleware to API routes:

```go
// Require authentication for all routes in this group
authorizedGroup := router.Group("/api/v1").Use(middleware.RequireAuthentication())

// Require specific role for certain operations
adminGroup := authorizedGroup.Group("/admin").Use(middleware.RequireRole("administrator"))

// Verify resource access for document operations
documentGroup.GET("/:id", middleware.VerifyTenantResourceAccess(authService, "document", "read"), handlers.GetDocument)
documentGroup.PUT("/:id", middleware.VerifyTenantResourceAccess(authService, "document", "write"), handlers.UpdateDocument)
documentGroup.DELETE("/:id", middleware.VerifyTenantResourceAccess(authService, "document", "delete"), handlers.DeleteDocument)
```

### Managing Permissions

To manage permissions programmatically:

```go
// Create a new permission
permission := models.NewPermission(
    roleID,
    models.ResourceTypeDocument,
    documentID,
    models.PermissionTypeWrite,
    tenantID,
    createdByUserID,
)

// Save the permission
permissionID, err := permissionRepository.Create(ctx, permission)

// Check if a permission exists
exists, err := permissionRepository.Exists(ctx, roleID, models.ResourceTypeDocument, documentID, models.PermissionTypeWrite)
```

## Authorization Best Practices

Best practices for implementing and using the authorization system.

### Principle of Least Privilege

1. Assign the minimum permissions necessary for users to perform their tasks
2. Use more restrictive roles (Reader, Contributor) instead of Administrator when possible
3. Apply permissions at the folder level rather than tenant-wide when appropriate
4. Regularly audit and review permissions to remove unnecessary access

### Defense in Depth

1. Implement authorization checks at multiple levels (API, service, data)
2. Always verify tenant context for every operation
3. Don't rely solely on frontend controls - always enforce permissions in the backend
4. Log all authorization decisions for audit and security monitoring

### Performance Considerations

1. Cache frequently used permissions to reduce database load
2. Use efficient permission checking algorithms for deeply nested folder structures
3. Consider denormalizing permissions for frequently accessed resources
4. Monitor authorization performance and optimize as needed

## Audit and Compliance

The authorization system includes comprehensive audit logging to support security monitoring and compliance requirements.

### Authorization Audit Logs

All authorization events are logged for security monitoring and compliance:

1. Permission checks (successful and failed)
2. Permission changes (create, update, delete)
3. Cross-tenant access attempts
4. Role assignments and changes

### Compliance Support

The authorization system supports compliance requirements through:

1. Complete audit trail of all access decisions
2. Strict tenant isolation for data segregation
3. Fine-grained permission controls
4. Regular permission reviews and reports

### Security Monitoring

Authorization logs are integrated with security monitoring systems to detect:

1. Unusual permission changes
2. Excessive permission denials
3. Potential privilege escalation attempts
4. Cross-tenant access attempts

## Troubleshooting

Common authorization issues and their solutions.

### Permission Denied Errors

- Verify that the user has the required role for the operation
- Check that the user belongs to the correct tenant
- Ensure the resource being accessed belongs to the user's tenant
- Review the audit logs for detailed permission denial reasons
- Check if the permission might be inherited from a parent folder

### Tenant Isolation Issues

- Verify that tenant context is correctly set in all requests
- Check that database queries include tenant filters
- Ensure middleware is correctly applied to all routes
- Review logs for any cross-tenant access attempts

### Role and Permission Management

- Verify that roles are correctly assigned to users
- Check that permissions are correctly assigned to roles
- Ensure permission inheritance is working as expected
- Review the permission evaluation order and logic