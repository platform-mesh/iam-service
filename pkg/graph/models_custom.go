package graph

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Base struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type User struct {
	Base
	ID                    string  `gorm:"type:uuid;primary_key"`
	TenantID              string  `gorm:"index:idx_user_id,unique;index:idx_email,unique;index:idx_user_id_email,unique"`
	UserID                string  `gorm:"index:idx_user_id,unique,where:user_id != '';index:idx_user_id_email,unique;check:user_id_or_email,user_id <> '' IS TRUE OR email <> '' IS TRUE"` // nolint: lll
	Email                 string  `gorm:"index:idx_email,unique,where:email != '';index:idx_user_id_email,unique"`
	FirstName             *string `gorm:"index:idx_first_name"`
	LastName              *string `gorm:"index:idx_last_name"`
	InvitationOutstanding bool
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}

	return nil
}

type Team struct {
	Base
	ID           string `gorm:"type:uuid;primary_key"`
	TenantID     string `gorm:"index:idx_team_name,unique"`
	Name         string `gorm:"index:idx_team_name,unique"`
	ParentTeam   *Team
	ParentTeamID *string
	DbChildTeams []*Team `gorm:"foreignKey:ParentTeamID"`
}

func (t *Team) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

type TeamConnection struct {
	Teams    []*Team   `json:"teams"`
	PageInfo *PageInfo `json:"pageInfo"`
}

type UserConnection struct {
	User     []*User   `json:"user"`
	PageInfo *PageInfo `json:"pageInfo,omitempty"`
}

type PageInfo struct {
	TotalCount int `json:"totalCount"`
}

type Invite struct {
	Email  string       `json:"email"`
	Entity *EntityInput `json:"entity"`
	Roles  []string     `json:"roles"`
}

// An entity of Hyperspace Portal
type EntityInput struct {
	// the type of entity e.g. team, project etc.
	EntityType string `json:"entityType"`
	// the identifier for the entity itself e.g. name or id
	EntityID string `json:"entityId"`
}

type Change struct {
	UserID string   `json:"userId"`
	Roles  []string `json:"roles"`
}

type GrantedUser struct {
	User  *User   `json:"user"`
	Roles []*Role `json:"roles,omitempty"`
}

type GrantedUserConnection struct {
	Users    []*GrantedUser `json:"users,omitempty"`
	PageInfo *PageInfo      `json:"pageInfo"`
}

type Role struct {
	DisplayName   string        `json:"displayName"`
	TechnicalName string        `json:"technicalName"`
	Permissions   []*Permission `json:"permissions,omitempty"`
}

type TeamInput struct {
	Name        string `json:"name"`
	AdminUserID string `json:"adminUserID"`
}

type TenantInfo struct {
	TenantID     string   `json:"tenantId"`
	Subdomain    string   `json:"subdomain"`
	EmailDomain  string   `json:"emailDomain"`
	EmailDomains []string `json:"emailDomains,omitempty"`
}

type UserInput struct {
	UserID                string  `json:"userId"`
	Email                 string  `json:"email"`
	FirstName             *string `json:"firstName,omitempty"`
	LastName              *string `json:"lastName,omitempty"`
	InvitationOutstanding *bool   `json:"invitationOutstanding,omitempty"`
}

type Zone struct {
	ZoneID   string `json:"zoneId"`
	TenantID string `json:"tenantId"`
}

type Permission struct {
	DisplayName string `json:"displayName"`
	Relation    string `json:"relation"`
}
