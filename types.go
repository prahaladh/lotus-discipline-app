package main

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

type HabitInput struct {
	Name        string `json:"name" binding:"required"`
	GoalMinutes int    `json:"goalMinutes" binding:"required,min=1"`
	Unit        string `json:"unit"`
}

type RegisterRequest struct {
	Username string       `json:"username" binding:"required"`
	Password string       `json:"password" binding:"required"`
	Habits   []HabitInput `json:"habits" binding:"required,min=1"`
}

type CompleteTaskRequest struct {
	HabitID string `json:"habitId" binding:"required"`
	Minutes int    `json:"minutes" binding:"required,min=1"`
}

type DailyCheckinResponse struct {
	Phase        string  `json:"phase"`
	LotusStatus  string  `json:"lotusStatus"`
	GrowthPct    int     `json:"growthPercent"`
	DayInProgram int     `json:"dayInProgram"`
	Habits       []Habit `json:"habits"`
}

type Habit struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	GoalMinutes int    `json:"goalMinutes"`
	Unit        string `json:"unit"`
	CurrentMins int    `json:"currentMinutes"`
	Completed   bool   `json:"completed"`
}

func determinePhase(day int) Phase {
	if day < 14 { return PhaseMud }
	if day < 28 { return PhaseStem }
	if day < 42 { return PhaseBloom }
	return PhaseThrive
}

func scaledMinutes(goal int, p Phase) int {
	switch p {
	case PhaseMud: return 2
	case PhaseStem: return max(1, int(float64(goal)*0.1))
	case PhaseBloom: return max(1, int(float64(goal)*0.6))
	default: return goal
	}
}

func lotusStatusForPhase(p Phase, day int) (LotusStatus, int) {
	switch p {
	case PhaseMud: return LotusSeedling, (day * 100) / 14
	case PhaseStem: return LotusSprout, 25 + ((day - 14) * 100) / 14
	case PhaseBloom: return LotusBud, 50 + ((day - 28) * 100) / 14
	default: return LotusBloom, 100
	}
}

func max(a, b int) int { if a > b { return a }; return b }