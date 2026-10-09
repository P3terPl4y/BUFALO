package monitoring

import (
	"fmt"
	"testing"
)

func TestRejectionHistoryIsBoundedAndDoesNotExposeIP(t *testing.T) {
	for n := 0; n < AttackHistoryLimit+10; n++ {
		RecordAttack("capacity", 429, fmt.Sprintf("198.18.0.%d", n))
	}
	events := RecentAttacks()
	if len(events) != AttackHistoryLimit {
		t.Fatal(len(events))
	}
	for _, event := range events {
		if len(event.Identity) != 16 || event.Status != 429 || event.Count < 1 {
			t.Fatal(event)
		}
	}
	events[0].Identity = "modified"
	if RecentAttacks()[0].Identity == "modified" {
		t.Fatal("snapshot aliases history")
	}
}
