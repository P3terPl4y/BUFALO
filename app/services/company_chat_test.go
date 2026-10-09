package services_test

import (
	"errors"
	"testing"

	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/tests"
)

func seedCompanyChatActors(t *testing.T) (*models.Empresa, *models.User, *models.User, *models.User) {
	t.Helper()
	owner := tests.NewUserPublicador("chat-owner@test.local")
	if err := facades.Orm().Query().Create(owner); err != nil {
		t.Fatal(err)
	}
	company := &models.Empresa{Tipo: models.TipoBroker, NombreLegal: "Chat broker", Estado: models.EmpresaActiva, OwnerID: &owner.ID}
	if err := facades.Orm().Query().Create(company); err != nil {
		t.Fatal(err)
	}
	owner.EmpresaID = &company.ID
	if _, err := facades.Orm().Query().Model(&models.User{}).Where("id = ?", owner.ID).Update("empresa_id", company.ID); err != nil {
		t.Fatal(err)
	}

	member := tests.NewUserChofer("chat-member@test.local")
	if err := facades.Orm().Query().Create(member); err != nil {
		t.Fatal(err)
	}
	profile := tests.NewChoferProfile()
	profile.UserID, profile.EmpresaID = member.ID, company.ID
	profile.NumeroLicencia = "CHAT-LIC-1"
	if err := facades.Orm().Query().Create(profile); err != nil {
		t.Fatal(err)
	}

	outsider := tests.NewUserPublicador("chat-outsider@test.local")
	if err := facades.Orm().Query().Create(outsider); err != nil {
		t.Fatal(err)
	}
	return company, owner, member, outsider
}

func TestCompanyChatMembershipMessagesAndModeration(t *testing.T) {
	tests.ResetDB(t)
	company, owner, member, outsider := seedCompanyChatActors(t)

	ownerAccess, err := services.GetCompanyChatAccess(company.ID, owner.ID)
	if err != nil || !ownerAccess.Member || !ownerAccess.Moderator {
		t.Fatalf("company owner should moderate chat: access=%+v err=%v", ownerAccess, err)
	}
	memberAccess, err := services.GetCompanyChatAccess(company.ID, member.ID)
	if err != nil || !memberAccess.Member || memberAccess.Moderator {
		t.Fatalf("company member should be able to chat without moderation: access=%+v err=%v", memberAccess, err)
	}
	if _, err := services.GetCompanyChatAccess(company.ID, outsider.ID); !errors.Is(err, services.ErrCompanyChatForbidden) {
		t.Fatalf("outsider got company chat access: %v", err)
	}

	if err := services.SendCompanyChatMessage(company.ID, member.ID, "  Hola equipo  "); err != nil {
		t.Fatalf("member could not send message: %v", err)
	}
	rows, err := services.ListCompanyChatMessages(company.ID, owner.ID)
	if err != nil || len(rows) != 1 || rows[0].Mensaje != "Hola equipo" || rows[0].UserID != member.ID || rows[0].User == nil {
		t.Fatalf("message was not persisted with author: rows=%+v err=%v", rows, err)
	}
	messageID := rows[0].ID
	if err := services.ModerateCompanyChatMessage(company.ID, messageID, member.ID); !errors.Is(err, services.ErrCompanyChatForbidden) {
		t.Fatalf("non-owner moderated message: %v", err)
	}
	if err := services.ModerateCompanyChatMessage(company.ID, messageID, owner.ID); err != nil {
		t.Fatalf("company owner could not moderate message: %v", err)
	}
	rows, err = services.ListCompanyChatMessages(company.ID, member.ID)
	if err != nil || len(rows) != 1 || rows[0].ModeradoEn == nil {
		t.Fatalf("moderation was not persisted for members: rows=%+v err=%v", rows, err)
	}
	if err := services.ModerateCompanyChatMessage(company.ID, messageID, owner.ID); !errors.Is(err, services.ErrCompanyChatMessageMissing) {
		t.Fatalf("already moderated message was accepted again: %v", err)
	}
	if err := services.SendCompanyChatMessage(company.ID, outsider.ID, "mensaje fuera de empresa"); !errors.Is(err, services.ErrCompanyChatForbidden) {
		t.Fatalf("outsider could send message: %v", err)
	}
}

