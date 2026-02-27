## Lotus Discipline API (Go)

Backend for the Lotus Discipline habit/phases system. Provides REST endpoints for mobile/web apps to drive the four phases: **Mud → Stem → Bloom → Thrive** and the animated lotus state.

### Tech stack

- **Language**: Go (Gin framework)
- **Database**: PostgreSQL
- **Auth (current)**: simple `userId` query parameter (JWT can be added later)

### Environment variables

- **`DATABASE_URL`**: PostgreSQL DSN, e.g.  
  `postgres://postgres:password@localhost:5432/lotus_local?sslmode=disable`
- **`PORT`** (optional): HTTP port (defaults to `8080`).

### Database setup (local)

1. Create a local database:

```bash
createdb lotus_local
psql lotus_local -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"
psql lotus_local -f db_schema.sql
```

2. Set `DATABASE_URL`:

```bash
# PowerShell
$env:DATABASE_URL = "postgres://postgres:password@localhost:5432/lotus_local?sslmode=disable"
```

### Run the API locally

```bash
go mod tidy
go run .
```

Server listens on `http://localhost:8080` by default.

### Core endpoints

- **POST `/api/register`**
  - **Body**:
    ```json
    {
      "email": "you@example.com",
      "habits": [
        { "name": "Lotus Sit", "goalMinutes": 2 },
        { "name": "Run", "goalMinutes": 20 },
        { "name": "Read", "goalMinutes": 30 }
      ]
    }
    ```
  - **Behavior**:
    - Upserts `users` by email.
    - Ensures a `user_programs` row with `start_date = CURRENT_DATE`.
    - Upserts each habit into `habits`, links via `user_habits`.
  - **Response**:
    ```json
    {
      "userId": "<uuid>",
      "message": "user registered"
    }
    ```

- **GET `/api/daily-check-in?userId=<uuid>`**
  - **Behavior**:
    - Loads the user's `start_date` and computes `dayInProgram`.
    - Derives `phase` (`mud`, `stem`, `bloom`, `thrive`) from the day.
    - Computes lotus state: `lotusStatus` (`seedling`, `sprout`, `bud`, `bloom`) and `growthPercent`.
    - Loads user habits and scales `currentMinutes` based on phase:
      - Mud: only `Lotus Sit` visible, 2 minutes.
      - Stem: ~10% of goal.
      - Bloom: ~60% of goal.
      - Thrive: 100% of goal.
    - Checks today's `habit_completions` to set checklist completion.
  - **Response (shape)**:
    ```json
    {
      "phase": "stem",
      "lotusStatus": "sprout",
      "growthPercent": 37,
      "dayInProgram": 15,
      "habits": [
        {
          "id": "<habit_uuid>",
          "name": "Run",
          "goalMinutes": 20,
          "phase": "stem",
          "currentMinutes": 2,
          "progress": 0.1
        }
      ],
      "checklist": [
        { "id": "lotus_sit", "description": "Complete your Lotus Sit", "completed": false },
        { "id": "all_habits", "description": "Complete all today's habits", "completed": false }
      ]
    }
    ```

- **POST `/api/complete-task?userId=<uuid>`**
  - **Body**:
    ```json
    { "habitId": "<habit_uuid>", "minutes": 5 }
    ```
  - **Behavior**:
    - Upserts a row in `habit_completions` for `(user_id, habit_id, today)` with `minutes`.
  - **Response**:
    ```json
    { "message": "task completion recorded" }
    ```

- **GET `/api/lotus-status?userId=<uuid>`**
  - **Behavior**:
    - Uses `start_date` and today's date to compute current phase and lotus growth.
  - **Response**:
    ```json
    {
      "phase": "bloom",
      "lotusStatus": "bud",
      "growthPercent": 62
    }
    ```

### Notes for the frontend (mobile/web)

- Use `/api/register` once to create the user and capture habits; store the returned `userId`.
- On each app open/day:
  - Call `/api/daily-check-in` to:
    - Render the lotus growth UI from `phase`, `lotusStatus`, `growthPercent`.
    - Render the daily habit list and checklist.
- When the user finishes a habit:
  - Call `/api/complete-task` and then refresh `/api/daily-check-in`.

