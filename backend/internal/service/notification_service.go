package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
)

// NotificationStore adalah kebutuhan pembacaan notifikasi dari sisi service.
type NotificationStore interface {
	List(context.Context, uuid.UUID, *bool, int, int) (domain.NotificationPage, error)
	UnreadCount(context.Context, uuid.UUID) (int, error)
	MarkRead(context.Context, uuid.UUID, uuid.UUID, time.Time) error
	Dismiss(context.Context, uuid.UUID, uuid.UUID, time.Time) error
}

// NotificationService melayani inbox milik pengguna yang sedang login.
//
// Tidak ada operasi yang menerima recipient dari request. Seluruh method memakai
// `identity.UserID` dari token sehingga cakupan baris ditentukan server, bukan pemanggil.
type NotificationService struct {
	notifications NotificationStore
	now           func() time.Time
}

func NewNotificationService(notifications NotificationStore) *NotificationService {
	return &NotificationService{notifications: notifications, now: time.Now}
}

func (s *NotificationService) List(
	ctx context.Context,
	identity domain.Identity,
	isRead *bool,
	page, limit int,
) (domain.NotificationPage, error) {
	page, limit = normalizePaging(page, limit)
	result, err := s.notifications.List(ctx, identity.UserID, isRead, page, limit)
	if err != nil {
		return domain.NotificationPage{}, fmt.Errorf("list notifications: %w", err)
	}
	return result, nil
}

func (s *NotificationService) UnreadCount(
	ctx context.Context,
	identity domain.Identity,
) (int, error) {
	total, err := s.notifications.UnreadCount(ctx, identity.UserID)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return total, nil
}

// MarkRead bersifat idempotent; pemanggilan berulang tetap berhasil.
func (s *NotificationService) MarkRead(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
) error {
	err := s.notifications.MarkRead(ctx, identity.UserID, id, s.now().UTC())
	if err != nil {
		return mapNotificationOwnershipError(err, "mark notification read")
	}
	return nil
}

// Dismiss mengisi `dismissed_at`; notifikasi tidak pernah dihapus fisik sehingga event yang
// sama tidak dapat muncul kembali ketika producer melakukan retry.
func (s *NotificationService) Dismiss(
	ctx context.Context,
	identity domain.Identity,
	id uuid.UUID,
) error {
	err := s.notifications.Dismiss(ctx, identity.UserID, id, s.now().UTC())
	if err != nil {
		return mapNotificationOwnershipError(err, "dismiss notification")
	}
	return nil
}

// mapNotificationOwnershipError menyeragamkan notifikasi yang tidak ada dan notifikasi milik
// pengguna lain menjadi satu kode 403 (D-032). Perbedaan kode akan membocorkan ID mana yang
// benar-benar ada.
func mapNotificationOwnershipError(err error, operation string) error {
	if errors.Is(err, repository.ErrNotFound) {
		return domain.ErrForbidden
	}
	return fmt.Errorf("%s: %w", operation, err)
}
