package screens

// NavigateMsg is sent when navigating to a different screen
type NavigateMsg struct {
	Screen string // "menu", "config", "stats", "duplicates", "backup", "quit"
}

// ErrorMsg is sent when an error occurs
type ErrorMsg struct {
	Err error
}

// SuccessMsg is sent when an operation succeeds
type SuccessMsg struct {
	Message string
}
