package main

import (
	"time"
)

// MongoDB models matching Rust implementation exactly

type Coordinates struct {
	Lat float64 `bson:"lat"`
	Lng float64 `bson:"lng"`
}

type Address struct {
	Street      string      `bson:"street"`
	City        string      `bson:"city"`
	State       string      `bson:"state"`
	ZipCode     string      `bson:"zipCode"`
	Country     string      `bson:"country"`
	Coordinates Coordinates `bson:"coordinates"`
}

type Education struct {
	Institution string `bson:"institution"`
	Degree      string `bson:"degree"`
	Year        int32  `bson:"year"`
	Description string `bson:"description"`
}

type Experience struct {
	Company     string `bson:"company"`
	Position    string `bson:"position"`
	Duration    string `bson:"duration"`
	Description string `bson:"description"`
}

type Profile struct {
	Bio        string       `bson:"bio"`
	Interests  []string     `bson:"interests"`
	Skills     []string     `bson:"skills"`
	Education  []Education  `bson:"education"`
	Experience []Experience `bson:"experience"`
}

type NotificationSettings struct {
	Email bool `bson:"email"`
	Push  bool `bson:"push"`
	SMS   bool `bson:"sms"`
}

type Setting struct {
	Key       string    `bson:"key"`
	Value     string    `bson:"value"`
	Timestamp time.Time `bson:"timestamp"`
	Metadata  string    `bson:"metadata"`
}

type Preferences struct {
	Language      string               `bson:"language"`
	Timezone      string               `bson:"timezone"`
	Notifications NotificationSettings `bson:"notifications"`
	Settings      []Setting            `bson:"settings"`
}

type DataEntry struct {
	Key       string    `bson:"key"`
	Value     string    `bson:"value"`
	Timestamp time.Time `bson:"timestamp"`
	Metadata  string    `bson:"metadata"`
}

type Attachment struct {
	Name     string `bson:"name"`
	Size     int32  `bson:"size"`
	FileType string `bson:"type"`
}

type Message struct {
	ID          string       `bson:"id"`
	From        string       `bson:"from"`
	To          string       `bson:"to"`
	Subject     string       `bson:"subject"`
	Body        string       `bson:"body"`
	Timestamp   time.Time    `bson:"timestamp"`
	Read        bool         `bson:"read"`
	Attachments []Attachment `bson:"attachments"`
}

type Group struct {
	ID     string    `bson:"id"`
	Name   string    `bson:"name"`
	Joined time.Time `bson:"joined"`
}

type SocialMedia struct {
	Posts       []DataEntry `bson:"posts"`
	Connections []string    `bson:"connections"`
	Groups      []Group     `bson:"groups"`
}

type LargeData struct {
	Blob1 string `bson:"blob1"`
	Blob2 string `bson:"blob2"`
	Blob3 string `bson:"blob3"`
	Blob4 string `bson:"blob4"`
	Blob5 string `bson:"blob5"`
}

// User is the main MongoDB document (matches Rust User struct)
type User struct {
	ID          int64       `bson:"_id"`
	Username    string      `bson:"username"`
	Email       string      `bson:"email"`
	FirstName   string      `bson:"firstName"`
	LastName    string      `bson:"lastName"`
	Age         int32       `bson:"age"`
	CreatedAt   time.Time   `bson:"createdAt"`
	UpdatedAt   time.Time   `bson:"updatedAt"`
	Address     Address     `bson:"address"`
	Profile     Profile     `bson:"profile"`
	Preferences Preferences `bson:"preferences"`
	ActivityLog []DataEntry `bson:"activityLog"`
	Transactions []DataEntry `bson:"transactions"`
	Messages    []Message   `bson:"messages"`
	SocialMedia SocialMedia `bson:"socialMedia"`
	LargeData   LargeData   `bson:"largeData"`
}
