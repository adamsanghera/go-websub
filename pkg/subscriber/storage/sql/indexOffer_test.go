package sql

import (
	"testing"
)

func TestSQL_IndexNormal(t *testing.T) {
	man, err := New(NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = man.Shutdown(); err != nil {
			t.Fatal(err)
		}
	}()

	err = man.IndexOffer(
		map[string]string{
			"abc.com/topic": "abc.com/topic_hub",
		})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSQL_IndexNullElements(t *testing.T) {
	man, err := New(NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = man.Shutdown(); err != nil {
			t.Fatal(err)
		}
	}()

	err = man.IndexOffer(
		map[string]string{
			"abc.com/topic": "",
		})
	if err == nil {
		t.Fatal("Was able to index a null hub url")
	}

	err = man.IndexOffer(
		map[string]string{
			"": "abc.com/topic_hub_url",
		})
	if err == nil {
		t.Fatal("Was able to index a null topic url")
	}
}
