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
