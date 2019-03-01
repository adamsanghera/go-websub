package sql

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

/*
	# Test Cases

	## Valid Cases

	1. An active subscription is invalidated
	2. An initiated, but unleased subscription is invalidated

	## Error Cases

	1. Callback DNE
	2. Repeated-invalidation
	3. Context cancelled.
*/

func TestSQL_ValidCases(t *testing.T) {
	sqlStor, err := New(NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = sqlStor.Shutdown(); err != nil {
			t.Fatal(err)
		}
	}()

	sqlStor.IndexOffer(map[string]string{
		"topic": "hub",
	})

	// 1. Active subscription invalidated
	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "callback")
	if err != nil {
		t.Fatal(err)
	}

	err = sqlStor.ExtendLease(context.Background(), "callback", time.Now().Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}

	err = sqlStor.Invalidate(context.Background(), "callback", "hub denied")
	if err != nil {
		t.Fatal(err)
	}

	_, err = sqlStor.GetSubscription("callback")
	if err != sql.ErrNoRows {
		t.Fatal(err)
	}

	// 2. Inactive, but initiated subscription invalidated
	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "callback2")
	if err != nil {
		t.Fatal(err)
	}

	err = sqlStor.Invalidate(context.Background(), "callback2", "hub denied")
	if err != nil {
		t.Fatal(err)
	}

	_, err = sqlStor.GetSubscription("callback2")
	if err != sql.ErrNoRows {
		t.Fatal(err)
	}
}

func TestSQL_Invalidate_ErrCases(t *testing.T) {
	sqlStor, err := New(NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = sqlStor.Shutdown(); err != nil {
			t.Fatal(err)
		}
	}()

	// 1. Callback DNE
	err = sqlStor.Invalidate(context.Background(), "nonexistantcb", "bad reason")
	if errUp, ok := err.(ErrUpdateFailed); !ok || errUp.numTouched != 0 {
		t.Fatal(err)
	}

	// 2. Repeated invalidation
	sqlStor.IndexOffer(map[string]string{
		"topic": "hub",
	})

	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "callback")
	if err != nil {
		t.Fatal(err)
	}

	err = sqlStor.Invalidate(context.Background(), "callback", "good reason")
	if err != nil {
		t.Fatal(err)
	}

	err = sqlStor.Invalidate(context.Background(), "callback", "good reason")
	if errUp, ok := err.(ErrUpdateFailed); !ok || errUp.numTouched != 0 {
		t.Fatal(err)
	}

	// 3. Context cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = sqlStor.Invalidate(ctx, "callback", "good reason")
	if err != context.Canceled {
		t.Fatal(err)
	}
}
