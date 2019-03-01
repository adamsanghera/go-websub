package sql

import (
	"context"
	"testing"
	"time"
)

/*
	Test Cases:

	(Valid cases)
	1. Lease extended on a fresh callback
	2. Lease extended on an active callback

	(Simple error cases)
	1. Lease extended on an expired callback
	2. Lease extended, but the callback DNE
	3. Context expends during lease extension
	4. Lease given is in the past

	(Complex error cases)
	1. Lease extension and cancel are fighting
*/

func TestSQL_ExtendLease(t *testing.T) {
	sqlStor, err := New(NewConfig())
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err := sqlStor.Shutdown()
		if err != nil {
			t.Fatal(err)
		}
	}()

	sqlStor.IndexOffer(map[string]string{
		"topic": "hub",
	})

	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "callback")
	if err != nil {
		t.Fatal(err)
	}

	// Valid 1
	err = sqlStor.ExtendLease(context.Background(), "callback", time.Now().Add(time.Second*6))
	if err != nil {
		t.Fatal(err)
	}

	// Valid 2
	err = sqlStor.ExtendLease(context.Background(), "callback", time.Now().Add(time.Second*8))
	if err != nil {
		t.Fatal(err)
	}
}

func TestSQL_ExtendLease_SimpleErr(t *testing.T) {
	sqlStor, err := New(NewConfig())
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err := sqlStor.Shutdown()
		if err != nil {
			t.Fatal(err)
		}
	}()

	sqlStor.IndexOffer(map[string]string{
		"topic": "hub",
	})

	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "callback")
	if err != nil {
		t.Fatal(err)
	}

	// 1. Lease extended on an expired callback
	err = sqlStor.ExtendLease(context.Background(), "callback", time.Now().Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)

	err = sqlStor.ExtendLease(context.Background(), "callback", time.Now().Add(time.Second))
	if _, ok := err.(ErrUpdateFailed); !ok {
		t.Fatal(err)
	}

	// 2. Lease extended, but the callback DNE
	err = sqlStor.ExtendLease(context.Background(), "bad_callback", time.Now().Add(time.Second))
	if _, ok := err.(ErrUpdateFailed); !ok {
		t.Fatal(err)
	}

	// 3. Context expends during lease extension
	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "fresh_callback")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = sqlStor.ExtendLease(ctx, "fresh_callback", time.Now().Add(12*time.Second))
	if err != context.Canceled {
		t.Fatal(err)
	}

	// 4. Lease given is in the past
	badTime := time.Now().Add(-1 * time.Minute)
	err = sqlStor.ExtendLease(context.Background(), "fresh_callback", badTime)
	if _, ok := err.(ErrNewLeaseInPast); !ok {
		t.Fatal(err)
	}
}

func TestSQL_ExtendLease_ComplexErr(t *testing.T) {
	sqlStor, err := New(NewConfig())
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err := sqlStor.Shutdown()
		if err != nil {
			t.Fatal(err)
		}
	}()

	sqlStor.IndexOffer(map[string]string{
		"topic": "hub",
	})

	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "callback")
	if err != nil {
		t.Fatal(err)
	}

}
