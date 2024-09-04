package counter

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
	key     = "key:count"
	hashVal = "bdbede5669d5e48d6e6c2967aeed2f72f03868ac"
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
	mock.ExpectEvalSha(hashVal, []string{key}, 1000000, 3, 2).SetVal(int64(0))

	limiter, err := NewCounterRateLimiter(context.Background(), db, key, time.Second,
		3,
		2,
		WithAntiDDos(false))
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
	mock.ExpectEvalSha(hashVal, []string{key}, 1000000, 3, 2).SetVal(int64(1))

	limiter, err := NewCounterRateLimiter(context.Background(), db, key, time.Second,
		3,
		2,
		WithAntiDDos(false))
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
		mock.ExpectEvalSha(hashVal, []string{key}, 1000000, 3, 2).SetVal(int64(0))
	}

	limiter, err := NewCounterRateLimiter(context.Background(), db, key, time.Second,
		3,
		2,
		WithAntiDDos(false))
	if err != nil {
		t.Errorf("unexpected error, %v", err)
		return
	}

	waitCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = limiter.Wait(waitCtx)
	assert.Contains(t, err.Error(), "timeout")
}
