package api

type IndexData struct {
	Name      string
	Libraries []string
	ListData  struct {
		Completed int
		Watching  int
		Planning  int
		Dropped   int
		Paused    int
	}
}
