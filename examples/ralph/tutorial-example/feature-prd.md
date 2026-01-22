# Feature: User Profile Management

## Overview
Implement a complete user profile management system allowing users to view and update their profile information.

## User Stories

### US-001: View User Profile
**As a** registered user  
**I want to** view my profile information  
**So that** I can see my current details

**Acceptance Criteria:**
- [ ] GET /api/profile endpoint returns user profile
- [ ] Response includes: username, email, created_at, avatar_url
- [ ] Returns 401 if not authenticated
- [ ] Returns 404 if user not found
- [ ] Unit tests pass with 100% coverage

**Estimate:** 30 minutes

### US-002: Update User Profile
**As a** registered user  
**I want to** update my profile information  
**So that** I can keep my details current

**Acceptance Criteria:**
- [ ] PUT /api/profile endpoint updates user profile
- [ ] Can update: username, email, avatar_url
- [ ] Validates email format
- [ ] Username must be unique (returns 409 if duplicate)
- [ ] Returns updated profile on success
- [ ] Returns 400 for invalid input
- [ ] Unit tests pass with 100% coverage

**Estimate:** 45 minutes

### US-003: Upload Profile Avatar
**As a** registered user  
**I want to** upload a profile picture  
**So that** I can personalize my account

**Acceptance Criteria:**
- [ ] POST /api/profile/avatar endpoint accepts image upload
- [ ] Supports JPEG, PNG formats (max 2MB)
- [ ] Resizes image to 200x200px
- [ ] Stores image in cloud storage (S3-compatible)
- [ ] Updates avatar_url in user profile
- [ ] Returns 413 for files > 2MB
- [ ] Returns 415 for unsupported formats
- [ ] Unit tests pass with 100% coverage

**Estimate:** 60 minutes

## Technical Requirements

### API Design
- RESTful endpoints following existing patterns
- JWT authentication required for all endpoints
- Standard error responses (400, 401, 404, 409, 413, 415)
- JSON request/response bodies

### Data Model
```javascript
{
  "id": "uuid",
  "username": "string (3-50 chars, alphanumeric + underscore)",
  "email": "string (valid email format)",
  "avatar_url": "string (URL or null)",
  "created_at": "ISO 8601 timestamp",
  "updated_at": "ISO 8601 timestamp"
}
```

### Testing Requirements
- Unit tests for all endpoints
- Test success cases
- Test all error conditions
- Mock external services (storage)
- 100% code coverage on new code

### Security Considerations
- No SQL injection vulnerabilities
- Input sanitization on all fields
- Rate limiting on upload endpoints (5 uploads/hour)
- Validate file types by content, not extension
- Prevent path traversal attacks

## Definition of Done
- [ ] All acceptance criteria met
- [ ] All unit tests passing
- [ ] Code review completed
- [ ] API documentation updated
- [ ] No security vulnerabilities
- [ ] Integration with existing auth system verified
