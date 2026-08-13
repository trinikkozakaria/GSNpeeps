package service

import (
	"context"
	"sync"
)

type Pinger interface {
	Ping(context.Context) error
}

type HealthResult struct {
	Status string `json:"status"`
	DB     string `json:"db"`
	Redis  string `json:"redis"`
}

type HealthService struct {
	database Pinger
	redis    Pinger
}

func NewHealthService(database, redis Pinger) *HealthService {
	return &HealthService{database: database, redis: redis}
}

func (s *HealthService) Check(ctx context.Context) (HealthResult, bool) {
	result := HealthResult{Status: "ok", DB: "ok", Redis: "ok"}
	var dbErr, redisErr error
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		dbErr = s.database.Ping(ctx)
	}()
	go func() {
		defer wait.Done()
		redisErr = s.redis.Ping(ctx)
	}()
	wait.Wait()

	healthy := true
	if dbErr != nil {
		result.DB = "unavailable"
		healthy = false
	}
	if redisErr != nil {
		result.Redis = "unavailable"
		healthy = false
	}
	if !healthy {
		result.Status = "degraded"
	}
	return result, healthy
}