func TestCompanyChatRejectsEmptyAndOversizedMessages(t *testing.T) {
	tests.ResetDB(t)
	company, owner, _, _ := seedCompanyChatActors(t)
	if err := services.SendCompanyChatMessage(company.ID, owner.ID, " \n\t "); !errors.Is(err, services.ErrCompanyChatMessage) {
		t.Fatalf("empty message accepted: %v", err)
	}
	if err := services.SendCompanyChatMessage(company.ID, owner.ID, string(make([]rune, 1001))); !errors.Is(err, services.ErrCompanyChatMessage) {
		t.Fatalf("oversized message accepted: %v", err)
	}
}

func TestCompanyMembershipReturnsActionableConflictReasons(t *testing.T) {
	tests.ResetDB(t)
	availableCarrier := tests.SeedEmpresa(t, "carrier", "Carrier disponible")
	chofer := tests.NewUserChofer("membership-request@test.local")
	if err := facades.Orm().Query().Create(chofer); err != nil {
		t.Fatal(err)
	}
	profile := tests.NewChoferProfile()
	profile.UserID = chofer.ID
	profile.NumeroLicencia = "MEMBERSHIP-LIC-1"
	if err := facades.Orm().Query().Create(profile); err != nil {
		t.Fatal(err)
	}
	if eligible, err := services.CanRequestCompanyMembership(chofer.ID, availableCarrier.ID); err != nil || !eligible {
		t.Fatalf("eligible chofer should see the request action: eligible=%t err=%v", eligible, err)
	}
	if err := services.RequestCompanyMembership(chofer.ID, availableCarrier.ID); err != nil {
		t.Fatalf("first eligible membership request failed: %v", err)
	}
	if eligible, err := services.CanRequestCompanyMembership(chofer.ID, availableCarrier.ID); err != nil || eligible {
		t.Fatalf("pending user must not see another request action: eligible=%t err=%v", eligible, err)
	}
	if err := services.RequestCompanyMembership(chofer.ID, availableCarrier.ID); !errors.Is(err, services.ErrMembershipDenied) || !errors.Is(err, services.ErrMembershipPending) {
		t.Fatalf("duplicate request should explain the pending conflict: %v", err)
	}

	broker := tests.SeedEmpresa(t, "broker", "Tipo incorrecto")
	other := tests.NewUserChofer("membership-type@test.local")
	if err := facades.Orm().Query().Create(other); err != nil {
		t.Fatal(err)
	}
	otherProfile := tests.NewChoferProfile()
	otherProfile.UserID = other.ID
	otherProfile.NumeroLicencia = "MEMBERSHIP-LIC-2"
	if err := facades.Orm().Query().Create(otherProfile); err != nil {
		t.Fatal(err)
	}
	if err := services.RequestCompanyMembership(other.ID, broker.ID); !errors.Is(err, services.ErrMembershipWrongCompanyType) {
		t.Fatalf("wrong company type should have a distinct reason: %v", err)
	}

	assigned := tests.SeedEmpresa(t, "carrier", "Carrier asignado")
	assignedUser := tests.NewUserChofer("membership-assigned@test.local")
	if err := facades.Orm().Query().Create(assignedUser); err != nil {
		t.Fatal(err)
	}
	assignedProfile := tests.NewChoferProfile()
	assignedProfile.UserID, assignedProfile.EmpresaID = assignedUser.ID, assigned.ID
	assignedProfile.NumeroLicencia = "MEMBERSHIP-LIC-3"
	if err := facades.Orm().Query().Create(assignedProfile); err != nil {
		t.Fatal(err)
	}
	if err := services.RequestCompanyMembership(assignedUser.ID, availableCarrier.ID); !errors.Is(err, services.ErrMembershipAlreadyAssociated) {
		t.Fatalf("already-associated profile should have a distinct reason: %v", err)
	}
}

