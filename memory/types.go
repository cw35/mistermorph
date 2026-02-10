package memory

import "time"

type RequestContext string

const (
	ContextUnknown RequestContext = "unknown"
	ContextPublic  RequestContext = "public"
	ContextPrivate RequestContext = "private"
)

type Frontmatter struct {
	CreatedAt        string     `yaml:"created_at"`
	UpdatedAt        string     `yaml:"updated_at"`
	Summary          string     `yaml:"summary"`
	SessionID        string     `yaml:"session_id,omitempty"`
	Tags             []string   `yaml:"tags,omitempty"`
	ContactIDs       StringList `yaml:"contact_id,omitempty"`
	ContactNicknames StringList `yaml:"contact_nickname,omitempty"`
}

type Manager struct {
	Dir           string
	ShortTermDays int
	Now           func() time.Time
}

type KVItem struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

type PromoteDraft struct {
	GoalsProjects []KVItem `json:"goals_projects"`
	KeyFacts      []KVItem `json:"key_facts"`
}

type SessionDraft struct {
	Summary        string       `json:"summary"`
	SessionSummary []KVItem     `json:"session_summary"`
	TemporaryFacts []KVItem     `json:"temporary_facts"`
	Promote        PromoteDraft `json:"promote"`
}

type ShortTermSummary struct {
	Date    string
	Summary string
	RelPath string
}

// ShortTermContent is the parsed representation of a short-term session file.
type ShortTermContent struct {
	SessionSummary []KVItem
	TemporaryFacts []KVItem
	RelatedLinks   []LinkItem
}

type LongTermContent struct {
	Goals []KVItem
	Facts []KVItem
}

type LinkItem struct {
	Text   string
	Target string
}
