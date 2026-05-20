package event

type Level string

const (
	LevelDebug   Level = "debug"
	LevelInfo    Level = "info"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
	LevelFatal   Level = "fatal"
)

func (l Level) Valid() bool {
	switch l {
	case LevelDebug, LevelInfo, LevelWarning, LevelError, LevelFatal:
		return true
	default:
		return false
	}
}