func TestCompanyDirectoryShowsMembershipsAndHidesThemFromBrowseResults(t *testing.T) {
	tests.ResetDB(t)
	memberCompany := tests.SeedEmpresa(t, "carrier", "Carrier de Chofer")
	availableCompany := tests.SeedEmpresa(t, "carrier", "Carrier disponible")
	owner := tests.NewUserPublicador("directory-owner@test.local")
	if err := facades.Orm().Query().Create(owner); err != nil {
		t.Fatal(err)
	}
	ownedCompany := &models.Empresa{Tipo: models.TipoBroker, NombreLegal: "Broker propio", Estado: models.EmpresaActiva, OwnerID: &owner.ID}
	if err := facades.Orm().Query().Create(ownedCompany); err != nil {
		t.Fatal(err)
	}

	member := tests.NewUserChofer("directory-member@test.local")
	if err := facades.Orm().Query().Create(member); err != nil {
		t.Fatal(err)
	}
	profile := tests.NewChoferProfile()
	profile.UserID, profile.EmpresaID = member.ID, memberCompany.ID
	profile.NumeroLicencia = "DIRECTORY-LIC-1"
	if err := facades.Orm().Query().Create(profile); err != nil {
		t.Fatal(err)
	}

	service := services.NewEmpresaService()
	myCompanies, _, err := service.GetCompaniesForUser(member.ID, 1, 10)
	if err != nil || len(myCompanies) != 1 || myCompanies[0].ID != memberCompany.ID {
		t.Fatalf("member profile association missing from workspace: companies=%+v err=%v", myCompanies, err)
	}
	available, _, err := service.GetAvailableCompanies(member.ID, map[string]string{"role": "chofer"}, 1, 10)
	if err != nil || len(available) != 1 || available[0].ID != availableCompany.ID {
		t.Fatalf("directory should only return unassociated compatible companies: companies=%+v err=%v", available, err)
	}
	ownerCompanies, _, err := service.GetCompaniesForUser(owner.ID, 1, 10)
	if err != nil || len(ownerCompanies) != 1 || ownerCompanies[0].ID != ownedCompany.ID {
		t.Fatalf("owner company missing from workspace: companies=%+v err=%v", ownerCompanies, err)
	}
}

