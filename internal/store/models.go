package store

type User struct {
	ID           string `db:"id" json:"id"`
	Name         string `db:"name" json:"name"`
	Email        string `db:"email" json:"email"`
	PasswordHash string `db:"password_hash" json:"-"`
	APIKey       string `db:"api_key" json:"api_key"`
	WebhookURL   string `db:"webhook_url" json:"webhook_url"`
	IsSuperuser  bool   `db:"is_superuser" json:"is_superuser"`
	Active       bool   `db:"active" json:"active"`
	CreatedAt    int64  `db:"created_at" json:"created_at"`
	UpdatedAt    int64  `db:"updated_at" json:"updated_at"`
}

type UserInput struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	APIKey       string
	WebhookURL   string
	IsSuperuser  bool
	Active       bool
}

type VerificationRecord struct {
	ID            string `db:"id" json:"id"`
	Email         string `db:"email" json:"email"`
	Status        string `db:"status" json:"status"`
	Message       string `db:"message" json:"message"`
	Source        string `db:"source" json:"source"`
	ProbeToken    string `db:"probe_token" json:"probe_token"`
	SMTPAccountID string `db:"smtp_account_id" json:"smtp_account_id"`
	UserID        string `db:"user_id" json:"user_id"`

	CheckCount int  `db:"check_count" json:"check_count"`
	Finalized  bool `db:"finalized" json:"finalized"`

	FirstCheckedAt int64 `db:"first_checked_at" json:"first_checked_at"`
	LastCheckedAt  int64 `db:"last_checked_at" json:"last_checked_at"`
	NextCheckAt    int64 `db:"next_check_at" json:"next_check_at"`
	CreatedAt      int64 `db:"created_at" json:"created_at"`
	UpdatedAt      int64 `db:"updated_at" json:"updated_at"`
}

type SMTPAccount struct {
	ID          string `db:"id" json:"id"`
	UserID      string `db:"user_id" json:"user_id"`
	Host        string `db:"host" json:"host"`
	Port        int    `db:"port" json:"port"`
	Username    string `db:"username" json:"username"`
	Password    string `db:"password" json:"-"`
	Sender      string `db:"sender" json:"sender"`
	IMAPHost    string `db:"imap_host" json:"imap_host"`
	IMAPPort    int    `db:"imap_port" json:"imap_port"`
	IMAPMailbox string `db:"imap_mailbox" json:"imap_mailbox"`
	DailyLimit  int    `db:"daily_limit" json:"daily_limit"`
	SentToday   int    `db:"sent_today" json:"sent_today"`
	ResetDate   string `db:"reset_date" json:"reset_date"`
	Active      bool   `db:"active" json:"active"`
	CreatedAt   int64  `db:"created_at" json:"created_at"`
	UpdatedAt   int64  `db:"updated_at" json:"updated_at"`
}

type SMTPAccountInput struct {
	ID          string
	UserID      string
	Host        string
	Port        int
	Username    string
	Password    string
	Sender      string
	IMAPHost    string
	IMAPPort    int
	IMAPMailbox string
	DailyLimit  int
	Active      bool
}

type EmailTemplate struct {
	ID              string `db:"id" json:"id"`
	UserID          string `db:"user_id" json:"user_id"`
	Name            string `db:"name" json:"name"`
	SubjectTemplate string `db:"subject_template" json:"subject_template"`
	BodyTemplate    string `db:"body_template" json:"body_template"`
	Active          bool   `db:"active" json:"active"`
	CreatedAt       int64  `db:"created_at" json:"created_at"`
	UpdatedAt       int64  `db:"updated_at" json:"updated_at"`
}

type EmailTemplateInput struct {
	ID              string
	UserID          string
	Name            string
	SubjectTemplate string
	BodyTemplate    string
	Active          bool
}
