package application

import (
	"context"

	"contribution-tracker/internal/domain"
)

type BackupService struct {
	backupRepo BackupRepository
}

func NewBackupService(backupRepo BackupRepository) *BackupService {
	return &BackupService{backupRepo: backupRepo}
}

func (s *BackupService) Export(ctx context.Context) (*domain.BackupFile, error) {
	return s.backupRepo.Export(ctx)
}

func (s *BackupService) Restore(ctx context.Context, data *domain.BackupFile) error {
	return s.backupRepo.Restore(ctx, data)
}
