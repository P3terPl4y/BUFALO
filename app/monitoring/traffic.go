package monitoring

import "sync/atomic"

var trafficCounters struct{ limited, banned, observed, storeErrors, capacity, transport, body, readTimeout atomic.Uint64 }
var trafficSettings atomic.Pointer[TrafficSettings]

type TrafficSettings struct {
	Mode            string `json:"mode"`
	Rate            int64  `json:"rate"`
	Burst           int64  `json:"burst"`
	StaticRate      int64  `json:"static_rate"`
	ShortBanSeconds int64  `json:"short_ban_seconds"`
	LongBanSeconds  int64  `json:"long_ban_seconds"`
	RequestCapacity int    `json:"request_capacity"`
	AuthCapacity    int    `json:"auth_capacity"`
}

func ConfigureTraffic(settings TrafficSettings) { trafficSettings.Store(&settings) }

func RecordTraffic(outcome string) {
	switch outcome {
	case "limited":
		trafficCounters.limited.Add(1)
	case "banned":
		trafficCounters.banned.Add(1)
	case "observed":
		trafficCounters.observed.Add(1)
	case "store_error":
		trafficCounters.storeErrors.Add(1)
	case "transport_rejected":
		trafficCounters.transport.Add(1)
	case "body_rejected":
		trafficCounters.body.Add(1)
	case "read_timeout":
		trafficCounters.readTimeout.Add(1)
	case "session_capacity":
		trafficCounters.capacity.Add(1)
	case "capacity":
		trafficCounters.capacity.Add(1)
	}
}

type TrafficSnapshot struct {
	RecentRejections  []AttackEvent        `json:"recent_rejections"`
	SecurityScan      SecurityScanSnapshot `json:"security_scan"`
	TransportRejected uint64               `json:"transport_rejected"`
	BodyRejected      uint64               `json:"body_rejected"`
	ReadTimeouts      uint64               `json:"read_timeouts"`
	Policy            TrafficSettings      `json:"policy"`
	Limited           uint64               `json:"limited"`
	Banned            uint64               `json:"banned"`
	Observed          uint64               `json:"observed"`
	StoreErrors       uint64               `json:"store_errors"`
	CapacityRejected  uint64               `json:"capacity_rejected"`
}

func Traffic() TrafficSnapshot {
	settings := TrafficSettings{Mode: "unconfigured"}
	if configured := trafficSettings.Load(); configured != nil {
		settings = *configured
	}
	return TrafficSnapshot{RecentRejections: RecentAttacks(), SecurityScan: SecurityScan(), TransportRejected: trafficCounters.transport.Load(), BodyRejected: trafficCounters.body.Load(), ReadTimeouts: trafficCounters.readTimeout.Load(), Policy: settings, Limited: trafficCounters.limited.Load(), Banned: trafficCounters.banned.Load(), Observed: trafficCounters.observed.Load(), StoreErrors: trafficCounters.storeErrors.Load(), CapacityRejected: trafficCounters.capacity.Load()}
}
