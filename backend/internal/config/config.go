package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App       App
	HTTP      HTTP
	Postgres  Postgres
	Redis     Redis
	JWT       JWT
	Auth      Auth
	Storage   Storage
	MinIO     MinIO
	Nextcloud Nextcloud
}

type App struct {
	Environment string
	LogLevel    string
}

type HTTP struct {
	Address           string
	AllowedOrigin     string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxBodyBytes      int64
	MaxUploadBytes    int64
	TrustProxy        bool
}

type Postgres struct {
	URL             string
	MaxConnections  int32
	MinConnections  int32
	MaxConnLifetime time.Duration
	HealthTimeout   time.Duration
}

type Redis struct {
	URL           string
	PoolSize      int
	DialTimeout   time.Duration
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	HealthTimeout time.Duration
}

type JWT struct {
	Secret string
	TTL    time.Duration
}

type Auth struct {
	LoginFailureLimit int
	LoginRateLimit    int
	RequestLimit      int
	RequestWindow     time.Duration
	ArgonMemoryKiB    uint32
	ArgonIterations   uint32
	ArgonParallelism  uint8
}

type Nextcloud struct {
	BaseURL     string
	Username    string
	AppPassword string
	RootFolder  string
	Timeout     time.Duration
}

// StorageDriver memilih adapter object storage aktif di balik interface filestore.Store.
type StorageDriver string

const (
	// StorageDriverMinIO adalah default sejak keputusan 2026-08-20; lihat
	// docs/architecture/minio-integration.md.
	StorageDriverMinIO     StorageDriver = "minio"
	StorageDriverNextcloud StorageDriver = "nextcloud"
)

type Storage struct {
	Driver StorageDriver
}

