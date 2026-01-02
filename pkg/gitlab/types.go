package gitlab

import "time"

// User represents a GitLab user.
type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	State     string `json:"state"`
	AvatarURL string `json:"avatar_url"`
	WebURL    string `json:"web_url"`
	Email     string `json:"email,omitempty"`
}

// Project represents a GitLab project.
type Project struct {
	ID                int        `json:"id"`
	Name              string     `json:"name"`
	NameWithNamespace string     `json:"name_with_namespace"`
	Path              string     `json:"path"`
	PathWithNamespace string     `json:"path_with_namespace"`
	Description       string     `json:"description"`
	DefaultBranch     string     `json:"default_branch"`
	Visibility        string     `json:"visibility"`
	WebURL            string     `json:"web_url"`
	SSHURLToRepo      string     `json:"ssh_url_to_repo"`
	HTTPURLToRepo     string     `json:"http_url_to_repo"`
	Archived          bool       `json:"archived"`
	CreatedAt         *time.Time `json:"created_at"`
	LastActivityAt    *time.Time `json:"last_activity_at"`
	Namespace         *Namespace `json:"namespace,omitempty"`
	Owner             *User      `json:"owner,omitempty"`
}

// Namespace represents a GitLab namespace.
type Namespace struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	FullPath string `json:"full_path"`
	WebURL   string `json:"web_url"`
}

// Issue represents a GitLab issue.
type Issue struct {
	ID          int        `json:"id"`
	IID         int        `json:"iid"`
	ProjectID   int        `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	State       string     `json:"state"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	ClosedBy    *User      `json:"closed_by,omitempty"`
	Labels      []string   `json:"labels"`
	Milestone   *Milestone `json:"milestone,omitempty"`
	Assignees   []User     `json:"assignees,omitempty"`
	Assignee    *User      `json:"assignee,omitempty"`
	Author      *User      `json:"author"`
	WebURL      string     `json:"web_url"`
	Weight      int        `json:"weight,omitempty"`
	Confidential bool      `json:"confidential"`
}

// MergeRequest represents a GitLab merge request.
type MergeRequest struct {
	ID              int        `json:"id"`
	IID             int        `json:"iid"`
	ProjectID       int        `json:"project_id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	State           string     `json:"state"`
	CreatedAt       *time.Time `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
	MergedAt        *time.Time `json:"merged_at,omitempty"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	SourceBranch    string     `json:"source_branch"`
	TargetBranch    string     `json:"target_branch"`
	SourceProjectID int        `json:"source_project_id"`
	TargetProjectID int        `json:"target_project_id"`
	Labels          []string   `json:"labels"`
	Milestone       *Milestone `json:"milestone,omitempty"`
	Assignees       []User     `json:"assignees,omitempty"`
	Assignee        *User      `json:"assignee,omitempty"`
	Author          *User      `json:"author"`
	MergedBy        *User      `json:"merged_by,omitempty"`
	MergeStatus     string     `json:"merge_status"`
	SHA             string     `json:"sha"`
	MergeCommitSHA  string     `json:"merge_commit_sha,omitempty"`
	Draft           bool       `json:"draft"`
	WorkInProgress  bool       `json:"work_in_progress"`
	WebURL          string     `json:"web_url"`
	DiffRefs        *DiffRefs  `json:"diff_refs,omitempty"`
}

// DiffRefs contains the refs for a merge request diff.
type DiffRefs struct {
	BaseSHA  string `json:"base_sha"`
	HeadSHA  string `json:"head_sha"`
	StartSHA string `json:"start_sha"`
}

// Label represents a GitLab label.
type Label struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	TextColor   string `json:"text_color"`
	Description string `json:"description"`
	Priority    int    `json:"priority,omitempty"`
	IsProjectLabel bool `json:"is_project_label"`
}

// Milestone represents a GitLab milestone.
type Milestone struct {
	ID          int        `json:"id"`
	IID         int        `json:"iid"`
	ProjectID   int        `json:"project_id,omitempty"`
	GroupID     int        `json:"group_id,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	State       string     `json:"state"`
	DueDate     string     `json:"due_date,omitempty"`
	StartDate   string     `json:"start_date,omitempty"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	WebURL      string     `json:"web_url"`
}

// Pipeline represents a GitLab CI/CD pipeline.
type Pipeline struct {
	ID        int        `json:"id"`
	IID       int        `json:"iid"`
	ProjectID int        `json:"project_id"`
	SHA       string     `json:"sha"`
	Ref       string     `json:"ref"`
	Status    string     `json:"status"`
	Source    string     `json:"source"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	WebURL    string     `json:"web_url"`
	User      *User      `json:"user,omitempty"`
}

