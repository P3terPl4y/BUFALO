package services

import (
	"github.com/goravel/framework/contracts/database/orm"
	"goravel/app/facades"
	"goravel/app/models"
	"time"
)

// RecordLoadActivity writes an audit event using the caller's transaction so
// the timeline can never claim a change that was rolled back.
func recordLoadActivity(tx orm.Query, loadID uint, state models.EstadoCarga, comment string, actorID uint) error {
	var actorPtr *uint
	if actorID > 0 {
		actorPtr = &actorID
	}
	return tx.Create(&models.CargaHistorial{
		CargaID: loadID,
		Estado:  state, Comentario: &comment, UsuarioID: actorPtr, CreatedAt: time.Now().UTC(),
	})
}

// LoadActivity returns a bounded, newest-first activity history for a load.
func LoadActivity(loadID uint) ([]models.CargaHistorial, error) {
	var rows []models.CargaHistorial
	err := facades.Orm().Query().Where("carga_id = ?", loadID).Order("created_at desc").Limit(50).Find(&rows)
	return rows, err
}
