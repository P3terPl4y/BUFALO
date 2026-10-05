package monitoring

import (
	"testing"
	"time"
)

func TestPresenceCountsUniqueRecentUsersAndExpiresThem(t *testing.T) {
	base := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	RecordUserActivity(810001, base)
	RecordUserActivity(810001, base.Add(30*time.Second))
	RecordUserActivity(810002, base.Add(-OnlineWindow-time.Second))
	RecordUserActivity(810003, base.Add(20*time.Second))
	RecordUserActivity(0, base)

	if got := OnlineUserCount(base.Add(31 * time.Second)); got != 2 {
		t.Fatalf("online count = %d, want 2", got)
	}
	if got := ActiveOnlineUserCount([]uint{810001, 810003, 810003}, base.Add(31*time.Second)); got != 2 {
		t.Fatalf("active connected count = %d, want 2 (unique active accounts)", got)
	}
	if got := ActiveOnlineUserCount([]uint{810001}, base.Add(31*time.Second)); got != 1 {
		t.Fatalf("disabled or unlisted recent user count = %d, want 1", got)
	}
	if got := OnlineUserCount(base.Add(3 * time.Minute)); got != 0 {
		t.Fatalf("expired online count = %d, want 0", got)
	}
}
