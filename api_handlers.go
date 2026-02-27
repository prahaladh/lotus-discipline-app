package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// setupRouter creates the Gin engine and registers all API routes.
func setupRouter() *gin.Engine {
	router := gin.Default()

	// CORS: allow all origins for development.
	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	api := router.Group("/api")
	{
		api.POST("/register", registerHandler)
		api.GET("/daily-check-in", dailyCheckinHandler)
		api.POST("/complete-task", completeTaskHandler)
		api.GET("/lotus-status", lotusStatusHandler)
	}

	return router
}

// HTTP handlers.

func registerHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()
	tx, err := db.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin transaction"})
		return
	}
	defer tx.Rollback(ctx)

	var userID string
	if err := tx.QueryRow(ctx,
		"INSERT INTO users (email) VALUES ($1) ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email RETURNING id",
		req.Email,
	).Scan(&userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upsert user"})
		return
	}

	// Ensure user has a program start date.
	if _, err := tx.Exec(ctx,
		"INSERT INTO user_programs (user_id, start_date) VALUES ($1, CURRENT_DATE) ON CONFLICT (user_id) DO NOTHING",
		userID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user program"})
		return
	}

	// Upsert habits and link them to the user.
	for _, h := range req.Habits {
		unit := h.Unit
		if unit == "" {
			unit = "minutes"
		}

		var habitID string
		if err := tx.QueryRow(ctx,
			"INSERT INTO habits (name, goal_minutes, unit) VALUES ($1, $2, $3) "+
				"ON CONFLICT (name) DO UPDATE SET goal_minutes = EXCLUDED.goal_minutes, unit = EXCLUDED.unit RETURNING id",
			h.Name,
			h.GoalMinutes,
			unit,
		).Scan(&habitID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upsert habit"})
			return
		}

		if _, err := tx.Exec(ctx,
			"INSERT INTO user_habits (user_id, habit_id) VALUES ($1, $2) ON CONFLICT (user_id, habit_id) DO NOTHING",
			userID,
			habitID,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to link user habit"})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	// For now, return the userId so the mobile app can use it as a query param.
	c.JSON(http.StatusCreated, gin.H{
		"userId":  userID,
		"message": "user registered",
	})
}

func dailyCheckinHandler(c *gin.Context) {
	userID := c.Query("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId query parameter is required"})
		return
	}

	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not configured"})
		return
	}

	ctx := c.Request.Context()

	// Get the user's program start date.
	var startDate time.Time
	if err := db.QueryRow(ctx,
		"SELECT start_date FROM user_programs WHERE user_id = $1",
		userID,
	).Scan(&startDate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user program not found"})
		return
	}

	// Calculate day in program.
	today := time.Now().Truncate(24 * time.Hour)
	start := startDate.Truncate(24 * time.Hour)
	diff := today.Sub(start)
	day := int(diff.Hours()/24) + 1

	phase := determinePhase(day)
	lotus, pct := lotusStatusForPhase(phase, day)

	// Load user's habits from DB.
	rows, err := db.Query(ctx,
		`SELECT h.id, h.name, h.goal_minutes, h.unit
         FROM habits h
         JOIN user_habits uh ON uh.habit_id = h.id
         WHERE uh.user_id = $1
         ORDER BY h.name`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load habits"})
		return
	}
	defer rows.Close()

	var rawHabits []Habit
	for rows.Next() {
		var h Habit
		if err := rows.Scan(&h.ID, &h.Name, &h.GoalMinutes, &h.Unit); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan habit"})
			return
		}
		rawHabits = append(rawHabits, h)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read habits"})
		return
	}

	// Load today's completions to mark checklist items.
	compRows, err := db.Query(ctx,
		`SELECT habit_id
         FROM habit_completions
         WHERE user_id = $1 AND completed_on = CURRENT_DATE`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load completions"})
		return
	}
	defer compRows.Close()

	completedHabits := make(map[string]bool)
	for compRows.Next() {
		var hid string
		if err := compRows.Scan(&hid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan completion"})
			return
		}
		completedHabits[hid] = true
	}
	if err := compRows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read completions"})
		return
	}

	var activeHabits []Habit
	var lotusHabitID string
	for _, h := range rawHabits {
		mins := scaledMinutes(h.GoalMinutes, phase)

		// Hide non-lotus habits during Mud phase as per spec.
		if phase == PhaseMud && h.Name != "Lotus Sit" {
			continue
		}

		progress := float64(mins) / float64(h.GoalMinutes)
		if h.Name == "Lotus Sit" {
			lotusHabitID = h.ID
		}

		// Expose the real DB id so the client can send it back
		// as habitId when calling /complete-task.
		activeHabits = append(activeHabits, Habit{
			ID:          h.ID,
			Name:        h.Name,
			GoalMinutes: h.GoalMinutes,
			Phase:       phase,
			CurrentMins: mins,
			Progress:    progress,
		})
	}

	// Build checklist with completion status.
	lotusCompleted := lotusHabitID != "" && completedHabits[lotusHabitID]
	allCompleted := len(activeHabits) > 0 && len(completedHabits) == len(activeHabits)

	checklist := []ChecklistItem{
		{ID: "lotus_sit", Description: "Complete your Lotus Sit", Completed: lotusCompleted},
		{ID: "all_habits", Description: "Complete all today's habits", Completed: allCompleted},
	}

	resp := DailyCheckinResponse{
		Phase:        phase,
		LotusStatus:  lotus,
		GrowthPct:    pct,
		DayInProgram: day,
		Habits:       activeHabits,
		Checklist:    checklist,
	}

	c.JSON(http.StatusOK, resp)
}

func completeTaskHandler(c *gin.Context) {
	userID := c.Query("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId query parameter is required"})
		return
	}

	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not configured"})
		return
	}

	var req CompleteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Record or update today's completion for this habit.
	if _, err := db.Exec(ctx,
		`INSERT INTO habit_completions (user_id, habit_id, minutes, completed_on)
         VALUES ($1, $2, $3, CURRENT_DATE)
         ON CONFLICT (user_id, habit_id, completed_on)
         DO UPDATE SET minutes = EXCLUDED.minutes`,
		userID,
		req.HabitID,
		req.Minutes,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record completion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "task completion recorded",
	})
}

func lotusStatusHandler(c *gin.Context) {
	userID := c.Query("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId query parameter is required"})
		return
	}

	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not configured"})
		return
	}

	ctx := context.Background()

	var startDate time.Time
	if err := db.QueryRow(ctx,
		"SELECT start_date FROM user_programs WHERE user_id = $1",
		userID,
	).Scan(&startDate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user program not found"})
		return
	}

	today := time.Now().Truncate(24 * time.Hour)
	start := startDate.Truncate(24 * time.Hour)
	diff := today.Sub(start)
	day := int(diff.Hours()/24) + 1

	phase := determinePhase(day)
	lotus, pct := lotusStatusForPhase(phase, day)

	resp := LotusStatusResponse{
		Phase:       phase,
		LotusStatus: lotus,
		GrowthPct:   pct,
	}

	c.JSON(http.StatusOK, resp)
}

