package domain

import "time"

type BackupMetadata struct {
	AppVersion string
	ExportedAt time.Time
	ExportedBy string
}

type BackupFile struct {
	Metadata     BackupMetadata
	Users        []User
	Accounts     []UserAccount
	Teams        []Team
	Repositories []Repository
	Config       map[string]string
}
