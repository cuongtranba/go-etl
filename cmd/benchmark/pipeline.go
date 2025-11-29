package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cuong/go-etl/pkg/etl"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

// TransformedUser holds all transformed data for one user (15 tables)
type TransformedUser struct {
	User        PGUser
	Address     PGAddress
	Profile     PGProfile
	Education   []PGEducation
	Experience  []PGExperience
	Preferences PGPreferences
	Settings    []PGSettings
	ActivityLog []PGActivityLog
	Transactions []PGTransactions
	Messages    []PGMessages
	Attachments  []PGAttachments
	SocialMedia PGSocialMedia
	Posts       []PGPosts
	Groups      []PGGroups
	LargeData   PGLargeData
}

// UserETL implements ETLProcessor for User migration
type UserETL struct {
	mongoClient *mongo.Client
	postgresDB  *gorm.DB
}

// NewUserETL creates a new User ETL processor
func NewUserETL(mongoClient *mongo.Client, postgresDB *gorm.DB) *UserETL {
	return &UserETL{
		mongoClient: mongoClient,
		postgresDB:  postgresDB,
	}
}

// PreProcess runs migrations
func (u *UserETL) PreProcess(ctx context.Context) error {
	fmt.Println("Starting ETL pipeline...")
	return AutoMigrateAll(u.postgresDB)
}

// Extract reads users from MongoDB
func (u *UserETL) Extract(ctx context.Context) (<-chan etl.Payload[User], error) {
	ch := make(chan etl.Payload[User], 100)

	collection := u.mongoClient.Database("sample_db").Collection("users")

	go func() {
		defer close(ch)

		cursor, err := collection.Find(ctx, bson.M{})
		if err != nil {
			ch <- etl.Payload[User]{Err: fmt.Errorf("failed to create cursor: %w", err)}
			return
		}
		defer cursor.Close(ctx)

		for cursor.Next(ctx) {
			var user User
			if err := cursor.Decode(&user); err != nil {
				ch <- etl.Payload[User]{Err: fmt.Errorf("failed to decode user: %w", err)}
				return
			}

			select {
			case <-ctx.Done():
				return
			case ch <- etl.Payload[User]{Data: user}:
			}
		}

		if err := cursor.Err(); err != nil {
			ch <- etl.Payload[User]{Err: fmt.Errorf("cursor error: %w", err)}
		}
	}()

	return ch, nil
}

