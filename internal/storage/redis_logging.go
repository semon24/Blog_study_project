package storage

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLoggerHook struct{}

func (RedisLoggerHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (RedisLoggerHook) ProcessHook(
	next redis.ProcessHook,
) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		log.Printf("Начало Redis команды: %s", cmd.Name())
		err := next(ctx, cmd)
		log.Printf(
			"Redis конец: %s | время: %s | ошибка: %v",
			cmd.Name(),
			time.Since(start),
			err,
		)
		return err
	}
}

func (RedisLoggerHook) ProcessPipelineHook(
	next redis.ProcessPipelineHook,
) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()

		log.Printf("Redis pipeline начало: команд=%d", len(cmds))

		err := next(ctx, cmds)

		log.Printf(
			"Redis pipeline конец: команд=%d | время=%s | ошибка=%v",
			len(cmds),
			time.Since(start),
			err,
		)

		return err
	}
}
