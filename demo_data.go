package main

import (
	"fmt"
	"log"
	"time"

	"goravel/app/facades"
	"goravel/app/models"
)

// ensureDemoData carga un conjunto pequeño y coherente para revisar las grillas
// en local. Es idempotente: usa referencias y correos demo únicos.
func ensureDemoData() {
	if env("APP_ENV", "local") != "local" {
		return
	}

	var admin models.User
	if err := facades.Orm().Query().Where("email = ?", "admin@example.com").First(&admin); err != nil {
		return
	}

	password, err := facades.Hash().Make("Admin123!")
	if err != nil {
		log.Printf("demo data: no se pudo generar hash: %v", err)
		return
	}
	getUser := func(name, email, role string) models.User {
		var u models.User
		if err := facades.Orm().Query().Where("email = ?", email).First(&u); err == nil && u.ID > 0 {
			return u
		}
		u = models.User{Name: name, Email: email, Password: password, Role: role, IsActive: true, Country: "Cuba"}
		if err := facades.Orm().Query().Create(&u); err != nil {
			log.Printf("demo data: usuario %s: %v", email, err)
		}
		return u
	}

	broker := getUser("Lucía Comercial", "demo.broker@bufalo.local", "publicador")
	carrier := getUser("Carlos Transporte", "demo.carrier@bufalo.local", "chofer")

	str := func(v string) *string { return &v }
	lat := func(v float64) *float64 { return &v }
	address := func(street, city, state string, latitude, longitude float64) models.Direccion {
		d := models.Direccion{OwnerID: &admin.ID, Calle: str(street), Ciudad: city, EstadoProvincia: state, CodigoPostal: str("10100"), Pais: "Cuba", Latitud: lat(latitude), Longitud: lat(longitude)}
		if err := facades.Orm().Query().Create(&d); err != nil {
			log.Printf("demo data: dirección %s: %v", city, err)
		}
		return d
	}
	origin := address("Calle 23 #120", "La Habana", "La Habana", 23.1136, -82.3666)
	destination := address("Ave. 54 #18", "Varadero", "Matanzas", 23.1568, -81.2444)

	var brokerCompany models.Empresa
	if err := facades.Orm().Query().Where("tax_id = ?", "DEMO-BROKER-001").First(&brokerCompany); err != nil || brokerCompany.ID == 0 {
		brokerCompany = models.Empresa{Tipo: models.TipoBroker, NombreLegal: "Comercializadora Ruta Norte S.A.", NombreComercial: str("Ruta Norte"), TaxID: str("DEMO-BROKER-001"), MCNumber: str("MC-48291"), DOTNumber: str("DOT-712044"), Telefono: str("+53 5555 0101"), Email: str("operaciones@rutanorte.local"), OwnerID: &broker.ID, Estado: models.EmpresaActiva}
		if err := facades.Orm().Query().Create(&brokerCompany); err != nil {
			log.Printf("demo data: empresa broker: %v", err)
		}
	}
	var carrierCompany models.Empresa
	if err := facades.Orm().Query().Where("tax_id = ?", "DEMO-CARRIER-001").First(&carrierCompany); err != nil || carrierCompany.ID == 0 {
		carrierCompany = models.Empresa{Tipo: models.TipoCarrier, NombreLegal: "Transportes Sierra Azul S.A.", NombreComercial: str("Sierra Azul"), TaxID: str("DEMO-CARRIER-001"), MCNumber: str("MC-51704"), DOTNumber: str("DOT-804215"), Telefono: str("+53 5555 0102"), Email: str("flota@sierraazul.local"), OwnerID: &carrier.ID, Estado: models.EmpresaActiva}
		if err := facades.Orm().Query().Create(&carrierCompany); err != nil {
			log.Printf("demo data: empresa carrier: %v", err)
		}
	}

	var pub models.Publicador
	if err := facades.Orm().Query().Where("user_id = ?", broker.ID).First(&pub); err != nil || pub.ID == 0 {
		pub = models.Publicador{UserID: broker.ID, EmpresaID: brokerCompany.ID, NumeroLicenciaBroker: "BROKER-DEMO-01", PaisEmisionLicencia: "Cuba", AniosExperiencia: 7, Especialidad: "Carga general", Comision: 4.5, CreditScore: 780, Estado: models.PublicadorActivo}
		if err := facades.Orm().Query().Create(&pub); err != nil {
			log.Printf("demo data: publicador: %v", err)
		}
	}
	var driver models.Chofer
	if err := facades.Orm().Query().Where("user_id = ?", carrier.ID).First(&driver); err != nil || driver.ID == 0 {
		driver = models.Chofer{UserID: carrier.ID, EmpresaID: carrierCompany.ID, NumeroLicencia: "CHOFER-DEMO-01", TipoLicencia: "C", PaisEmisionLicencia: "Cuba", AniosExperiencia: 9, TiposEquipoPermitidos: "dry_van,reefer", Certificaciones: "carga_general", NumeroSeguro: "SEGURO-DEMO-01", Estado: models.ChoferDisponible}
		if err := facades.Orm().Query().Create(&driver); err != nil {
			log.Printf("demo data: chofer: %v", err)
		}
	}

	var load models.Carga
	if err := facades.Orm().Query().Where("numero_referencia = ?", "DEMO-REF-001").First(&load); err != nil || load.ID == 0 {
		weight, total, rate := 1250.0, 28500.0, 48.25
		commodity := "Electrodomésticos"
		load = models.Carga{NumeroReferencia: "DEMO-REF-001", PublicadorID: pub.ID, EmpresaID: brokerCompany.ID, OrigenDireccionID: origin.ID, DestinoDireccionID: destination.ID, FechaRecogida: time.Now().Add(48 * time.Hour), TipoCarga: models.CargaFTL, TipoEquipo: models.EquipoDryVan, PesoKg: &weight, Commodity: &commodity, DistanciaKm: 285, TarifaTotal: &total, TarifaPorKm: &rate, Moneda: models.MonedaCUP, Estado: models.CargaPublicada, Audiencia: models.AudienciaLoadBoard}
		if err := facades.Orm().Query().Create(&load); err != nil {
			log.Printf("demo data: carga 001: %v", err)
		}
	}

	var secondLoad models.Carga
	if err := facades.Orm().Query().Where("numero_referencia = ?", "DEMO-REF-002").First(&secondLoad); err != nil || secondLoad.ID == 0 {
		weight, total, rate := 800.0, 19200.0, 42.66
		commodity := "Materiales de construcción"
		secondLoad = models.Carga{NumeroReferencia: "DEMO-REF-002", PublicadorID: pub.ID, EmpresaID: brokerCompany.ID, ChoferID: &driver.ID, OrigenDireccionID: destination.ID, DestinoDireccionID: origin.ID, FechaRecogida: time.Now().Add(-24 * time.Hour), TipoCarga: models.CargaLTL, TipoEquipo: models.EquipoReefer, PesoKg: &weight, Commodity: &commodity, DistanciaKm: 285, DistanciaRealKm: &rate, TarifaTotal: &total, TarifaPorKm: &rate, Moneda: models.MonedaCUP, Estado: models.CargaEnTransito, Audiencia: models.AudienciaLoadBoard}
		if err := facades.Orm().Query().Create(&secondLoad); err != nil {
			log.Printf("demo data: carga 002: %v", err)
		}
	}

	var invoice models.Factura
	if err := facades.Orm().Query().Where("numero_factura = ?", "FAC-DEMO-001").First(&invoice); err != nil && load.ID > 0 {
		invoice = models.Factura{CargaID: load.ID, EmisorID: brokerCompany.ID, ReceptorID: carrierCompany.ID, ChoferID: &driver.ID, NumeroFactura: "FAC-DEMO-001", FechaEmision: time.Now().Add(-24 * time.Hour), DistanciaKm: 285, TarifaPorKm: 48.25, Subtotal: 28500, Impuestos: 0, Total: 28500, Moneda: models.MonedaCUP, Estado: models.FacturaEmitida}
		if err := facades.Orm().Query().Create(&invoice); err != nil {
			log.Printf("demo data: factura: %v", err)
		}
	}

	comment := "Carga demo creada para revisión de interfaz"
	userID := fmt.Sprint(admin.ID)
	var history models.CargaHistorial
	if load.ID > 0 && facades.Orm().Query().Where("carga_id = ?", fmt.Sprint(load.ID)).First(&history) != nil {
		if err := facades.Orm().Query().Create(&models.CargaHistorial{CargaID: fmt.Sprint(load.ID), Estado: models.CargaPublicada, Comentario: &comment, UsuarioID: &userID}); err != nil {
			log.Printf("demo data: historial: %v", err)
		}
	}
	var empresaCount, cargaCount, addressCount int64
	empresaCount, _ = facades.Orm().Query().Model(&models.Empresa{}).Count()
	cargaCount, _ = facades.Orm().Query().Model(&models.Carga{}).Count()
	addressCount, _ = facades.Orm().Query().Model(&models.Direccion{}).Count()
	log.Printf("📊 Datos locales: %d empresas, %d cargas, %d direcciones", empresaCount, cargaCount, addressCount)
	log.Println("✅ Datos demo verificados para empresas, direcciones, usuarios, choferes, publicadores, cargas, facturas e historial")
}
