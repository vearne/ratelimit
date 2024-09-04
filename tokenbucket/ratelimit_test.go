package tokenbucket

import (
	"context"
	"fmt"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

const (
	key     = "key:token"
	hashVal = "1fc288109bccebf36ff083517ed03c4901c80be1"
)

func MyMatch(expected, actual []interface{}) error {
	expectedStr := fmt.Sprintf("%v", expected)
	actualStr := fmt.Sprintf("%v", actual)
	if expectedStr == actualStr {
		return nil
	}
	log.Printf("expectedStr:%v, actualStr:%v", expectedStr, actualStr)
	return fmt.Errorf("not equal, expectedStr:%s, actualStr:%s", expectedStr, actualStr)
}

func TestTakeFail(t *testing.T) {
	db, mock := redismock.NewClientMock()

	mock = mock.CustomMatch(MyMatch)
	mock.ExpectPing().SetVal("PONG")

	mock.ExpectScriptExists(hashVal).SetVal([]bool{true})
	mock.ExpectEvalSha(hashVal, []string{key}, 3, 2, 1).SetVal(int64(0))

	limiter, err := NewTokenBucketRateLimiter(context.Background(), db, key,
		time.Second,
		3,
		1,
		2, WithAntiDDos(false))
	if err != nil {
		t.Errorf("unexpected error, %v", err)
		return
	}

	ok, err := limiter.Take(context.Background())
	if err != nil {
		t.Errorf("unexpected error, %v", err)
		return
	}
	if !ok {
		assert.Equal(t, ok, false)
	}
}

func TestTakeSuccess(t *testing.T) {
	db, mock := redismock.NewClientMock()

	mock = mock.CustomMatch(MyMatch)
	mock.ExpectPing().SetVal("PONG")

	mock.ExpectScriptExists(hashVal).SetVal([]bool{true})
	mock.ExpectEvalSha(hashVal, []string{key}, 3, 2, 1).SetVal(int64(1))

	limiter, err := NewTokenBucketRateLimiter(context.Background(), db, key,
		time.Second,
		3,
		1,
		2, WithAntiDDos(false))
	if err != nil {
		t.Errorf("unexpected error, %v", err)
		return
	}

	ok, err := limiter.Take(context.Background())
	if err != nil {
		t.Errorf("unexpected error, %v", err)
		return
	}
	if !ok {
		assert.Equal(t, ok, true)
	}
}

func TestContextTimeOut(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock = mock.CustomMatch(MyMatch)
	mock.ExpectPing().SetVal("PONG")

	mock.ExpectScriptExists(hashVal).SetVal([]bool{true, true})
	for i := 0; i < 1000; i++ {
		mock.ExpectEvalSha(hashVal, []string{key}, 3, 2, 1).SetVal(int64(0))
	}

	limiter, err := NewTokenBucketRateLimiter(context.Background(), db, key,
		time.Second,
		3,
		1,
		2)
	if err != nil {
		t.Errorf("unexpected error, %v", err)
		return
	}

	waitCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = limiter.Wait(waitCtx)
	assert.Contains(t, err.Error(), "timeout")
}
