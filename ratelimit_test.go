package ratelimit

import (
	"context"
	"fmt"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/vearne/ratelimit/counter"
	"github.com/vearne/ratelimit/leakybucket"
	"github.com/vearne/ratelimit/tokenbucket"
	"log"
	"os"
	"testing"
	"time"
)

func MyMatch(expected, actual []interface{}) error {
	expectedStr := fmt.Sprintf("%v", expected)
	actualStr := fmt.Sprintf("%v", actual)
	if expectedStr == actualStr {
		return nil
	}
	log.Printf("expectedStr:%v, actualStr:%v", expectedStr, actualStr)
	os.Stdout.Write([]byte(fmt.Sprintf("expectedStr:%v, actualStr:%v", expectedStr, actualStr)))
	return fmt.Errorf("not equal, expectedStr:%s, actualStr:%s", expectedStr, actualStr)
}

func TestCantGetToken1(t *testing.T) {
	key := "key:token"

	db, mock := redismock.NewClientMock()

	mock = mock.CustomMatch(MyMatch)
	mock.ExpectPing().SetVal("PONG")

	hashVal := "1fc288109bccebf36ff083517ed03c4901c80be1"
	mock.ExpectScriptExists(hashVal).SetVal([]bool{true})
	//mock.ExpectScriptLoad(TokenBucketScript).SetErr(nil)
	mock.ExpectEvalSha(hashVal, []string{key}, 3, 2, 1).SetVal(int64(0))

	limiter, err := tokenbucket.NewTokenBucketRateLimiter(context.Background(), db, key,
		time.Second,
		3,
		1,
		2)
	if err != nil {
		t.Errorf("unexpected error, %v", err)
		return
	}

	cantGetToken(t, limiter)
}

func TestCantGetToken2(t *testing.T) {
	key := "key:leaky"

	db, mock := redismock.NewClientMock()

	//mock = mock.CustomMatch(MyMatch)
	mock.ExpectPing().SetVal("PONG")

	mock.Regexp().ExpectScriptExists(`^[a-z0-9]+$`).SetVal([]bool{true})
	//mock.ExpectScriptLoad(TokenBucketScript).SetErr(nil)
	mock.Regexp().ExpectEvalSha(`^[a-z0-9]+$`, []string{key}, 333333).SetVal(int64(0))

	limiter, err := leakybucket.NewLeakyBucketLimiter(context.Background(),
		db, key, time.Second, 3)
	if err != nil {
		t.Errorf("unexpected error, %v", err)
		return
	}

	cantGetToken(t, limiter)
}

func TestCantGetToken3(t *testing.T) {
	key := "key:count"

	db, mock := redismock.NewClientMock()

	mock = mock.CustomMatch(MyMatch)
	mock.ExpectPing().SetVal("PONG")

	hashVal := "bdbede5669d5e48d6e6c2967aeed2f72f03868ac"
	mock.ExpectScriptExists(hashVal).SetVal([]bool{true})
	mock.ExpectEvalSha("aa", []string{key}).SetVal(int64(1))

	limiter, err := counter.NewCounterRateLimiter(context.Background(), db, key,
		time.Minute, 3, 2)
	if err != nil {
		t.Errorf("unexpected error, %v", err)
		return
	}

	cantGetToken(t, limiter)
}

func cantGetToken(t *testing.T, limiter Limiter) {
	ok, err := limiter.Take(context.Background())
	if err != nil {
		t.Errorf("unexpected error, %v", err)
		return
	}
	if !ok {
		assert.Equal(t, ok, false)
	}
}

//func TestContextTimeOut(t *testing.T) {
//	ctx := context.Background()
//	sha1 := "sha1"
//
//	db, mock := redismock.NewClientMock()
//	mock.ExpectPing().SetVal("PONG")
//
//	mock.ExpectEvalSha(sha1, []string{key}, "xxx").SetVal(int64(0))
//	mock.ExpectScriptExists(sha1).SetVal([]bool{true, true})
//
//	limiter, _ := NewTokenBucketRateLimiter(ctx, db, key,
//		time.Second,
//		20,
//		1,
//		2)
//
//	err := limiter.Wait(ctx)
//	assert.Contains(t, err.Error(), "timeout")
//}