func TestCompanyOwnerCanRejectAndUserCanReapplyThenJoin(t *testing.T) {
	tests.ResetDB(t)
	owner := tests.NewUserPublicador("membership-owner@test.local")
	if err := facades.Orm().Query().Create(owner); err != nil {
		t.Fatal(err)
	}
	company := &models.Empresa{Tipo: models.TipoCarrier, NombreLegal: "Owner carrier", Estado: models.EmpresaActiva, OwnerID: &owner.ID}
	if err := facades.Orm().Query().Create(company); err != nil {
		t.Fatal(err)
	}
	if err := services.RequestCompanyMembership(owner.ID, company.ID); !errors.Is(err, services.ErrMembershipCompanyOwner) {
		t.Fatalf("company owner must not be able to request affiliation: %v", err)
	}
	outsider := tests.NewUserPublicador("membership-outsider@test.local")
	if err := facades.Orm().Query().Create(outsider); err != nil {
		t.Fatal(err)
	}
	user := tests.NewUserChofer("membership-reapply@test.local")
	if err := facades.Orm().Query().Create(user); err != nil {
		t.Fatal(err)
	}
	profile := tests.NewChoferProfile()
	profile.UserID = user.ID
	profile.NumeroLicencia = "REAPPLY-1"
	if err := facades.Orm().Query().Create(profile); err != nil {
		t.Fatal(err)
	}
	if err := services.RequestCompanyMembership(user.ID, company.ID); err != nil {
		t.Fatal(err)
	}
	var request services.CompanyMembershipRequest
	var pending []services.CompanyMembershipRequest
	if err := facades.Orm().Query().Where("user_id = ?", user.ID).Where("status = ?", "pending").Find(&pending); err != nil || len(pending) != 1 {
		t.Fatalf("expected a single pending request: %+v err=%v", pending, err)
	}
	request = pending[0]
	if err := services.DecideCompanyMembership(outsider.ID, request.ID, false); !errors.Is(err, services.ErrMembershipDenied) {
		t.Fatalf("non-owner decided request: %v", err)
	}
	if err := services.DecideCompanyMembership(owner.ID, request.ID, false); err != nil {
		t.Fatalf("owner could not reject: %v", err)
	}
	var userAfter models.User
	if err := facades.Orm().Query().Where("id = ?", user.ID).First(&userAfter); err != nil {
		t.Fatal(err)
	}
	if userAfter.EmpresaID != nil {
		t.Fatal("rejected user must stay unaffiliated")
	}
	if err := services.RequestCompanyMembership(user.ID, company.ID); err != nil {
		t.Fatalf("user could not reapply after rejection: %v", err)
	}
	pending = nil
	if err := facades.Orm().Query().Where("user_id = ?", user.ID).Where("status = ?", "pending").Find(&pending); err != nil || len(pending) != 1 {
		t.Fatalf("expected a single reapplication: %+v err=%v", pending, err)
	}
	request = pending[0]
	if err := services.DecideCompanyMembership(owner.ID, request.ID, true); err != nil {
		t.Fatalf("owner could not approve: %v (request=%+v)", err, request)
	}
	if err := facades.Orm().Query().Where("id = ?", user.ID).First(&userAfter); err != nil {
		t.Fatal(err)
	}
	if userAfter.EmpresaID == nil || *userAfter.EmpresaID != company.ID {
		t.Fatal("approved user was not associated")
	}
}

func TestUserNotificationAndEmailOutboxCommitTogether(t *testing.T) {
	tests.ResetDB(t)
	user := tests.NewUserChofer("notification@test.local")
	if err := facades.Orm().Query().Create(user); err != nil {
		t.Fatal(err)
	}
	actor := tests.NewUserPublicador("notification-actor@test.local")
	if err := facades.Orm().Query().Create(actor); err != nil {
		t.Fatal(err)
	}
	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		t.Fatal(err)
	}
	if err := services.CreateUserNotification(tx, user.ID, "load_assigned", "Carga asignada", "Se te asignó una carga.", "load", 42, true, actor.ID); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	count, err := facades.Orm().Query().Model(&models.UserNotification{}).Where("user_id = ?", user.ID).Count()
	if err != nil || count != 1 {
		t.Fatalf("inbox row missing: count=%d err=%v", count, err)
	}
	rows, _, err := services.ListUserNotifications(user.ID, 1)
	if err != nil || len(rows) != 1 || rows[0].Actor == nil || rows[0].Actor.ID != actor.ID || rows[0].Actor.Name != actor.Name {
		t.Fatalf("notification actor was not retained and loaded: rows=%+v err=%v", rows, err)
	}
	unread, err := services.UnreadUserNotificationCount(user.ID)
	if err != nil || unread != 1 {
		t.Fatalf("unread count incorrect: count=%d err=%v", unread, err)
	}
	var outboxCount int64
	if err := facades.DB().Select(&outboxCount, "SELECT count(*) FROM notification_email_outbox WHERE user_id = $1 AND sent_at IS NULL", user.ID); err != nil || outboxCount != 1 {
		t.Fatalf("email outbox row missing: count=%d err=%v", outboxCount, err)
	}
}
