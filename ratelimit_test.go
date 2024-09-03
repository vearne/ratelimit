package ratelimit

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	key = "key:token"
)

func TestCantGetTokenBefore(t *testing.T) {
	ctrl := gomock.NewController(t)
	// Assert that Bar() is invoked.
	defer ctrl.Finish()
	client := NewMockCmdable(ctrl)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var sc redis.StatusCmd
	sc.SetVal("PONG")
	client.EXPECT().Ping(ctx).Return(&sc)
	var c redis.Cmd
	c.SetVal(int64(0))
	client.EXPECT().EvalSha(ctx, gomock.Any(), []string{key}, gomock.Any()).Return(&c).AnyTimes()
	var bsc redis.BoolSliceCmd
	bsc.SetVal([]bool{true})
	client.EXPECT().ScriptExists(ctx, gomock.Any()).Return(&bsc)
	limiter, _ := NewTokenBucketRateLimiter(ctx, client, key,
		time.Second,
		3,
		1,
		2)

	for i := 0; i < 3; i++ {
		err := limiter.Wait(ctx)
		if i == 2 {
			assert.Contains(t, err.Error(), "can't get token")
		}
	}
}

func TestContextTimeOut(t *testing.T) {
	ctrl := gomock.NewController(t)
	// Assert that Bar() is invoked.
	defer ctrl.Finish()
	client := NewMockCmdable(ctrl)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var sc redis.StatusCmd
	sc.SetVal("PONG")
	client.EXPECT().Ping(ctx).Return(&sc)
	var c redis.Cmd
	c.SetVal(int64(0))
	client.EXPECT().EvalSha(ctx, gomock.Any(), []string{key}, gomock.Any()).Return(&c).AnyTimes()
	var bsc redis.BoolSliceCmd
	bsc.SetVal([]bool{true})
	client.EXPECT().ScriptExists(ctx, gomock.Any()).Return(&bsc)
	limiter, _ := NewTokenBucketRateLimiter(ctx, client, key,
		time.Second,
		20,
		1,
		2)

	err := limiter.Wait(ctx)
	assert.Contains(t, err.Error(), "timeout")
}
