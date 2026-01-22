# Project-Specific Agent Instructions

This file provides context that helps AI agents work effectively in this repository.

## Repository Context
- **Language:** Python 3.11+
- **Framework:** FastAPI
- **Database:** PostgreSQL with SQLAlchemy ORM
- **Testing:** pytest with coverage requirements
- **Migration Tool:** Alembic

## Code Conventions

### API Endpoints
- Use RESTful patterns: GET, POST, PUT, DELETE
- Return JSON with snake_case field names
- Use HTTP status codes correctly (200, 201, 400, 401, 404, 409, etc.)
- Error responses: `{"error": "message", "code": "ERROR_CODE"}`

### Authentication
- JWT tokens via Bearer authentication
- Use `get_current_user()` dependency injection
- Token expiry: 24 hours
- Middleware: `auth_middleware.py`

### Database
- **Soft deletes:** Use `deleted_at` field, not hard deletes
- **Migrations:** Use Alembic for all schema changes
  ```bash
  alembic revision -m "description"
  alembic upgrade head
  ```
- **Transactions:** Use SQLAlchemy sessions with proper commit/rollback
- **Constraints:** Prefer database constraints over application validation

### Testing
- **Location:** All tests in `tests/` directory
- **Naming:** `test_<module>_<function>_<scenario>.py`
- **Coverage:** Minimum 90%, aim for 100% on new code
- **Run tests:** `pytest tests/ --cov=src --cov-report=term-missing`
- **Fixtures:** Reuse fixtures from `tests/conftest.py`

### Code Style
- **Formatter:** Black (line length: 100)
- **Linter:** Ruff
- **Type hints:** Required on all public functions
- **Docstrings:** Google style for public APIs

## Project Structure

```
project/
├── src/
│   ├── routes/          # API endpoints
│   ├── models/          # SQLAlchemy models
│   ├── schemas/         # Pydantic schemas
│   ├── services/        # Business logic
│   └── utils/           # Helper functions
├── tests/
│   ├── conftest.py      # Shared fixtures
│   └── test_*.py        # Test modules
├── migrations/          # Alembic migrations
└── alembic.ini          # Migration config
```

## Common Patterns

### Creating a New Endpoint

1. **Define the route:**
```python
from fastapi import APIRouter, Depends
from src.auth import get_current_user
from src.schemas import ProfileResponse

router = APIRouter(prefix="/api")

@router.get("/profile", response_model=ProfileResponse)
async def get_profile(current_user = Depends(get_current_user)):
    return ProfileResponse.from_orm(current_user)
```

2. **Add tests:**
```python
def test_get_profile_success(client, auth_headers):
    response = client.get("/api/profile", headers=auth_headers)
    assert response.status_code == 200
    assert "username" in response.json()
```

3. **Run validation:**
```bash
black src/ tests/
ruff check src/ tests/
pytest tests/ --cov=src
```

### Database Migrations

1. **Create migration:**
```bash
alembic revision -m "add unique constraint to username"
```

2. **Edit migration file:**
```python
def upgrade():
    # Add data cleanup if needed
    op.execute("UPDATE users SET username = username || '_' || id WHERE ...")
    
    # Add constraint
    op.create_unique_constraint('uq_users_username', 'users', ['username'])

def downgrade():
    op.drop_constraint('uq_users_username', 'users')
```

3. **Apply migration:**
```bash
alembic upgrade head
```

### Error Handling

```python
from fastapi import HTTPException

# Not found
raise HTTPException(status_code=404, detail="User not found")

# Validation error
raise HTTPException(status_code=400, detail="Invalid email format")

# Conflict (duplicate)
raise HTTPException(status_code=409, detail="Username already exists")

# Unauthorized
raise HTTPException(status_code=401, detail="Invalid credentials")
```

## Known Issues & Gotchas

### 1. Username Uniqueness
- ⚠️ **Issue:** Username field lacks UNIQUE constraint in database
- **Impact:** Race condition possible with concurrent updates
- **Fix:** Add UNIQUE constraint via migration (see iterations)

### 2. Test Database State
- ⚠️ **Issue:** Tests may leave data in database
- **Impact:** Migrations can fail on existing test data
- **Fix:** Use `pytest --create-db` to reset test database

### 3. Soft Deletes
- ℹ️ **Pattern:** All models use `deleted_at` timestamp
- **Queries:** Always filter `WHERE deleted_at IS NULL`
- **Helper:** Use `active_users()` query method

### 4. JWT Tokens
- ℹ️ **Storage:** Tokens stored in `Authorization: Bearer <token>` header
- **Validation:** Automatic via `get_current_user()` dependency
- **Testing:** Use `auth_headers` fixture for authenticated requests

## Ralph Loop Specific

### When Working on PRD Tasks

1. **Always read these files first:**
   - `prd.json` - Current status of all tasks
   - `progress.txt` - Previous iteration learnings
   - This `AGENTS.md` - Project conventions

2. **Before committing:**
   - ✅ Run full test suite
   - ✅ Check code formatting (black)
   - ✅ Check linting (ruff)
   - ✅ Verify all acceptance criteria met
   - ✅ Update prd.json with completion status
   - ✅ Append learnings to progress.txt

3. **If tests fail:**
   - ❌ DO NOT commit
   - 📝 Document failure in progress.txt:
     - What was attempted
     - Exact error messages
     - Insights for next iteration
   - 🔄 Update prd.json status to "in_progress"

4. **Task selection priority:**
   - Choose lowest-numbered incomplete user story
   - Within a story, work on acceptance criteria in order
   - Complete one story before starting the next

## Getting Help

### Running the Application
```bash
# Install dependencies
pip install -r requirements.txt

# Run migrations
alembic upgrade head

# Start server
uvicorn src.main:app --reload

# Run tests
pytest tests/ --cov=src
```

### Common Commands
```bash
# Format code
black src/ tests/

# Lint
ruff check src/ tests/

# Type check
mypy src/

# Generate coverage report
pytest --cov=src --cov-report=html
```

### Debug Tips
- Check logs in `logs/app.log`
- Use `breakpoint()` for debugging in pytest
- SQL queries logged at DEBUG level
- Use `PYTHONBREAKPOINT=0` to disable breakpoints in CI

---

**Last Updated:** 2026-01-22  
**Maintained by:** Ralph Loop workflow  
**Questions?** See main README or check progress.txt for recent learnings
