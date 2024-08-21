package db

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	gorm.Model
	ID            string `gorm:"type:uuid;primaryKey"`
	DisplayName   string `gorm:"not null"`
	TechnicalName string `gorm:"not null;uniqueIndex:idx_technical_name_entity_type_entity_id"`
	EntityType    string `gorm:"not null;uniqueIndex:idx_technical_name_entity_type_entity_id"`
	EntityID      string `gorm:"uniqueIndex:idx_technical_name_entity_type_entity_id;default:''"`
}

func (r *Role) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

func (d *Database) GetRolesForEntity(ctx context.Context, entityType string, entityID string) ([]Role, error) {

	var roles []Role
	builder := d.db.Where(&Role{EntityType: entityType})

	if entityID != "" {
		builder.Or(&Role{EntityType: entityType, EntityID: entityID})
	}

	res := builder.Find(&roles)

	return roles, res.Error
}

func (d *Database) GetRolesByTechnicalNames(ctx context.Context, entityType string, technicalNames []string) ([]*Role, error) {

	var roles []*Role
	res := d.db.Where("technical_name IN ? AND entity_type = ?", technicalNames, entityType).Find(&roles)

	return roles, res.Error
}
