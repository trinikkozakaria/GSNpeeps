package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// notificationStoreStub mencatat argumen agar cakupan penerima dapat diperiksa.
type notificationStoreStub struct {
	listRecipient  uuid.UUID
	listIsRead     *bool
	listPage       int
	listLimit      int
	page           domain.NotificationPage
	unread         int
	markRecipient  uuid.UUID
	markID         uuid.UUID
	markErr        error
	dismissedBy    uuid.UUID
	dismissedID    uuid.UUID
	dismissErr     error
	dismissedTimes int
}

func (s *notificationStoreStub) List(
	_ context.Context, recipient uuid.UUID, isRead *bool, page, limit int,
) (domain.NotificationPage, error) {
	s.listRecipient, s.listIsRead, s.listPage, s.listLimit = recipient, isRead, page, limit
	return s.page, nil
}

func (s *notificationStoreStub) UnreadCount(
	_ context.Context, recipient uuid.UUID,
) (int, error) {
	s.listRecipient = recipient
	return s.unread, nil
}

func (s *notificationStoreStub) MarkRead(
	_ context.Context, recipient, id uuid.UUID, _ time.Time,
) error {
	s.markRecipient, s.markID = recipient, id
	return s.markErr
}

func (s *notificationStoreStub) Dismiss(
	_ context.Context, recipient, id uuid.UUID, _ time.Time,
) error {
	s.dismissedBy, s.dismissedID = recipient, id
	s.dismissedTimes++
	return s.dismissErr
}

// Penerima selalu berasal dari token, tidak pernah dari parameter request.
func TestNotificationListIsScopedToAuthenticatedUser(t *testing.T) {
	store := &notificationStoreStub{}
	service := NewNotificationService(store)
	identity := employeeIdentity()
	isRead := false

	_, err := service.List(context.Background(), identity, &isRead, 2, 25)

	require.NoError(t, err)
	assert.Equal(t, identity.UserID, store.listRecipient)
	require.NotNil(t, store.listIsRead)
	assert.False(t, *store.listIsRead)
	assert.Equal(t, 2, store.listPage)
	assert.Equal(t, 25, store.listLimit)
}

func TestNotificationListClampsPaging(t *testing.T) {
	store := &notificationStoreStub{}
	service := NewNotificationService(store)

	_, err := service.List(context.Background(), employeeIdentity(), nil, 0, 5_000)

	require.NoError(t, err)
	assert.Equal(t, 1, store.listPage)
	assert.Equal(t, 100, store.listLimit)
}

func TestUnreadCountIsScopedToAuthenticatedUser(t *testing.T) {
	store := &notificationStoreStub{unread: 4}
	service := NewNotificationService(store)
	identity := employeeIdentity()

	total, err := service.UnreadCount(context.Background(), identity)

	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Equal(t, identity.UserID, store.listRecipient)
}

// Notifikasi yang tidak dimiliki dan yang tidak ada sama-sama menghasilkan 403 (D-032)
// sehingga keberadaan ID milik orang lain tidak dapat dipetakan.
func TestMarkReadHidesForeignNotificationsBehindForbidden(t *testing.T) {
	store := &notificationStoreStub{markErr: repository.ErrNotFound}
	service := NewNotificationService(store)

	err := service.MarkRead(context.Background(), employeeIdentity(), uuid.New())

	assert.ErrorIs(t, err, domain.ErrForbidden)
	assert.NotErrorIs(t, err, domain.ErrNotFound)
}

func TestDismissHidesForeignNotificationsBehindForbidden(t *testing.T) {
	store := &notificationStoreStub{dismissErr: repository.ErrNotFound}
	service := NewNotificationService(store)

	err := service.Dismiss(context.Background(), employeeIdentity(), uuid.New())

	assert.ErrorIs(t, err, domain.ErrForbidden)
	assert.NotErrorIs(t, err, domain.ErrNotFound)
}

func TestMarkReadPassesOwnerAndIdentifier(t *testing.T) {
	store := &notificationStoreStub{}
	service := NewNotificationService(store)
	identity := employeeIdentity()
	id := uuid.New()

	require.NoError(t, service.MarkRead(context.Background(), identity, id))

	assert.Equal(t, identity.UserID, store.markRecipient)
	assert.Equal(t, id, store.markID)
}

// Dismiss berulang tetap berhasil dan tetap merupakan soft-delete.
func TestDismissIsIdempotent(t *testing.T) {
	store := &notificationStoreStub{}
	service := NewNotificationService(store)
	identity := employeeIdentity()
	id := uuid.New()

	require.NoError(t, service.Dismiss(context.Background(), identity, id))
	require.NoError(t, service.Dismiss(context.Background(), identity, id))

	assert.Equal(t, 2, store.dismissedTimes)
	assert.Equal(t, identity.UserID, store.dismissedBy)
	assert.Equal(t, id, store.dismissedID)
}

func TestMarkReadWrapsUnexpectedFailure(t *testing.T) {
	store := &notificationStoreStub{markErr: errors.New("koneksi putus")}
	service := NewNotificationService(store)

	err := service.MarkRead(context.Background(), employeeIdentity(), uuid.New())

	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrForbidden)
	assert.Contains(t, err.Error(), "mark notification read")
}