// Transform converts MongoDB User to PostgreSQL models
func (u *UserETL) Transform(ctx context.Context, user User) TransformedUser {
	// Main user table
	pgUser := PGUser{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Age:       user.Age,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	// Address table
	coordsJSON, _ := json.Marshal(map[string]interface{}{
		"lat": user.Address.Coordinates.Lat,
		"lng": user.Address.Coordinates.Lng,
	})
	pgAddress := PGAddress{
		ID:          user.ID,
		UserID:      user.ID,
		Street:      user.Address.Street,
		City:        user.Address.City,
		State:       user.Address.State,
		ZipCode:     user.Address.ZipCode,
		Country:     user.Address.Country,
		Coordinates: coordsJSON,
	}

	// Profile table
	interestsJSON, _ := json.Marshal(user.Profile.Interests)
	skillsJSON, _ := json.Marshal(user.Profile.Skills)
	pgProfile := PGProfile{
		ID:        user.ID,
		UserID:    user.ID,
		Bio:       user.Profile.Bio,
		Interests: interestsJSON,
		Skills:    skillsJSON,
	}

	// Education records
	education := make([]PGEducation, 0, len(user.Profile.Education))
	for idx, edu := range user.Profile.Education {
		education = append(education, PGEducation{
			ID:          user.ID*10000 + int64(idx),
			ProfileID:   user.ID,
			Institution: edu.Institution,
			Degree:      edu.Degree,
			Year:        edu.Year,
			Description: edu.Description,
		})
	}

	// Experience records
	experience := make([]PGExperience, 0, len(user.Profile.Experience))
	for idx, exp := range user.Profile.Experience {
		experience = append(experience, PGExperience{
			ID:          user.ID*10000 + int64(idx),
			ProfileID:   user.ID,
			Company:     exp.Company,
			Position:    exp.Position,
			Duration:    exp.Duration,
			Description: exp.Description,
		})
	}

	// Preferences table
	notifJSON, _ := json.Marshal(user.Preferences.Notifications)
	pgPreferences := PGPreferences{
		ID:            user.ID,
		UserID:        user.ID,
		Language:      user.Preferences.Language,
		Timezone:      user.Preferences.Timezone,
		Notifications: notifJSON,
	}

	// Settings records
	settings := make([]PGSettings, 0, len(user.Preferences.Settings))
	for idx, setting := range user.Preferences.Settings {
		settings = append(settings, PGSettings{
			ID:           user.ID*10000 + int64(idx),
			PreferenceID: user.ID,
			Key:          setting.Key,
			Value:        setting.Value,
			Timestamp:    setting.Timestamp,
			Metadata:     setting.Metadata,
		})
	}

	// Activity log records
	activityLog := make([]PGActivityLog, 0, len(user.ActivityLog))
	for idx, log := range user.ActivityLog {
		activityLog = append(activityLog, PGActivityLog{
			ID:        user.ID*10000 + int64(idx),
			UserID:    user.ID,
			Key:       log.Key,
			Value:     log.Value,
			Timestamp: log.Timestamp,
			Metadata:  log.Metadata,
		})
	}

	// Transaction records
	transactions := make([]PGTransactions, 0, len(user.Transactions))
	for idx, tx := range user.Transactions {
		transactions = append(transactions, PGTransactions{
			ID:        user.ID*10000 + int64(idx),
			UserID:    user.ID,
			Key:       tx.Key,
			Value:     tx.Value,
			Timestamp: tx.Timestamp,
			Metadata:  tx.Metadata,
		})
	}

	// Messages and attachments
	messages := make([]PGMessages, 0, len(user.Messages))
	attachments := make([]PGAttachments, 0)
	for _, msg := range user.Messages {
		messages = append(messages, PGMessages{
			ID:        msg.ID,
			UserID:    user.ID,
			From:      msg.From,
			To:        msg.To,
			Subject:   msg.Subject,
			Body:      msg.Body,
			Timestamp: msg.Timestamp,
			Read:      msg.Read,
		})

		for idx, att := range msg.Attachments {
			attachments = append(attachments, PGAttachments{
				ID:        user.ID*10000 + int64(idx),
				MessageID: msg.ID,
				Name:      att.Name,
				Size:      att.Size,
				FileType:  att.FileType,
			})
		}
	}

	// Social media table
	connectionsJSON, _ := json.Marshal(user.SocialMedia.Connections)
	pgSocialMedia := PGSocialMedia{
		ID:          user.ID,
		UserID:      user.ID,
		Connections: connectionsJSON,
	}

	// Posts records
	posts := make([]PGPosts, 0, len(user.SocialMedia.Posts))
	for idx, post := range user.SocialMedia.Posts {
		posts = append(posts, PGPosts{
			ID:            user.ID*10000 + int64(idx),
			SocialMediaID: user.ID,
			Key:           post.Key,
			Value:         post.Value,
			Timestamp:     post.Timestamp,
			Metadata:      post.Metadata,
		})
	}

	// Groups records
	groups := make([]PGGroups, 0, len(user.SocialMedia.Groups))
	for _, group := range user.SocialMedia.Groups {
		groups = append(groups, PGGroups{
			ID:            group.ID,
			SocialMediaID: user.ID,
			Name:          group.Name,
			Joined:        group.Joined,
		})
	}

	// Large data table
	pgLargeData := PGLargeData{
		ID:     user.ID,
		UserID: user.ID,
		Blob1:  user.LargeData.Blob1,
		Blob2:  user.LargeData.Blob2,
		Blob3:  user.LargeData.Blob3,
		Blob4:  user.LargeData.Blob4,
		Blob5:  user.LargeData.Blob5,
	}

	return TransformedUser{
		User:        pgUser,
		Address:     pgAddress,
		Profile:     pgProfile,
		Education:   education,
		Experience:  experience,
		Preferences: pgPreferences,
		Settings:    settings,
		ActivityLog: activityLog,
		Transactions: transactions,
		Messages:    messages,
		Attachments:  attachments,
		SocialMedia: pgSocialMedia,
		Posts:       posts,
		Groups:      groups,
		LargeData:   pgLargeData,
	}
}

// Load inserts transformed data into PostgreSQL in batches
func (u *UserETL) Load(ctx context.Context, items []TransformedUser) error {
	if len(items) == 0 {
		return nil
	}

	// Collect all entities by table
	users := make([]PGUser, 0, len(items))
	addresses := make([]PGAddress, 0, len(items))
	profiles := make([]PGProfile, 0, len(items))
	var allEducation []PGEducation
	var allExperience []PGExperience
	preferences := make([]PGPreferences, 0, len(items))
	var allSettings []PGSettings
	var allActivityLog []PGActivityLog
	var allTransactions []PGTransactions
	var allMessages []PGMessages
	var allAttachments []PGAttachments
	socialMedia := make([]PGSocialMedia, 0, len(items))
	var allPosts []PGPosts
	var allGroups []PGGroups
	largeData := make([]PGLargeData, 0, len(items))

	for _, item := range items {
		users = append(users, item.User)
		addresses = append(addresses, item.Address)
		profiles = append(profiles, item.Profile)
		allEducation = append(allEducation, item.Education...)
		allExperience = append(allExperience, item.Experience...)
		preferences = append(preferences, item.Preferences)
		allSettings = append(allSettings, item.Settings...)
		allActivityLog = append(allActivityLog, item.ActivityLog...)
		allTransactions = append(allTransactions, item.Transactions...)
		allMessages = append(allMessages, item.Messages...)
		allAttachments = append(allAttachments, item.Attachments...)
		socialMedia = append(socialMedia, item.SocialMedia)
		allPosts = append(allPosts, item.Posts...)
		allGroups = append(allGroups, item.Groups...)
		largeData = append(largeData, item.LargeData)
	}

	// Batch insert in dependency order
	fmt.Printf("Batch inserting %d users...\n", len(users))
	if err := u.postgresDB.CreateInBatches(users, 500).Error; err != nil {
		return fmt.Errorf("failed to insert users: %w", err)
	}

	fmt.Printf("Batch inserting %d addresses...\n", len(addresses))
	if err := u.postgresDB.CreateInBatches(addresses, 500).Error; err != nil {
		return fmt.Errorf("failed to insert addresses: %w", err)
	}

	fmt.Printf("Batch inserting %d profiles...\n", len(profiles))
	if err := u.postgresDB.CreateInBatches(profiles, 500).Error; err != nil {
		return fmt.Errorf("failed to insert profiles: %w", err)
	}

	if len(allEducation) > 0 {
		fmt.Printf("Batch inserting %d education records...\n", len(allEducation))
		if err := u.postgresDB.CreateInBatches(allEducation, 500).Error; err != nil {
			return fmt.Errorf("failed to insert education: %w", err)
		}
	}

	if len(allExperience) > 0 {
		fmt.Printf("Batch inserting %d experience records...\n", len(allExperience))
		if err := u.postgresDB.CreateInBatches(allExperience, 500).Error; err != nil {
			return fmt.Errorf("failed to insert experience: %w", err)
		}
	}

	fmt.Printf("Batch inserting %d preferences...\n", len(preferences))
	if err := u.postgresDB.CreateInBatches(preferences, 500).Error; err != nil {
		return fmt.Errorf("failed to insert preferences: %w", err)
	}

	if len(allSettings) > 0 {
		fmt.Printf("Batch inserting %d settings...\n", len(allSettings))
		if err := u.postgresDB.CreateInBatches(allSettings, 500).Error; err != nil {
			return fmt.Errorf("failed to insert settings: %w", err)
		}
	}

	if len(allActivityLog) > 0 {
		fmt.Printf("Batch inserting %d activity logs...\n", len(allActivityLog))
		if err := u.postgresDB.CreateInBatches(allActivityLog, 500).Error; err != nil {
			return fmt.Errorf("failed to insert activity log: %w", err)
		}
	}

	if len(allTransactions) > 0 {
		fmt.Printf("Batch inserting %d transactions...\n", len(allTransactions))
		if err := u.postgresDB.CreateInBatches(allTransactions, 500).Error; err != nil {
			return fmt.Errorf("failed to insert transactions: %w", err)
		}
	}

	if len(allMessages) > 0 {
		fmt.Printf("Batch inserting %d messages...\n", len(allMessages))
		if err := u.postgresDB.CreateInBatches(allMessages, 500).Error; err != nil {
			return fmt.Errorf("failed to insert messages: %w", err)
		}
	}

	if len(allAttachments) > 0 {
		fmt.Printf("Batch inserting %d attachments...\n", len(allAttachments))
		if err := u.postgresDB.CreateInBatches(allAttachments, 500).Error; err != nil {
			return fmt.Errorf("failed to insert attachments: %w", err)
		}
	}

	fmt.Printf("Batch inserting %d social media records...\n", len(socialMedia))
	if err := u.postgresDB.CreateInBatches(socialMedia, 500).Error; err != nil {
		return fmt.Errorf("failed to insert social media: %w", err)
	}

	if len(allPosts) > 0 {
		fmt.Printf("Batch inserting %d posts...\n", len(allPosts))
		if err := u.postgresDB.CreateInBatches(allPosts, 500).Error; err != nil {
			return fmt.Errorf("failed to insert posts: %w", err)
		}
	}

	if len(allGroups) > 0 {
		fmt.Printf("Batch inserting %d groups...\n", len(allGroups))
		if err := u.postgresDB.CreateInBatches(allGroups, 500).Error; err != nil {
			return fmt.Errorf("failed to insert groups: %w", err)
		}
	}

	fmt.Printf("Batch inserting %d large data records...\n", len(largeData))
	if err := u.postgresDB.CreateInBatches(largeData, 500).Error; err != nil {
		return fmt.Errorf("failed to insert large data: %w", err)
	}

	fmt.Printf("✓ Batch inserted %d users with all related data!\n", len(items))
	return nil
}

// PostProcess cleanup after ETL
func (u *UserETL) PostProcess(ctx context.Context) error {
	fmt.Println("ETL pipeline completed successfully!")
	return nil
}
