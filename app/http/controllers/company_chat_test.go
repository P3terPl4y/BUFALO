package controllers

import (
	"strings"
	"testing"
	"time"

	"goravel/app/models"
)

func TestCompanyChatResponseRedactsModeratedMessageAndPrivateFields(t *testing.T) {
	moderatedAt := time.Now()
	choferProfileID := uint(14)
	rows := []models.EmpresaChatMensaje{
		{ID: 7, UserID: 4, Mensaje: "mensaje privado", CreatedAt: time.Now(), User: &models.User{Role: "chofer", ChoferID: &choferProfileID, Name: "Miembro", Email: "private@example.test", ProfilePhoto: "/storage/avatar.webp"}},
		{ID: 8, UserID: 5, Mensaje: "contenido moderado", ModeradoEn: &moderatedAt, User: &models.User{Name: "Otro miembro", Email: "hidden@example.test"}},
	}
	response := companyChatMessagesResponse(rows)
	if len(response) != 2 || response[0].Message != "mensaje privado" || response[0].UserName != "Miembro" || response[0].UserPhoto != "/storage/avatar.webp" || response[0].ProfileURL != "/choferes/14" {
		t.Fatalf("active message response incorrect: %+v", response)
	}
	if !response[1].Moderated || strings.Contains(response[1].Message, "contenido moderado") {
		t.Fatalf("moderated text was not redacted: %+v", response[1])
	}
	encoded := response[0].UserName + response[0].Message + response[1].UserName + response[1].Message
	if strings.Contains(encoded, "@example.test") {
		t.Fatalf("message response exposed email: %s", encoded)
	}
}
