package sql

import (
	"context"
	"testing"
	"time"

	"github.com/mattn/go-sqlite3"
)

/*
	# Test Cases

	## Valid Cases

	1. Link indexed, but has no callback assigned
	2. New callback, for a hot link
	3. New callback, for a cold link
	4. Recycled callback, from an overwritten link

	## Error Cases

	1. Unindexed link
	2. Recycled callback, from hot link
	3. Recycled callback, from cold but recorded link
	4. Double-dipping
*/

func TestSQL_NewCallback_ValidCases(t *testing.T) {
	sqlStor, err := New(NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = sqlStor.Shutdown(); err != nil {
			t.Fatal(err)
		}
	}()

	// 1. Indexed, not yet active
	sqlStor.IndexOffer(map[string]string{
		"topic": "hub",
	})

	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "cb1")
	if err != nil {
		t.Fatal(err)
	}

	// 2. New callback endpoint, for a link that is already hot
	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "cb_new")
	if err != nil {
		t.Fatal(err)
	}

	// 3. New callback for link that was previously active, but is not currently
	err = sqlStor.Invalidate(context.Background(), "cb_new", "whatever")
	if err != nil {
		t.Fatal(err)
	}

	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "cb2") // row with cb1 replaced
	if err != nil {
		t.Fatal(err)
	}

	// 4. Reusing an old (erased) callback on a separate link
	sqlStor.IndexOffer(map[string]string{
		"otherTopic": "newHub",
	})

	err = sqlStor.NewCallback(context.Background(), "otherTopic", "newHub", "cb_new")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSQL_NewCallback_ErrorCases(t *testing.T) {
	sqlStor, err := New(NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = sqlStor.Shutdown(); err != nil {
			t.Fatal(err)
		}
	}()

	// 1. Unindexed link
	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "cb")
	if sqliteErr, ok := err.(sqlite3.Error); !ok || sqliteErr.ExtendedCode != sqlite3.ErrConstraintForeignKey {
		t.Fatal(err)
	}

	// 2. Recycled callback, from hot link
	sqlStor.IndexOffer(map[string]string{
		"topic": "hub",
	})
	sqlStor.IndexOffer(map[string]string{
		"topic2": "hub2",
	})
	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "cb")
	if err != nil {
		t.Fatal(err)
	}
	err = sqlStor.ExtendLease(context.Background(), "cb", time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}

	err = sqlStor.NewCallback(context.Background(), "topic2", "hub2", "cb")
	if sqliteErr, ok := err.(sqlite3.Error); !ok || sqliteErr.ExtendedCode != sqlite3.ErrConstraintPrimaryKey {
		t.Fatal(err)
	}

	// 3. Recycled callback, from cold but recorded link
	err = sqlStor.Invalidate(context.Background(), "cb", "good reason")
	if err != nil {
		t.Fatal(err)
	}

	err = sqlStor.NewCallback(context.Background(), "topic", "hub", "cb")
	if sqliteErr, ok := err.(sqlite3.Error); !ok || sqliteErr.ExtendedCode != sqlite3.ErrConstraintPrimaryKey {
		t.Fatal(err)
	}

	// 4. Double-dipping
	err = sqlStor.NewCallback(context.Background(), "topic2", "hub2", "cb2")
	if err != nil {
		t.Fatal(err)
	}

	err = sqlStor.NewCallback(context.Background(), "topic2", "hub2", "cb2")
	if sqliteErr, ok := err.(sqlite3.Error); !ok || sqliteErr.ExtendedCode != sqlite3.ErrConstraintPrimaryKey {
		t.Fatal(err)
	}
}
