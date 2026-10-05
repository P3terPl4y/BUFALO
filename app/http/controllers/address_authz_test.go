package controllers

import (
	"goravel/app/models"
	"testing"
)

func TestCanManageAddress(t *testing.T) {
	owner, other := uint(7), uint(8)
	cases := []struct {
		name, role string
		id         uint
		addr       *models.Direccion
		want       bool
	}{
		{"owner", "chofer", owner, &models.Direccion{OwnerID: &owner}, true},
		{"other user", "publicador", other, &models.Direccion{OwnerID: &owner}, false},
		{"legacy without owner", "admin", owner, &models.Direccion{}, true},
		{"unowned address", "publicador", owner, &models.Direccion{}, false},
		{"nil address", "admin", owner, nil, false},
		{"zero user", "admin", 0, &models.Direccion{OwnerID: &owner}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canManageAddress(tc.addr, tc.id, tc.role); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestCompanyAddressOwnershipAndOptionalSelection(t *testing.T) {
	owner, other, addressID := uint(7), uint(8), uint(42)
	for _, tc := range []struct {
		name string
		addr *models.Direccion
		uid  uint
		role string
		want bool
	}{
		{"optional", nil, owner, "publicador", true},
		{"owned", &models.Direccion{OwnerID: &owner}, owner, "chofer", true},
		{"another user's address", &models.Direccion{OwnerID: &owner}, other, "publicador", false},
		{"admin", &models.Direccion{OwnerID: &owner}, other, "admin", true},
		{"legacy unowned", &models.Direccion{}, owner, "publicador", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := canUseCompanyAddress(tc.addr, tc.uid, tc.role); got != tc.want {
				t.Fatalf("canUseCompanyAddress() = %v, want %v", got, tc.want)
			}
		})
	}
	if got := optionalAddressID(&addressID); got != &addressID {
		t.Fatal("nonzero address ID should be retained")
	}
	zero := uint(0)
	if got := optionalAddressID(&zero); got != nil {
		t.Fatal("zero address ID should normalize to nil")
	}
}

func TestValidCoordinatePair(t *testing.T) {
	lat, lon := 0.0, 0.0
	badLat, badLon := 91.0, 181.0
	for _, tc := range []struct {
		name string
		lat  *float64
		lon  *float64
		want bool
	}{
		{"coordinates optional", nil, nil, true},
		{"equator and prime meridian are valid", &lat, &lon, true},
		{"latitude only", &lat, nil, false},
		{"longitude only", nil, &lon, false},
		{"out of range", &badLat, &badLon, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := validCoordinatePair(tc.lat, tc.lon); got != tc.want {
				t.Fatalf("validCoordinatePair() = %v, want %v", got, tc.want)
			}
		})
	}
}
