package api

type Request struct {
	ID        int64
	Prompt    string
	Status    string
	Response  string
	CreatedAt string
}

type OutputLine struct {
	ID        int64
	RequestID int64
	LineNum   int
	LineType  string
	Content   string
	CreatedAt string
}
