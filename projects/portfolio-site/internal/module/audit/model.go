package audit

import "time"

// Log represents an audit log entry.
type Log struct {
	ID           int64     `json:"id"`
	ActorID      int64     `json:"actor_id,omitempty"`
	ActorName    string    `json:"actor_name,omitempty"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   int64     `json:"resource_id,omitempty"`
	Detail       string    `json:"detail,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// Action constants for audit logging.
const (
	ActionLogin          = "login"
	ActionLoginFailed    = "login_failed"
	ActionLogout         = "logout"
	ActionCreate         = "create"
	ActionUpdate         = "update"
	ActionDelete         = "delete"
	ActionPublish        = "publish"
	ActionUnpublish      = "unpublish"
	ActionArchive        = "archive"
	ActionUpload         = "upload"
	ActionView           = "view"
)

// ResourceType constants.
const (
	ResourceProject  = "project"
	ResourceArticle  = "article"
	ResourceAsset    = "asset"
	ResourceUser     = "user"
	ResourceCategory = "category"
	ResourceTag      = "tag"
	ResourceSetting  = "setting"
	ResourceContact  = "contact"
)

// Status constants.
const (
	StatusSuccess = "success"
	StatusFailure = "failure"
)

// Query is used for filtering audit logs.
type Query struct {
	Page         int
	PageSize     int
	ActorID      int64
	Action       string
	ResourceType string
	ResourceID   int64
	Status       string
	DateFrom     string
	DateTo       string
	OrderBy      string
}

// DefaultQuery returns sensible defaults for audit log queries.
func DefaultQuery() Query {
	return Query{
		Page:     1,
		PageSize: 50,
		OrderBy:  "created_at DESC",
	}
}
