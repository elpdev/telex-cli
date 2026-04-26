package contacts

import "time"

type ListParams struct {
	Page    int
	PerPage int
}

type ListContactsParams struct {
	ListParams
	ContactType string
	Query       string
	Sort        string
}

type Contact struct {
	ID                  int64                 `json:"id"`
	UserID              int64                 `json:"user_id"`
	ContactType         string                `json:"contact_type"`
	Name                string                `json:"name"`
	CompanyName         string                `json:"company_name"`
	Title               string                `json:"title"`
	Phone               string                `json:"phone"`
	Website             string                `json:"website"`
	DisplayName         string                `json:"display_name"`
	PrimaryEmailAddress string                `json:"primary_email_address"`
	EmailAddresses      []ContactEmailAddress `json:"email_addresses"`
	NoteFileID          *int64                `json:"note_file_id"`
	Note                *ContactNote          `json:"note"`
	Metadata            map[string]any        `json:"metadata"`
	CreatedAt           time.Time             `json:"created_at"`
	UpdatedAt           time.Time             `json:"updated_at"`
}

type ContactSummary struct {
	ID                  int64  `json:"id"`
	ContactType         string `json:"contact_type"`
	DisplayName         string `json:"display_name"`
	PrimaryEmailAddress string `json:"primary_email_address"`
}

type ContactEmailAddress struct {
	ID             int64     `json:"id"`
	EmailAddress   string    `json:"email_address"`
	Label          string    `json:"label"`
	PrimaryAddress bool      `json:"primary_address"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ContactNote struct {
	ContactID    int64      `json:"contact_id"`
	StoredFileID *int64     `json:"stored_file_id"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

type ContactCommunication struct {
	ID               int64          `json:"id"`
	ContactID        int64          `json:"contact_id"`
	Kind             string         `json:"kind"`
	CommunicableType string         `json:"communicable_type"`
	CommunicableID   int64          `json:"communicable_id"`
	OccurredAt       time.Time      `json:"occurred_at"`
	Metadata         map[string]any `json:"metadata"`
	Communication    map[string]any `json:"communication"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type ContactInput struct {
	ContactType    string
	Name           string
	CompanyName    string
	Title          string
	Phone          string
	Website        string
	EmailAddresses []ContactEmailAddressInput
	Metadata       map[string]any
}

type ContactEmailAddressInput struct {
	EmailAddress   string `json:"email_address"`
	Label          string `json:"label"`
	PrimaryAddress *bool  `json:"primary_address,omitempty"`
}

type ContactNoteInput struct {
	Title string
	Body  string
}
