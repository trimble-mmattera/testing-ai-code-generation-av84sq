## WHY - Vision & Purpose

### Purpose & Users

Problem:  The product needs a document management platform to allow customers to upload, search, and download documents.

Users: Developers will interact with the document management platform via APIs.

Why they will use it: Customers require a tight integration of their documents to business processes that exist within the current application.

## WHAT - Core Requirements

### Functional Requirements

System must allow for uploading of documents.

System must isolate documents by tenant.

System must allow for searching documents by their content and metadata.

System must allow for downloading of single or multiple documents.

System must allow for users to create folders to store documents in.

System must list all documents and folders.

## HOW - Planning & Implementation

### Technical Implementation

**Required Stack Components**

- Backend:  Microservices written in Golang.  Implement Clean Architecture and DDD principles.

- Infrastructure:  Document storage must be in AWS S3.  Microservices should have docker containers that can be deployed to Kubernetes.

**System Requirements**

- Performance:  API SLA of under 2 seconds.

- Security: Documents must be encrypted at rest.  Document must be scanned for viruses upon upload.  System should be secure and all communication encrypted in transit.

- Scalability:  Must handle 10,000 uploads a day of documents averaging 3MB.

- Reliability: Must have 99.99% uptime.

- Integration constraints:  Must use AWS .

### Business Requirements

**Access & Authentication**

- User types: API users

- Authentication requirements: API JWT Authentication.  JWT should specify the tenant the user belongs to.

- Access control: Role based security.

**Business Rules**

- Data validation rules:  All fields are required and must be validated.

- Process requirements: If a file is found to be malicious in a security scan it should be quarantined into a folder in which the user can no longer access the file.

- Compliance needs:  Should adhere to SOC2 and ISO27001 standards.

- Service level expectations: SLAs for API requests are 2 seconds.  Maximum file processing time is 5 minutes.

### Implementation Priorities

- **High Priority:** 

- APIs for core business functions (upload, download, list, search)

- Security scanning for documents that are uploaded

- Role based security

- Tenant Isolation

- **Medium Priority:** 

- Thumbnails created for every file uploaded

- Webhooks for notifications.

- Events emitted for every domain event.

- Full set of unit and integration tests to validate business logic.

- **Lower Priority:** 

- Verbose logging for triage

- Open telemetry data