package monitoring

import (
	"sync"
	"time"
)

// OnlineWindow is the maximum age of a user's authenticated activity to count
// as recently connected. Presence is process-local and resets on restart.
const OnlineWindow = 2 * time.Minute

var presence = struct {
	sync.Mutex
	users      map[uint]time.Time
	lastPruned time.Time
}{users: make(map[uint]time.Time)}

func RecordUserActivity(userID uint, at time.Time) {
	if userID == 0 {
		return
	}
	presence.Lock()
	presence.users[userID] = at
	if presence.lastPruned.IsZero() || at.Sub(presence.lastPruned) >= OnlineWindow {
		prunePresence(at)
		presence.lastPruned = at
	}
	presence.Unlock()
}

func OnlineUserCount(now time.Time) int {
	presence.Lock()
	defer presence.Unlock()

	count := 0
	prunePresence(now)
	for range presence.users {
		count++
	}
	return count
}

func ActiveOnlineUserCount(activeUserIDs []uint, now time.Time) int64 {
	presence.Lock()
	defer presence.Unlock()
	prunePresence(now)
	var count int64
	seen := make(map[uint]struct{}, len(activeUserIDs))
	for _, userID := range activeUserIDs {
		if _, duplicate := seen[userID]; duplicate {
			continue
		}
		seen[userID] = struct{}{}
		if lastSeen, ok := presence.users[userID]; ok && !lastSeen.After(now) && now.Sub(lastSeen) <= OnlineWindow {
			count++
		}
	}
	return count
}

func prunePresence(now time.Time) {
	for userID, lastSeen := range presence.users {
		if now.Sub(lastSeen) > OnlineWindow || lastSeen.After(now) {
			delete(presence.users, userID)
		}
	}
}
