package controllers

import (
	"testing"

	"goravel/app/models"
)

func TestCanEditEmpresaOnlyOwnerOrPlatformAdmin(t *testing.T) {
	ownerID, otherID := uint(41), uint(42)
	company := &models.Empresa{OwnerID: &ownerID}
	if !canEditEmpresa(company, ownerID, "publicador") {
		t.Fatal("company owner must be allowed to edit")
	}
	if canEditEmpresa(company, otherID, "publicador") {
		t.Fatal("non-owner publicador must not be allowed to edit")
	}
	if canEditEmpresa(company, otherID, "chofer") {
		t.Fatal("non-owner chofer must not be allowed to edit")
	}
	if canEditEmpresa(&models.Empresa{}, ownerID, "publicador") {
		t.Fatal("company without an owner must not be editable by a regular user")
	}
	if !canEditEmpresa(company, otherID, "admin") {
		t.Fatal("platform admin must retain company administration access")
	}
}
