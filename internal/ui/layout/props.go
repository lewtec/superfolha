package layout

// Chrome is shared by every HTML page.
type Chrome struct {
	Title    string
	Lang     string
	Flash    string
	Error    string
	LoggedIn bool
	Email    string
	T        func(string) string
}
