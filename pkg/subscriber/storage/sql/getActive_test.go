package sql

import (
	"context"
	"fmt"
	"testing"
	"time"
)

/*
	# Paging Tests

	# Correctly Identifying Active

	1. NewCallback fails
	2. NewCallback + ExtendLease passes
	3. NewCallback + ExtendLease + Time passes fails
	4. NewCallback + ExtendLease + Invalidate fails
	5. NewCallback + Invalidate fails
*/

func TestSQL_GetActive_PagingTest(t *testing.T) {
	sqlStor, err := New(NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = sqlStor.Shutdown(); err != nil {
			t.Fatal(err)
		}
	}()

	topicsToHubs := make(map[string]string)
	for idx := 1000; idx < 2000; idx++ {
		topicsToHubs[fmt.Sprintf("topic_num%d", idx)] = fmt.Sprintf("hub_num%d", idx)
	}
	err = sqlStor.IndexOffer(topicsToHubs)
	if err != nil {
		t.Fatal(err)
	}

	// Launch, mark active
	for idx := 1000; idx < 2000; idx++ {
		err := sqlStor.NewCallback(context.Background(), fmt.Sprintf("topic_num%d", idx), fmt.Sprintf("hub_num%d", idx), fmt.Sprintf("cb_num%d", idx))
		if err != nil {
			t.Fatal(err)
		}

		err = sqlStor.ExtendLease(context.Background(), fmt.Sprintf("cb_num%d", idx), time.Now().Add(10*time.Second))
		if err != nil {
			t.Fatal(err)
		}
	}

	// Get active ones
	subs, last, err := sqlStor.GetActive(50, "topic_num1000", "hub_num1000")
	if err != nil {
		t.Fatal(err)
	}

	// assertions...
	if last {
		t.Fatal("Reported last, when it wasn't")
	}

	if len(subs.Subscriptions) != 50 {
		t.Fatalf("Returned %d instead of exactly 50 results", len(subs.Subscriptions))
	}

	if subs.Subscriptions[0].Topic != "topic_num1001" {
		t.Fatalf("Started with {%s} instead of {%s}", subs.Subscriptions[0].Topic, "topic_num1001")
	}

	if subs.Subscriptions[49].Topic != "topic_num1050" {
		t.Fatalf("Ended with {%s} instead of {%s}", subs.Subscriptions[49].Topic, "topic_num1050")
	}
}

func TestSQL_GetActive_Identification(t *testing.T) {
	sqlStor, err := New(NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = sqlStor.Shutdown(); err != nil {
			t.Fatal(err)
		}
	}()

	err = sqlStor.IndexOffer(map[string]string{
		"topic": "hub",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 1. NewCallback fails
	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "callback")
	if err != nil {
		t.Fatal(err)
	}

	subs, last, err := sqlStor.GetActive(10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !last {
		t.Fatal("Should have been last page")
	}
	if len(subs.Subscriptions) != 0 {
		t.Fatalf("Expected no active subs, instead: %+v", subs.Subscriptions)
	}

	// 2. NewCallback + ExtendLease passes
	err = sqlStor.ExtendLease(context.Background(), "callback", time.Now().Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	subs, last, err = sqlStor.GetActive(10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !last {
		t.Fatal("Should have been last page")
	}
	if len(subs.Subscriptions) != 1 {
		t.Fatalf("Should be 1 active subscription, instead: %+v", subs.Subscriptions)
	}

	// 3. NewCallback + ExtendLease + Time passes fails
	time.Sleep(time.Second)
	subs, last, err = sqlStor.GetActive(10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !last {
		t.Fatal("Should have been last page")
	}
	if len(subs.Subscriptions) != 0 {
		t.Fatalf("Expected no active subs, instead: %+v", subs.Subscriptions)
	}

	// 4. NewCallback + ExtendLease + Invalidate fails
	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "newCallback")
	if err != nil {
		t.Fatal(err)
	}

	err = sqlStor.ExtendLease(context.Background(), "newCallback", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	err = sqlStor.Invalidate(context.Background(), "newCallback", "denied")
	if err != nil {
		t.Fatal(err)
	}

	subs, last, err = sqlStor.GetActive(10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !last {
		t.Fatal("Should have been last page")
	}
	if len(subs.Subscriptions) != 0 {
		t.Fatalf("Expected no active subs, instead: %+v", subs.Subscriptions)
	}

	// 5. NewCallback + Invalidate fails
	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "newestCallback")
	if err != nil {
		t.Fatal(err)
	}

	err = sqlStor.Invalidate(context.Background(), "newestCallback", "denied")
	if err != nil {
		t.Fatal(err)
	}

	subs, last, err = sqlStor.GetActive(10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !last {
		t.Fatal("Should have been last page")
	}
	if len(subs.Subscriptions) != 0 {
		t.Fatalf("Expected no active subs, instead: %+v", subs.Subscriptions)
	}
}
