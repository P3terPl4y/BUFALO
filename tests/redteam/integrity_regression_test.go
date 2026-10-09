package redteam

import (
	"context"
	"errors"
	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/tests"
	"io"
	"net/url"
	"strings"
	"sync"
	"testing"
)

func TestLoadPrivacyIntegrityAndInterest(t *testing.T) {
	tests.ResetDB(t)
	fx := seedFixture(t, 6)
	var owner, driver principal
	for _, p := range fx.people {
		if p.user.ID == fx.ownerID {
			owner = p
		}
		if p.role == "chofer" {
			driver = p
		}
	}
	_, err := facades.Orm().Query().Model(&models.Carga{}).Where("id = ?", fx.raceLoadID).Update("audiencia", models.AudienciaRedPrivada)
	if err != nil {
		t.Fatal(err)
	}
	res := request(t, driver.client, "GET", "/home", nil, "", "")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || strings.Contains(string(body), "RT-RACE-0001") {
		t.Fatalf("private load leaked on home: status=%d", res.StatusCode)
	}
	res = request(t, owner.client, "GET", "/home", nil, "", "")
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(body), "RT-RACE-0001") {
		t.Fatal("owner lost visibility")
	}
	var publisher models.Publicador
	if err := facades.Orm().Query().Where("user_id = ?", owner.user.ID).First(&publisher); err != nil {
		t.Fatal(err)
	}
	var delivered models.Carga
	if err := facades.Orm().Query().Where("numero_referencia = ?", "RT-LOAD-0001").First(&delivered); err != nil {
		t.Fatal(err)
	}
	foreign := models.Direccion{OwnerID: &driver.user.ID, Ciudad: "Private Address", EstadoProvincia: "Province", Pais: "Cuba"}
	if err := facades.Orm().Query().Create(&foreign); err != nil {
		t.Fatal(err)
	}
	form := url.Values{"numero_referencia": {"RT-RACE-0001"}, "origen_direccion_id": {u(foreign.ID)}, "destino_direccion_id": {u(fx.addressID)}, "fecha_recogida": {"2026-10-09T10:00"}, "tipo_carga": {string(models.CargaFTL)}, "tipo_equipo": {string(models.EquipoDryVan)}, "peso_kg": {"100"}, "distancia_km": {"10"}, "tarifa_total": {"100"}, "moneda": {"USD"}, "audiencia": {"load_board"}}
	res = request(t, owner.client, "POST", "/loads/"+u(fx.raceLoadID), form, owner.csrf, "")
	res.Body.Close()
	var check models.Carga
	facades.Orm().Query().Where("id = ?", fx.raceLoadID).First(&check)
	if check.OrigenDireccionID == foreign.ID {
		t.Fatal("foreign address accepted")
	}
	form.Set("origen_direccion_id", u(fx.addressID))
	form.Set("numero_referencia", "RT-LOAD-0001")
	res = request(t, owner.client, "POST", "/loads/"+u(delivered.ID), form, owner.csrf, "")
	res.Body.Close()
	check = models.Carga{}
	if err := facades.Orm().Query().Where("id = ?", delivered.ID).First(&check); err != nil {
		t.Fatal(err)
	}
	if !check.FechaRecogida.Equal(delivered.FechaRecogida) {
		t.Fatal("delivered load was modified")
	}
	// A nonmember cannot notify the owner of a private load.
	calls := 0
	sender := func(d, p *models.User, l *models.Carga, comment string) error {
		calls++
		if p.ID != owner.user.ID || d.ID != driver.user.ID || comment != "Available tomorrow" {
			t.Fatal("wrong recipient/comment")
		}
		return nil
	}
	if err := services.SendLoadInterest(driver.user.ID, fx.raceLoadID, "Available tomorrow", sender); err == nil {
		t.Fatal("private interest allowed")
	}
	_, err = facades.Orm().Query().Model(&models.Carga{}).Where("id = ?", fx.raceLoadID).Update("audiencia", models.AudienciaLoadBoard)
	if err != nil {
		t.Fatal(err)
	}
	if err := services.SendLoadInterest(driver.user.ID, fx.raceLoadID, "   ", sender); err == nil {
		t.Fatal("blank comment accepted")
	}
	if err := services.SendLoadInterest(driver.user.ID, fx.raceLoadID, strings.Repeat("x", 1001), sender); err == nil {
		t.Fatal("oversize comment accepted")
	}
	if err := services.SendLoadInterest(driver.user.ID, fx.raceLoadID, "Available tomorrow", sender); err != nil {
		t.Fatal(err)
	}
	if err := services.SendLoadInterest(driver.user.ID, fx.raceLoadID, "Available tomorrow", sender); err == nil {
		t.Fatal("duplicate interest accepted")
	}
	if calls != 1 {
		t.Fatalf("got %d notifications", calls)
	}
	// Negative and huge sizes must stay bounded even with enough rows to expose the original bug.
	var publicLoad models.Carga
	if err := facades.Orm().Query().Where("id = ?", fx.raceLoadID).First(&publicLoad); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 110; i++ {
		load := publicLoad
		load.ID = 0
		load.NumeroReferencia = "AUDIT-PAGE-" + u(uint(i+1))
		if err := facades.Orm().Query().Create(&load); err != nil {
			t.Fatal(err)
		}
	}
	for _, size := range []int{-1, 0, 1000000} {
		rows, _, err := services.NewCargaService().GetAllWithFilters(map[string]string{}, 1, size)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) > 100 || (size < 1 && len(rows) > 10) {
			t.Fatalf("unbounded size %d: %d rows", size, len(rows))
		}
	}

	var candidates []models.Carga
	if err := facades.Orm().Query().Where("numero_referencia LIKE ?", "AUDIT-PAGE-%").Order("id asc").Limit(10).Find(&candidates); err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 10 {
		t.Fatal("missing quota fixtures")
	}
	quotaSender := func(*models.User, *models.User, *models.Carga, string) error { return nil }
	for i, candidate := range candidates {
		err := services.SendLoadInterest(driver.user.ID, candidate.ID, "Interested", quotaSender)
		if i < 9 && err != nil {
			t.Fatalf("notification %d rejected: %v", i+2, err)
		}
		if i == 9 && err == nil {
			t.Fatal("eleventh daily notification accepted")
		}
	}
	// Different completed loads rated concurrently must preserve the aggregate.
	driverID := *delivered.ChoferID
	second := delivered
	second.ID = 0
	second.NumeroReferencia = "AUDIT-RATING"
	if err := facades.Orm().Query().Create(&second); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i, id := range []uint{delivered.ID, second.ID} {
		wg.Add(1)
		go func(id uint, score int) {
			defer wg.Done()
			<-start
			errs <- services.NewDriverCommunityService().RateCompletedLoad(id, publisher.ID, score, "")
		}(id, 1+i*4)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var rated models.Chofer
	facades.Orm().Query().Where("id = ?", driverID).First(&rated)
	if rated.RatingCount != 2 || rated.RatingAverage != 3 {
		t.Fatalf("wrong aggregate: count=%d avg=%f", rated.RatingCount, rated.RatingAverage)
	}
}

