package sql

// import (
// 	"fmt"
// 	"testing"
// )

// func TestSQL_GetInactive(t *testing.T) {
// 	sql, err := New(NewConfig())
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer func() {
// 		if err = sql.Shutdown(); err != nil {
// 			t.Fatal(err)
// 		}
// 	}()

// 	topicsToHubs := make(map[string]string)

// 	for idx := 1000; idx < 2000; idx++ {
// 		topicsToHubs[fmt.Sprintf("topic_num%d", idx)] = fmt.Sprintf("hub_num%d", idx)
// 	}

// 	err = sql.Index(topicsToHubs)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	subs, last, err := sql.GetInactive(50, "topic_num1000", "hub_num1000")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if last {
// 		t.Fatal("Reported last, when it wasn't", len(subs.Subscriptions))
// 	}

// 	if len(subs.Subscriptions) != 50 {
// 		t.Fatalf("Returned %d instead of exactly 50 results", len(subs.Subscriptions))
// 	}

// 	if subs.Subscriptions[0].Topic != "topic_num1001" {
// 		t.Fatalf("Started with {%s} instead of {%s}", subs.Subscriptions[0].Topic, "topic_num1001")
// 	}

// 	if subs.Subscriptions[49].Topic != "topic_num1050" {
// 		t.Fatalf("Ended with {%s} instead of {%s}", subs.Subscriptions[49].Topic, "topic_num1050")
// 	}

// }