// Job represents a GitLab CI/CD job.
type Job struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Stage      string     `json:"stage"`
	Status     string     `json:"status"`
	Ref        string     `json:"ref"`
	Tag        bool       `json:"tag"`
	Coverage   float64    `json:"coverage,omitempty"`
	CreatedAt  *time.Time `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Duration   float64    `json:"duration,omitempty"`
	User       *User      `json:"user,omitempty"`
	Pipeline   *Pipeline  `json:"pipeline,omitempty"`
	WebURL     string     `json:"web_url"`
}

// Commit represents a GitLab commit.
type Commit struct {
	ID             string     `json:"id"`
	ShortID        string     `json:"short_id"`
	Title          string     `json:"title"`
	Message        string     `json:"message"`
	AuthorName     string     `json:"author_name"`
	AuthorEmail    string     `json:"author_email"`
	AuthoredDate   *time.Time `json:"authored_date"`
	CommitterName  string     `json:"committer_name"`
	CommitterEmail string     `json:"committer_email"`
	CommittedDate  *time.Time `json:"committed_date"`
	CreatedAt      *time.Time `json:"created_at"`
	ParentIDs      []string   `json:"parent_ids"`
	WebURL         string     `json:"web_url"`
}

// Branch represents a GitLab branch.
type Branch struct {
	Name               string  `json:"name"`
	Merged             bool    `json:"merged"`
	Protected          bool    `json:"protected"`
	Default            bool    `json:"default"`
	DevelopersCanPush  bool    `json:"developers_can_push"`
	DevelopersCanMerge bool    `json:"developers_can_merge"`
	CanPush            bool    `json:"can_push"`
	WebURL             string  `json:"web_url"`
	Commit             *Commit `json:"commit,omitempty"`
}

// Tag represents a GitLab tag.
type Tag struct {
	Name      string   `json:"name"`
	Message   string   `json:"message"`
	Target    string   `json:"target"`
	Commit    *Commit  `json:"commit,omitempty"`
	Release   *Release `json:"release,omitempty"`
	Protected bool     `json:"protected"`
}

// Release represents a GitLab release.
type Release struct {
	TagName     string     `json:"tag_name"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	CreatedAt   *time.Time `json:"created_at"`
	ReleasedAt  *time.Time `json:"released_at"`
	Author      *User      `json:"author,omitempty"`
}

// Note represents a GitLab note (comment).
type Note struct {
	ID         int        `json:"id"`
	Body       string     `json:"body"`
	Author     *User      `json:"author"`
	CreatedAt  *time.Time `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
	System     bool       `json:"system"`
	NoteableID int        `json:"noteable_id"`
	NoteableType string   `json:"noteable_type"`
	Resolvable bool       `json:"resolvable"`
	Resolved   bool       `json:"resolved,omitempty"`
	ResolvedBy *User      `json:"resolved_by,omitempty"`
}

// Diff represents a file diff.
type Diff struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	AMode       string `json:"a_mode"`
	BMode       string `json:"b_mode"`
	Diff        string `json:"diff"`
	NewFile     bool   `json:"new_file"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
}

// Group represents a GitLab group.
type Group struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	FullName    string `json:"full_name"`
	FullPath    string `json:"full_path"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	WebURL      string `json:"web_url"`
	ParentID    int    `json:"parent_id,omitempty"`
}

// FileInfo represents information about a file in a repository.
type FileInfo struct {
	FileName     string `json:"file_name"`
	FilePath     string `json:"file_path"`
	Size         int    `json:"size"`
	Encoding     string `json:"encoding"`
	Content      string `json:"content"`
	ContentSHA256 string `json:"content_sha256"`
	Ref          string `json:"ref"`
	BlobID       string `json:"blob_id"`
	CommitID     string `json:"commit_id"`
	LastCommitID string `json:"last_commit_id"`
}

// TreeNode represents a node in the repository tree.
type TreeNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
	Mode string `json:"mode"`
}

// PaginationInfo contains pagination information from API responses.
type PaginationInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
	NextPage   int `json:"next_page,omitempty"`
	PrevPage   int `json:"prev_page,omitempty"`
}
