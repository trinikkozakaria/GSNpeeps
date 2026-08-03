package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	goredis "github.com/redis/go-redis/v9"
)

// permissionCacheTTL menjaga cache tetap segar bila sebuah instance melewatkan invalidasi,
// misalnya karena restart tepat setelah perubahan. Invalidasi eksplisit tetap menjadi jalur
// utama sehingga perubahan HR berlaku seketika.
const permissionCacheTTL = 5 * time.Minute

// PermissionCache menyimpan kapabilitas satu role sebagai satu dokumen JSON. Seluruh role
// dimuat sekaligus supaya satu pemeriksaan permission tidak menghasilkan satu round trip
// Redis per modul.
type PermissionCache struct {
	client *goredis.Client
}

func NewPermissionCache(client *goredis.Client) *PermissionCache {
	return &PermissionCache{client: client}
}

// Load mengembalikan kapabilitas role bila tersedia. Nilai kedua menyatakan cache hit.
func (c *PermissionCache) Load(
	ctx context.Context,
	role domain.RoleName,
) (map[string]bool, bool, error) {
	raw, err := c.client.Get(ctx, permissionKey(role)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read permission cache: %w", err)
	}
	capabilities := map[string]bool{}
	if err := json.Unmarshal(raw, &capabilities); err != nil {
		// Entri rusak diperlakukan sebagai miss; pemanggil membaca ulang dari database.
		return nil, false, fmt.Errorf("decode permission cache: %w", err)
	}
	return capabilities, true, nil
}

func (c *PermissionCache) Store(
	ctx context.Context,
	role domain.RoleName,
	capabilities map[string]bool,
) error {
	encoded, err := json.Marshal(capabilities)
	if err != nil {
		return fmt.Errorf("encode permission cache: %w", err)
	}
	if err := c.client.Set(ctx, permissionKey(role), encoded, permissionCacheTTL).Err(); err != nil {
		return fmt.Errorf("write permission cache: %w", err)
	}
	return nil
}

// Invalidate menghapus cache satu role. Dipanggil di dalam transaction perubahan permission
// sehingga kegagalannya membatalkan perubahan.
func (c *PermissionCache) Invalidate(ctx context.Context, role domain.RoleName) error {
	if err := c.client.Del(ctx, permissionKey(role)).Err(); err != nil {
		return fmt.Errorf("invalidate permission cache: %w", err)
	}
	return nil
}

func permissionKey(role domain.RoleName) string {
	return "permissions:" + string(role)
}
