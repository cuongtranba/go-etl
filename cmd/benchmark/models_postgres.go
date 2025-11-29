package main

import (
	"time"

	"gorm.io/datatypes"
)

// PostgreSQL models matching Rust SeaORM implementation (15 tables)

type PGUser struct {
	ID        int64     `gorm:"primaryKey"`
	Username  string    `gorm:"type:varchar(255);unique;not null"`
	Email     string    `gorm:"type:varchar(255);unique;not null"`
	FirstName string    `gorm:"type:varchar(255);not null"`
	LastName  string    `gorm:"type:varchar(255);not null"`
	Age       int32     `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (PGUser) TableName() string {
	return "users"
}

type PGAddress struct {
	ID          int64           `gorm:"primaryKey"`
	UserID      int64           `gorm:"not null;index"`
	Street      string          `gorm:"type:text;not null"`
	City        string          `gorm:"type:varchar(255);not null"`
	State       string          `gorm:"type:varchar(255);not null"`
	ZipCode     string          `gorm:"type:varchar(50);not null"`
	Country     string          `gorm:"type:varchar(255);not null"`
	Coordinates datatypes.JSON  `gorm:"type:jsonb"`
}

func (PGAddress) TableName() string {
	return "addresses"
}

type PGProfile struct {
	ID        int64          `gorm:"primaryKey"`
	UserID    int64          `gorm:"not null;index"`
	Bio       string         `gorm:"type:text"`
	Interests datatypes.JSON `gorm:"type:jsonb"`
	Skills    datatypes.JSON `gorm:"type:jsonb"`
}

func (PGProfile) TableName() string {
	return "profiles"
}

type PGEducation struct {
	ID          int64  `gorm:"primaryKey"`
	ProfileID   int64  `gorm:"not null;index"`
	Institution string `gorm:"type:text;not null"`
	Degree      string `gorm:"type:text;not null"`
	Year        int32  `gorm:"not null"`
	Description string `gorm:"type:text"`
}

func (PGEducation) TableName() string {
	return "education"
}

type PGExperience struct {
	ID          int64  `gorm:"primaryKey"`
	ProfileID   int64  `gorm:"not null;index"`
	Company     string `gorm:"type:text;not null"`
	Position    string `gorm:"type:text;not null"`
	Duration    string `gorm:"type:text;not null"`
	Description string `gorm:"type:text"`
}

func (PGExperience) TableName() string {
	return "experience"
}

type PGPreferences struct {
	ID            int64          `gorm:"primaryKey"`
	UserID        int64          `gorm:"not null;index"`
	Language      string         `gorm:"type:varchar(50);not null"`
	Timezone      string         `gorm:"type:varchar(100);not null"`
	Notifications datatypes.JSON `gorm:"type:jsonb"`
}

func (PGPreferences) TableName() string {
	return "preferences"
}

type PGSettings struct {
	ID           int64     `gorm:"primaryKey"`
	PreferenceID int64     `gorm:"not null;index"`
	Key          string    `gorm:"type:varchar(255);not null"`
	Value        string    `gorm:"type:text;not null"`
	Timestamp    time.Time `gorm:"not null"`
	Metadata     string    `gorm:"type:text"`
}

func (PGSettings) TableName() string {
	return "settings"
}

type PGActivityLog struct {
	ID        int64     `gorm:"primaryKey"`
	UserID    int64     `gorm:"not null;index"`
	Key       string    `gorm:"type:varchar(255);not null"`
	Value     string    `gorm:"type:text;not null"`
	Timestamp time.Time `gorm:"not null"`
	Metadata  string    `gorm:"type:text"`
}

func (PGActivityLog) TableName() string {
	return "activity_log"
}

type PGTransactions struct {
	ID        int64     `gorm:"primaryKey"`
	UserID    int64     `gorm:"not null;index"`
	Key       string    `gorm:"type:varchar(255);not null"`
	Value     string    `gorm:"type:text;not null"`
	Timestamp time.Time `gorm:"not null"`
	Metadata  string    `gorm:"type:text"`
}

func (PGTransactions) TableName() string {
	return "transactions"
}

type PGMessages struct {
	ID        string    `gorm:"primaryKey;type:varchar(255)"`
	UserID    int64     `gorm:"not null;index"`
	From      string    `gorm:"type:varchar(255);not null"`
	To        string    `gorm:"type:varchar(255);not null"`
	Subject   string    `gorm:"type:text;not null"`
	Body      string    `gorm:"type:text;not null"`
	Timestamp time.Time `gorm:"not null"`
	Read      bool      `gorm:"not null;default:false"`
}

func (PGMessages) TableName() string {
	return "messages"
}

type PGAttachments struct {
	ID        int64  `gorm:"primaryKey"`
	MessageID string `gorm:"type:varchar(255);not null;index"`
	Name      string `gorm:"type:varchar(255);not null"`
	Size      int32  `gorm:"not null"`
	FileType  string `gorm:"type:varchar(100);not null"`
}

func (PGAttachments) TableName() string {
	return "attachments"
}

type PGSocialMedia struct {
	ID          int64          `gorm:"primaryKey"`
	UserID      int64          `gorm:"not null;index"`
	Connections datatypes.JSON `gorm:"type:jsonb"`
}

func (PGSocialMedia) TableName() string {
	return "social_media"
}

type PGPosts struct {
	ID            int64     `gorm:"primaryKey"`
	SocialMediaID int64     `gorm:"not null;index"`
	Key           string    `gorm:"type:varchar(255);not null"`
	Value         string    `gorm:"type:text;not null"`
	Timestamp     time.Time `gorm:"not null"`
	Metadata      string    `gorm:"type:text"`
}

func (PGPosts) TableName() string {
	return "posts"
}

type PGGroups struct {
	ID            string    `gorm:"primaryKey;type:varchar(255)"`
	SocialMediaID int64     `gorm:"not null;index"`
	Name          string    `gorm:"type:varchar(255);not null"`
	Joined        time.Time `gorm:"not null"`
}

func (PGGroups) TableName() string {
	return "groups"
}

type PGLargeData struct {
	ID     int64  `gorm:"primaryKey"`
	UserID int64  `gorm:"not null;index"`
	Blob1  string `gorm:"type:text"`
	Blob2  string `gorm:"type:text"`
	Blob3  string `gorm:"type:text"`
	Blob4  string `gorm:"type:text"`
	Blob5  string `gorm:"type:text"`
}

func (PGLargeData) TableName() string {
	return "large_data"
}

// AutoMigrate creates all tables
func AutoMigrateAll(db interface{ AutoMigrate(...interface{}) error }) error {
	return db.AutoMigrate(
		&PGUser{},
		&PGAddress{},
		&PGProfile{},
		&PGEducation{},
		&PGExperience{},
		&PGPreferences{},
		&PGSettings{},
		&PGActivityLog{},
		&PGTransactions{},
		&PGMessages{},
		&PGAttachments{},
		&PGSocialMedia{},
		&PGPosts{},
		&PGGroups{},
		&PGLargeData{},
	)
}
