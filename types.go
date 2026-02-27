package main


// Core domain types and phase/lotus logic.

type Phase string

const (
	PhaseMud    Phase = "mud"
	PhaseStem   Phase = "stem"
	PhaseBloom  Phase = "bloom"
	PhaseThrive Phase = "thrive"
)

type LotusStatus string

const (
	LotusSeedling LotusStatus = "seedling"
	LotusSprout   LotusStatus = "sprout"
	LotusBud      LotusStatus = "bud"
	LotusBloom    LotusStatus = "bloom"
)

type Habit struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	GoalMinutes int     `json:"goalMinutes"`
	Unit        string  `json:"unit"`
	Phase       Phase   `json:"phase"`
	CurrentMins int     `json:"currentMinutes"`
	Progress    float64 `json:"progress"`
}

type RegisterRequest struct {
	Email  string       `json:"email" binding:"required,email"`
	Habits []HabitInput `json:"habits" binding:"required,min=1"`
}

type HabitInput struct {
	Name        string `json:"name" binding:"required"`
	GoalMinutes int    `json:"goalMinutes" binding:"required,min=1"`
	// Unit for the habit goal: "minutes", "pages", "reps", etc.
	Unit string `json:"unit"`
}

type DailyCheckinResponse struct {
	Phase        Phase          `json:"phase"`
	LotusStatus  LotusStatus    `json:"lotusStatus"`
	GrowthPct    int            `json:"growthPercent"`
	DayInProgram int            `json:"dayInProgram"`
	Habits       []Habit        `json:"habits"`
	Checklist    []ChecklistItem `json:"checklist"`
}

type ChecklistItem struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

type CompleteTaskRequest struct {
	HabitID string `json:"habitId" binding:"required"`
	Minutes int    `json:"minutes" binding:"required,min=1"`
}

type LotusStatusResponse struct {
	Phase       Phase       `json:"phase"`
	LotusStatus LotusStatus `json:"lotusStatus"`
	GrowthPct   int         `json:"growthPercent"`
}

// Phase timing and lotus growth helpers.

func determinePhase(day int) Phase {
	switch {
	case day < 14:
		return PhaseMud
	case day < 28:
		return PhaseStem
	case day < 42:
		return PhaseBloom
	default:
		return PhaseThrive
	}
}

func lotusStatusForPhase(p Phase, day int) (LotusStatus, int) {
	switch p {
	case PhaseMud:
		// 0–25%
		return LotusSeedling, min(25, (day*100)/14)
	case PhaseStem:
		// 25–50%
		return LotusSprout, 25 + min(25, ((day-14)*100)/14)
	case PhaseBloom:
		// 50–75%
		return LotusBud, 50 + min(25, ((day-28)*100)/14)
	case PhaseThrive:
		return LotusBloom, 100
	default:
		return LotusSeedling, 0
	}
}

func scaledMinutes(goal int, p Phase) int {
	switch p {
	case PhaseMud:
		if goal < 2 {
			return goal
		}
		return 2
	case PhaseStem:
		return max(1, int(float64(goal)*0.1))
	case PhaseBloom:
		return max(1, int(float64(goal)*0.6))
	case PhaseThrive:
		return goal
	default:
		return goal
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

