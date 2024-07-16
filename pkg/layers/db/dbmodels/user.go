package dbmodels

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type base struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type User struct {
	base
	ID        string `gorm:"type:uuid;primary_key"`
	TenantID  string `gorm:"index:idx_user_id,unique;index:idx_email,unique;index:idx_user_id_email,unique"`
	UserID    string `gorm:"index:idx_user_id,unique,where:user_id != '';index:idx_user_id_email,unique;check:user_id_or_email,user_id <> '' IS TRUE OR email <> '' IS TRUE"`
	Email     string `gorm:"index:idx_email,unique,where:email != '';index:idx_user_id_email,unique"`
	FirstName string
	LastName  string
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}

	u.CreatedAt = time.Now()

	return nil
}
