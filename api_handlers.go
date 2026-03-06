package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func setupRouter() *gin.Engine {
	router := gin.Default()

	// Robust CORS to allow local cross-origin communication
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := router.Group("/api")
	{
		api.GET("/health", healthCheckHandler)
		api.POST("/register", registerHandler)
		api.POST("/login", loginHandler)

		protected := api.Group("/")
		protected.Use(authMiddleware())
		{
			protected.GET("/daily-check-in", dailyCheckinHandler)
			protected.POST("/complete-task", completeTaskHandler)
			// Only expose dev tools in non-release environments
			if gin.Mode() != gin.ReleaseMode {
				protected.POST("/dev/set-day", devSetDayHandler)
			}
		}
	}
	return router
}

func healthCheckHandler(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": "Database connection not initialized"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": "Database unreachable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func loginHandler(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
		return
	}

	ctx := c.Request.Context()
	var userID, passwordHash string
	err := db.QueryRow(ctx, "SELECT id, password_hash FROM users WHERE username = $1", req.Username).Scan(&userID, &passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		log.Printf("Database error during login: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB Error"})
		return
	}

	if !checkPasswordHash(req.Password, passwordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := generateToken(userID)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"userId": userID, "token": token})
}

func dailyCheckinHandler(c *gin.Context) {
	val, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No user ID"})
		return
	}
	userID := val.(string)

	// Try to get the data from Redis first
	cacheKey := "daily-check-in:" + userID
	cachedData, err := GetCache(cacheKey)
	if err == nil {
		var response DailyCheckinResponse
		if json.Unmarshal([]byte(cachedData), &response) == nil {
			c.JSON(http.StatusOK, response)
			return
		}
	}

	ctx := c.Request.Context()

	var startDate time.Time
	err = db.QueryRow(ctx, "SELECT start_date FROM user_programs WHERE user_id = $1", userID).Scan(&startDate)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Program not found"})
		return
	}

	day := int(time.Since(startDate).Hours()/24) + 1
	phase := determinePhase(day)
	status, pct := lotusStatusForPhase(phase, day)

	rows, err := db.Query(ctx, `
		SELECT h.id, h.name, h.goal_minutes, h.unit,
		EXISTS (SELECT 1 FROM habit_completions WHERE habit_id = h.id AND user_id = $1 AND completed_on = CURRENT_DATE)
		FROM habits h 
		JOIN user_habits uh ON uh.habit_id = h.id 
		WHERE uh.user_id = $1 ORDER BY h.name`, userID)
	if err != nil {
		log.Printf("Database error querying habits: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB Error"})
		return
	}
	defer rows.Close()

	var habits []Habit
	for rows.Next() {
		var h Habit
		var completed bool
		rows.Scan(&h.ID, &h.Name, &h.GoalMinutes, &h.Unit, &completed)
		if phase == PhaseMud && h.Name != "Lotus Sit" {
			continue
		}
		h.CurrentMins = scaledMinutes(h.GoalMinutes, phase)
		h.Completed = completed
		habits = append(habits, h)
	}

	response := DailyCheckinResponse{
		Phase: string(phase), LotusStatus: string(status), GrowthPct: pct, DayInProgram: day, Habits: habits,
	}

	// Cache the response in Redis for 5 minutes
	jsonData, err := json.Marshal(response)
	if err == nil {
		SetCache(cacheKey, jsonData, 5*time.Minute)
	}

	c.JSON(http.StatusOK, response)
}

func completeTaskHandler(c *gin.Context) {
	val, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID := val.(string)
	var req CompleteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	_, err := db.Exec(ctx,
		"INSERT INTO habit_completions (user_id, habit_id, minutes, completed_on) VALUES ($1, $2, $3, CURRENT_DATE) ON CONFLICT (user_id, habit_id, completed_on) DO UPDATE SET minutes = EXCLUDED.minutes",
		userID, req.HabitID, req.Minutes)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Record fail"})
		return
	}

	// Invalidate the cache for the user
	cacheKey := "daily-check-in:" + userID
	rdb.Del(ctx, cacheKey)

	c.JSON(http.StatusOK, gin.H{"message": "Logged"})
}

func registerHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	ctx := c.Request.Context()
	tx, err := db.Begin(ctx)
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB Error"})
		return
	}
	defer tx.Rollback(ctx)

	var userID string
	// Use ON CONFLICT DO NOTHING and check if a row was returned.
	// If no row is returned, the user already exists.
	err = tx.QueryRow(ctx, "INSERT INTO users (username, password_hash) VALUES ($1, $2) ON CONFLICT (username) DO NOTHING RETURNING id", req.Username, passwordHash).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// This means the user already exists
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
		log.Printf("Failed to insert user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB Error"})
		return
	}

	_, err = tx.Exec(ctx, "INSERT INTO user_programs (user_id, start_date) VALUES ($1, CURRENT_DATE) ON CONFLICT DO NOTHING", userID)
	if err != nil {
		log.Printf("Failed to insert user_programs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB Error"})
		return
	}

	for _, h := range req.Habits {
		var hID string
		err = tx.QueryRow(ctx, "INSERT INTO habits (name, goal_minutes, unit) VALUES ($1, $2, $3) ON CONFLICT (name) DO UPDATE SET goal_minutes=EXCLUDED.goal_minutes, unit=EXCLUDED.unit RETURNING id", h.Name, h.GoalMinutes, h.Unit).Scan(&hID)
		if err != nil {
			log.Printf("Failed to insert habit: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB Error"})
			return
		}
		_, err = tx.Exec(ctx, "INSERT INTO user_habits (user_id, habit_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", userID, hID)
		if err != nil {
			log.Printf("Failed to insert user_habit: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB Error"})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB Error"})
		return
	}

	token, err := generateToken(userID)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"userId": userID, "token": token})
}

func devSetDayHandler(c *gin.Context) {
	val, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No user ID"})
		return
	}
	userID := val.(string)

	var req struct {
		Day int `json:"day" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Calculate new start date: Today - (Day - 1) days
	newStartDate := time.Now().AddDate(0, 0, -(req.Day - 1))

	ctx := c.Request.Context()
	_, err := db.Exec(ctx, "UPDATE user_programs SET start_date = $1 WHERE user_id = $2", newStartDate, userID)
	if err != nil {
		log.Printf("Failed to update start date: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB Error"})
		return
	}

	// Invalidate cache
	cacheKey := "daily-check-in:" + userID
	if rdb != nil {
		rdb.Del(ctx, cacheKey)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Time travel successful",
		"day":     req.Day,
		"phase":   determinePhase(req.Day),
	})
}