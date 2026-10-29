package model

import (
	"gorm.io/gorm"
	"std/migrations"
	"time"
)

// Base uses database-generated numeric IDs, exposed to clients as JSON numbers.
type Base struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type User struct {
	Base
	Username     string `gorm:"size:100" json:"username"`
	Admin        bool   `gorm:"not null;default:false" json:"admin"`
	PasswordHash []byte `gorm:"type:bytea" json:"-"`
	Name         string `gorm:"size:255;not null" json:"name"`
	Email        string `gorm:"size:255;uniqueIndex;not null" json:"email"`
	IsActive     bool   `gorm:"not null;default:true" json:"is_active"`
}
type Department struct {
	Base
	Name     string `gorm:"size:255;not null" json:"name"`
	Code     string `gorm:"size:50;uniqueIndex" json:"code"`
	ParentID *int64 `json:"parent_id"`
	IsActive bool   `gorm:"not null;default:true" json:"is_active"`
}
type DepartmentMember struct {
	Base
	DepartmentID int64  `gorm:"not null;uniqueIndex:uq_department_member"`
	UserID       int64  `gorm:"not null;uniqueIndex:uq_department_member"`
	Role         string `gorm:"size:20;not null"`
	IsActive     bool   `gorm:"not null;default:true"`
}
type DocumentType struct {
	Base
	Name           string `gorm:"size:100;not null" json:"name"`
	Code           string `gorm:"size:50;uniqueIndex" json:"code"`
	HasCorrelative bool   `gorm:"not null" json:"has_correlative"`
	IsActive       bool   `gorm:"not null;default:true" json:"is_active"`
}
type DocumentSequence struct {
	Base
	DocumentTypeID int64 `gorm:"not null"`
	DepartmentID   *int64
	Year           int `gorm:"not null"`
	LastNumber     int `gorm:"not null;default:0"`
}
type Document struct {
	Base
	DocumentTypeID int64 `gorm:"not null;index" json:"document_type_id"`
	// DocumentSequenceID is internal relational data. DocumentNumber is the
	// human-facing consecutive code returned by the API (for example STD-1-CIS).
	DocumentSequenceID   *int64  `gorm:"column:document_number" json:"document_sequence_id"`
	DocumentNumber       *string `gorm:"-" json:"document_number"`
	CorrelativeNumber    *int    `json:"correlative_number"`
	LegacyDocumentNumber *string `json:"legacy_document_number,omitempty"`
	CreatedBy            int64   `gorm:"not null" json:"created_by"`
	DepartmentID         *int64  `json:"department_id"`
	Name                 string  `gorm:"size:255;not null" json:"name"`
	Description          *string `gorm:"type:text" json:"description"`
	FileURL              *string `gorm:"type:text" json:"file_url"`
}
type Procedure struct {
	Base
	ProcedureNumber  *string    `gorm:"size:60" json:"procedure_number"`
	Name             string     `gorm:"size:255;not null" json:"name"`
	Description      *string    `gorm:"type:text" json:"description"`
	ProcedureType    string     `gorm:"size:12;not null" json:"procedure_type"`
	CreatedBy        int64      `gorm:"not null;index" json:"created_by"`
	SenderUserID     *int64     `json:"sender_user_id"`
	DepartmentID     *int64     `gorm:"column:sender_department_id;index" json:"sender_department_id"`
	Status           string     `gorm:"size:10;not null" json:"status"`
	ClosedAt         *time.Time `json:"closed_at"`
	ClosedBy         *int64     `json:"closed_by"`
	ClosingMessageID *int64     `json:"closing_message_id"`
	ClosingNote      *string    `json:"closing_note"`
}
type Message struct {
	Base
	CreatedBy    int64  `gorm:"column:created_by;not null" json:"created_by"`
	SenderUserID *int64 `json:"sender_user_id"`
	DepartmentID *int64 `gorm:"column:sender_department_id;index" json:"sender_department_id"`
	ProcedureID  *int64
	MessageType  string
	Visibility   string
	Status       string `gorm:"size:10;not null" json:"status"`
	Subject      string
	Body         string
	SentAt       *time.Time
}
type MessageRecipient struct {
	Base
	MessageID    int64
	RecipientID  *int64
	DepartmentID *int64
	Type         string
	Status       string
	ReadAt       *time.Time
}
type MessageRecipientRead struct {
	Base
	MessageRecipientID int64
	UserID             int64
	ReadAt             time.Time
}
type MessageDocument struct {
	Base
	MessageID  int64
	DocumentID int64
}
type MessageRelation struct {
	Base
	SourceMessageID int64
	TargetMessageID int64
	RelationType    string
	CreatedBy       int64
}
type AuditEvent struct {
	Base
	ActorID    int64
	EntityType string
	EntityID   int64
	Action     string
	Metadata   string
}

func Migrate(db *gorm.DB) error { return migrations.Apply(db) }
