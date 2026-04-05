type Project struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	Name          string    `json:"name"`
	RepoURL       string    `json:"repo_url"`
	BuildPreset   string    `json:"build_preset"`
	OutputDir     string    `json:"output_dir"`
	DefaultBranch string    `json:"default_branch"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}