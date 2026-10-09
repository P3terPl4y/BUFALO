package monitoring

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"sync"
	"time"
)

const AttackHistoryLimit = 128

type AttackEvent struct {
	At       time.Time `json:"at"`
	Outcome  string    `json:"outcome"`
	Status   int       `json:"status"`
	Identity string    `json:"identity"`
	Count    uint64    `json:"count"`
}

var attackHistory struct {
	sync.Mutex
	events      [AttackHistoryLimit]AttackEvent
	next, count int
	lastLog     time.Time
	sinceLog    uint64
}

// RecordAttack stores bounded rejection evidence without raw IPs, URLs,
// credentials or bodies. Journal output is aggregated at most once per second.
// Rejections indicate suspicious or excessive traffic, not proof of an attack.
func RecordAttack(outcome string, status int, ip string) {
	mac := hmac.New(sha256.New, []byte(os.Getenv("APP_KEY")))
	_, _ = mac.Write([]byte(ip))
	identity := hex.EncodeToString(mac.Sum(nil)[:8])
	now := time.Now().UTC()
	attackHistory.Lock()
	defer attackHistory.Unlock()
	previous := (attackHistory.next + AttackHistoryLimit - 1) % AttackHistoryLimit
	if attackHistory.count > 0 && attackHistory.events[previous].Outcome == outcome && attackHistory.events[previous].Identity == identity && now.Sub(attackHistory.events[previous].At) < time.Second {
		attackHistory.events[previous].Count++
	} else {
		attackHistory.events[attackHistory.next] = AttackEvent{now, outcome, status, identity, 1}
		attackHistory.next = (attackHistory.next + 1) % AttackHistoryLimit
		attackHistory.count = min(AttackHistoryLimit, attackHistory.count+1)
	}
	attackHistory.sinceLog++
	if now.Sub(attackHistory.lastLog) >= time.Second {
		log.Printf("security_rejections count=%d latest_outcome=%s status=%d identity=%s", attackHistory.sinceLog, outcome, status, identity)
		attackHistory.sinceLog = 0
		attackHistory.lastLog = now
	}
}
func RecentAttacks() []AttackEvent {
	attackHistory.Lock()
	defer attackHistory.Unlock()
	events := make([]AttackEvent, 0, attackHistory.count)
	for n := 0; n < attackHistory.count; n++ {
		events = append(events, attackHistory.events[(attackHistory.next+AttackHistoryLimit-1-n)%AttackHistoryLimit])
	}
	return events
}
