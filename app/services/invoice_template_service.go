package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"goravel/app/facades"
	"goravel/app/models"
)

const MaxInvoiceTemplatesPerPublisher = 24

var invoiceBlockID = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)
var invoiceAccentColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var invoiceBlocks = map[string]bool{
	"brand": true, "parties": true, "details": true, "items": true,
	"totals": true, "payment": true, "notes": true, "signature": true,
}

type FacturaPlantillaInput struct {
	ID             uint     `json:"id"`
	Nombre         string   `json:"name"`
	Preset         string   `json:"preset"`
	Formato        string   `json:"format"`
	Color          string   `json:"color"`
	Bloques        []string `json:"blocks"`
	Predeterminada bool     `json:"default"`
}

type FacturaPlantillaService struct{}

func NewFacturaPlantillaService() *FacturaPlantillaService { return &FacturaPlantillaService{} }

func (s *FacturaPlantillaService) List(publicadorID uint) ([]models.FacturaPlantilla, error) {
	if publicadorID == 0 {
		return nil, errors.New("publicador inválido")
	}
	var templates []models.FacturaPlantilla
	err := facades.Orm().Query().Where("publicador_id = ?", publicadorID).
		OrderBy("predeterminada", "desc").OrderBy("updated_at", "desc").Limit(MaxInvoiceTemplatesPerPublisher).Find(&templates)
	return templates, err
}

func (s *FacturaPlantillaService) Save(publicadorID uint, input FacturaPlantillaInput) (*models.FacturaPlantilla, error) {
	if publicadorID == 0 {
		return nil, errors.New("publicador inválido")
	}
	input.Nombre = strings.TrimSpace(input.Nombre)
	if len(input.Nombre) < 2 || len(input.Nombre) > 48 {
		return nil, errors.New("el nombre debe tener entre 2 y 48 caracteres")
	}
	if input.Formato != "a4" && input.Formato != "letter" && input.Formato != "receipt" {
		return nil, errors.New("formato inválido")
	}
	if !invoiceAccentColor.MatchString(input.Color) {
		return nil, errors.New("color inválido")
	}
	if input.Preset != "" && input.Preset != "classic" && input.Preset != "modern" && input.Preset != "receipt" {
		return nil, errors.New("preset inválido")
	}
	if len(input.Bloques) < 3 || len(input.Bloques) > 16 {
		return nil, errors.New("el diseño debe tener de 3 a 16 bloques")
	}
	seen := make(map[string]bool, len(input.Bloques))
	for _, block := range input.Bloques {
		if !invoiceBlockID.MatchString(block) || !invoiceBlocks[block] || seen[block] {
			return nil, fmt.Errorf("bloque no permitido: %q", block)
		}
		seen[block] = true
	}
	for _, required := range []string{"parties", "details", "totals"} {
		if !seen[required] {
			return nil, fmt.Errorf("el diseño debe incluir el bloque %s", required)
		}
	}
	blob, err := json.Marshal(input.Bloques)
	if err != nil {
		return nil, err
	}

	tx, err := facades.Orm().Query().BeginTransaction()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var owner models.Publicador
	if err := tx.Where("id = ?", publicadorID).LockForUpdate().First(&owner); err != nil || owner.ID == 0 {
		return nil, errors.New("perfil publicador no encontrado")
	}
	if input.ID > 0 {
		var current models.FacturaPlantilla
		if err := tx.Where("id = ? AND publicador_id = ?", input.ID, publicadorID).First(&current); err != nil || current.ID == 0 {
			return nil, errors.New("plantilla no encontrada")
		}
	} else {
		count, err := tx.Model(&models.FacturaPlantilla{}).Where("publicador_id = ?", publicadorID).Count()
		if err != nil {
			return nil, err
		}
		if count >= MaxInvoiceTemplatesPerPublisher {
			return nil, errors.New("límite de diseños alcanzado")
		}
	}
	if input.Predeterminada {
		if _, err := tx.Model(&models.FacturaPlantilla{}).Where("publicador_id = ?", publicadorID).Update("predeterminada", false); err != nil {
			return nil, err
		}
	}
	row := models.FacturaPlantilla{
		PublicadorID: publicadorID, Nombre: input.Nombre, Preset: input.Preset,
		Formato: input.Formato, Color: strings.ToUpper(input.Color), BloquesJSON: string(blob),
		Predeterminada: input.Predeterminada,
	}
	if input.ID > 0 {
		row.ID = input.ID
		result, err := tx.Model(&models.FacturaPlantilla{}).Where("id = ? AND publicador_id = ?", input.ID, publicadorID).
			Update(map[string]interface{}{"nombre": row.Nombre, "preset": row.Preset, "formato": row.Formato, "color": row.Color, "bloques_json": row.BloquesJSON, "predeterminada": row.Predeterminada})
		if err != nil {
			return nil, err
		}
		if result.RowsAffected != 1 {
			return nil, errors.New("plantilla modificada concurrentemente")
		}
	} else if err := tx.Create(&row); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *FacturaPlantillaService) GetOwned(publicadorID, templateID uint) (*models.FacturaPlantilla, error) {
	if publicadorID == 0 || templateID == 0 {
		return nil, errors.New("plantilla inválida")
	}
	var row models.FacturaPlantilla
	if err := facades.Orm().Query().Where("id = ? AND publicador_id = ?", templateID, publicadorID).First(&row); err != nil || row.ID == 0 {
		return nil, errors.New("plantilla no encontrada")
	}
	return &row, nil
}