type MinIO struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	Region    string
	Timeout   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		App: App{
			Environment: get("APP_ENV", "development"),
			LogLevel:    get("LOG_LEVEL", "info"),
		},
		HTTP: HTTP{
			Address:           get("HTTP_ADDRESS", ":8080"),
			AllowedOrigin:     os.Getenv("CORS_ALLOWED_ORIGIN"),
			ReadTimeout:       duration("HTTP_READ_TIMEOUT", 15*time.Second),
			ReadHeaderTimeout: duration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
			WriteTimeout:      duration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:       duration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout:   duration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
			MaxBodyBytes:      int64Value("HTTP_MAX_BODY_BYTES", 2<<20),
			// 5 MB berkas ditambah ruang untuk boundary dan field multipart lain.
			MaxUploadBytes: int64Value("HTTP_MAX_UPLOAD_BYTES", 6<<20),
			TrustProxy:     boolValue("HTTP_TRUST_PROXY", false),
		},
		Postgres: Postgres{
			URL:             os.Getenv("DATABASE_URL"),
			MaxConnections:  int32(intValue("POSTGRES_MAX_CONNECTIONS", 20)),
			MinConnections:  int32(intValue("POSTGRES_MIN_CONNECTIONS", 2)),
			MaxConnLifetime: duration("POSTGRES_MAX_CONN_LIFETIME", 30*time.Minute),
			HealthTimeout:   duration("POSTGRES_HEALTH_TIMEOUT", 3*time.Second),
		},
		Redis: Redis{
			URL:           os.Getenv("REDIS_URL"),
			PoolSize:      intValue("REDIS_POOL_SIZE", 10),
			DialTimeout:   duration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:   duration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout:  duration("REDIS_WRITE_TIMEOUT", 3*time.Second),
			HealthTimeout: duration("REDIS_HEALTH_TIMEOUT", 3*time.Second),
		},
		JWT: JWT{
			Secret: os.Getenv("JWT_SECRET"),
			TTL:    duration("JWT_TTL", 8*time.Hour),
		},
		Auth: Auth{
			LoginFailureLimit: intValue("AUTH_LOGIN_FAILURE_LIMIT", 5),
			LoginRateLimit:    intValue("AUTH_LOGIN_RATE_LIMIT", 10),
			RequestLimit:      intValue("AUTH_REQUEST_LIMIT", 120),
			RequestWindow:     duration("AUTH_REQUEST_WINDOW", time.Minute),
			ArgonMemoryKiB:    uint32(intValue("AUTH_ARGON_MEMORY_KIB", 64*1024)),
			ArgonIterations:   uint32(intValue("AUTH_ARGON_ITERATIONS", 3)),
			ArgonParallelism:  uint8(intValue("AUTH_ARGON_PARALLELISM", 2)),
		},
		Nextcloud: Nextcloud{
			BaseURL:     os.Getenv("NEXTCLOUD_WEBDAV_URL"),
			Username:    os.Getenv("NEXTCLOUD_USERNAME"),
			AppPassword: os.Getenv("NEXTCLOUD_APP_PASSWORD"),
			RootFolder:  get("NEXTCLOUD_ROOT_FOLDER", "GSNpeeps"),
			Timeout:     duration("NEXTCLOUD_HTTP_TIMEOUT", 20*time.Second),
		},
		Storage: Storage{
			Driver: StorageDriver(get("STORAGE_DRIVER", string(StorageDriverMinIO))),
		},
		MinIO: MinIO{
			Endpoint:  os.Getenv("MINIO_ENDPOINT"),
			AccessKey: os.Getenv("MINIO_ACCESS_KEY"),
			SecretKey: os.Getenv("MINIO_SECRET_KEY"),
			Bucket:    get("MINIO_BUCKET", "gsnpeeps"),
			UseSSL:    boolValue("MINIO_USE_SSL", false),
			// Region tetap dan diketahui di muka (self-hosted, bukan multi-region AWS), jadi
			// nilai default diisi eksplisit agar minio-go tidak perlu request GetBucketLocation
			// tambahan sebelum setiap request pertama.
			Region:  get("MINIO_REGION", "us-east-1"),
			Timeout: duration("MINIO_HTTP_TIMEOUT", 20*time.Second),
		},
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	var missing []string
	required := map[string]string{
		"CORS_ALLOWED_ORIGIN": c.HTTP.AllowedOrigin,
		"DATABASE_URL":        c.Postgres.URL,
		"REDIS_URL":           c.Redis.URL,
		"JWT_SECRET":          c.JWT.Secret,
	}
	// Hanya mewajibkan credential adapter object storage yang aktif; driver yang tidak
	// dipilih tidak perlu dikonfigurasi.
	switch c.Storage.Driver {
	case StorageDriverMinIO:
		required["MINIO_ENDPOINT"] = c.MinIO.Endpoint
		required["MINIO_ACCESS_KEY"] = c.MinIO.AccessKey
		required["MINIO_SECRET_KEY"] = c.MinIO.SecretKey
		required["MINIO_BUCKET"] = c.MinIO.Bucket
	case StorageDriverNextcloud:
		required["NEXTCLOUD_WEBDAV_URL"] = c.Nextcloud.BaseURL
		required["NEXTCLOUD_USERNAME"] = c.Nextcloud.Username
		required["NEXTCLOUD_APP_PASSWORD"] = c.Nextcloud.AppPassword
	default:
		return fmt.Errorf("unsupported STORAGE_DRIVER %q", c.Storage.Driver)
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	if c.App.Environment == "production" && c.HTTP.AllowedOrigin == "*" {
		return errors.New("CORS_ALLOWED_ORIGIN cannot be wildcard in production")
	}
	if len(c.JWT.Secret) < 32 {
		return errors.New("JWT_SECRET must contain at least 32 characters")
	}
	return nil
}

func get(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func duration(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func intValue(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil {
		return fallback
	}
	return value
}

func int64Value(name string, fallback int64) int64 {
	value, err := strconv.ParseInt(os.Getenv(name), 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func boolValue(name string, fallback bool) bool {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