func TestInterestFailureAllowsControlledRetry(t *testing.T) {
	tests.ResetDB(t)
	fx := seedFixture(t, 6)
	var driver principal
	for _, p := range fx.people {
		if p.role == "chofer" {
			driver = p
		}
	}
	fail := func(*models.User, *models.User, *models.Carga, string) error { return errors.New("SMTP unavailable") }
	if err := services.SendLoadInterest(driver.user.ID, fx.raceLoadID, "Interested", fail); err != nil {
		t.Fatal(err)
	}
	if err := services.SendLoadInterest(driver.user.ID, fx.raceLoadID, "Interested", fail); err == nil {
		t.Fatal("duplicate reservation allowed")
	}
	var rows []struct {
		ID       uint
		Attempts int
		Payload  string
	}
	if err := facades.DB().Select(&rows, "SELECT id, attempts, payload FROM notification_outbox WHERE sent_at IS NULL"); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Attempts != 1 || strings.Contains(rows[0].Payload, "Interested") {
		t.Fatal("failed delivery not retained encrypted")
	}
	if err := facades.DB().Statement("UPDATE notification_outbox SET available_at=CURRENT_TIMESTAMP"); err != nil {
		t.Fatal(err)
	}
	sends := 0
	sender := func(_ *models.User, _ *models.User, _ *models.Carga, comment string) error {
		sends++
		if comment != "Interested" {
			t.Fatal("payload changed")
		}
		return nil
	}
	if err := services.ProcessInterestOutbox(context.Background(), sender); err != nil {
		t.Fatal(err)
	}
	if err := services.ProcessInterestOutbox(context.Background(), sender); err != nil {
		t.Fatal(err)
	}
	if sends != 1 {
		t.Fatalf("wrong deliveries: %d", sends)
	}
}

func TestNetworkPaginationPreservesMembershipBeyondFirstPage(t *testing.T) {
	tests.ResetDB(t)
	fx := seedFixture(t, 6)
	var driver principal
	for _, p := range fx.people {
		if p.role == "chofer" {
			driver = p
		}
	}
	service := services.NewDriverCommunityService()
	if err := service.AddToCompanyNetwork(fx.brokerIDs[0], *driver.user.ChoferID); err != nil {
		t.Fatal(err)
	}
	company := &models.Empresa{ID: fx.carrierID}
	for i := 0; i < 110; i++ {
		user := &models.User{Name: "Pagination Driver", Email: "pagination-" + u(uint(i)) + "@test.invalid", Password: driver.user.Password, Role: "chofer", IsActive: true}
		profile := tests.NewChoferProfile()
		profile.NumeroLicencia = "PAG-" + u(uint(i))
		if err := services.NewUserService().CreateWithRole(user, company, profile, nil); err != nil {
			t.Fatal(err)
		}
		if err := service.AddToCompanyNetwork(fx.brokerIDs[0], profile.ID); err != nil {
			t.Fatal(err)
		}
	}
	first, err := service.ListCompanyNetwork(fx.brokerIDs[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.ListCompanyNetwork(fx.brokerIDs[0], 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 100 || len(second) != 11 {
		t.Fatalf("network pages: %d %d", len(first), len(second))
	}
	members, err := service.CompanyMemberIDs(fx.brokerIDs[0], []models.Chofer{{ID: *driver.user.ChoferID}})
	if err != nil || !members[*driver.user.ChoferID] {
		t.Fatal("membership lost beyond first page")
	}
	drivers, err := services.NewEmpresaService().GetChoferes(u(fx.carrierID), 2)
	if err != nil || len(drivers) == 0 || len(drivers) > 100 {
		t.Fatal("company driver page unavailable")
	}
	var owner principal
	for _, p := range fx.people {
		if p.user.ID == fx.ownerID {
			owner = p
		}
	}
	response := request(t, owner.client, "GET", "/red-choferes?member_page=2&page=2", nil, "", "")
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("paginated network status=%d", response.StatusCode)
	}
}
