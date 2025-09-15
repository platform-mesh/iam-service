/*
 * Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved.
 */

package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/platform-mesh/golang-commons/jwt"
	"gorm.io/gorm"
	"sigs.k8s.io/yaml"

	"github.com/platform-mesh/golang-commons/logger"

	"github.com/platform-mesh/iam-service/pkg/graph"
)

const (
	connectToDbError  = "Error connecting to database"
	readDataFileError = "error occurred when reading data file"
)

type UserHooks interface {
	UserInvited(ctx context.Context, user *graph.User, tenantID string, scope string, userInvited bool)
	UserCreated(ctx context.Context, user *graph.User, tenantID string)
	UserRemoved(ctx context.Context, user *graph.User, tenantID string)
	UserLogin(ctx context.Context, user *graph.User, tenantID string)
}

type Service interface {
	UserService

	Save(user *graph.User) error
	Close() error

	// tenant
	GetTenantConfigurationForContext(ctx context.Context) (*TenantConfiguration, error)

	// roles
	GetRolesForEntity(ctx context.Context, entityType string, entityID string) ([]Role, error)
	GetRolesByTechnicalNames(ctx context.Context, entityType string, technicalNames []string) ([]*Role, error)
}

type UserService interface {
	GetUserByID(ctx context.Context, tenantID string, userId string) (*graph.User, error)
	GetUsersByUserIDs(
		ctx context.Context, tenantID string, userIDs []string, limit, page int, searchTerm *string,
		sortBy *graph.SortByInput,
	) ([]*graph.User, error)
	GetUserByEmail(ctx context.Context, tenantID string, email string) (*graph.User, error)
	GetOrCreateUser(ctx context.Context, tenantID string, input graph.UserInput) (*graph.User, error)
	RemoveUser(ctx context.Context, tenantID string, userId string, email string) (bool, error)
	GetUsers(ctx context.Context, tenantID string, limit int, page int) (*graph.UserConnection, error)
	GetInvitesForEntity(ctx context.Context, tenantID, entityType, entityID string) ([]Invite, error)
	RemoveRoleFromInvite(ctx context.Context, criteria Invite, roleToDelete string) error
	DeleteInvite(ctx context.Context, criteria Invite) error
	InviteUser(ctx context.Context, tenantID string, invite graph.Invite, notifyByEmail bool) error
	SearchUsers(ctx context.Context, tenantID, query string) ([]*graph.User, error)
	LoginUserWithToken(ctx context.Context, tokenInfo jwt.WebToken, userID string, tenantID string, user *graph.User, email string) error
	// hooks
	SetUserHooks(hooks UserHooks)
	GetUserHooks() UserHooks
}

type DataLoader interface {
	LoadTenantConfigData(filePath string) error
	LoadRoleData(filePath string) error
	Close() error
	GetGormDB() *gorm.DB
}

// Database defines a connection to a DB
type Database struct {
	cfg       ConfigDatabase
	db        *gorm.DB
	logger    *logger.Logger
	userHooks UserHooks
}

var _ Service = (*Database)(nil)

// New returns an initialized Database struct
func New(cfg ConfigDatabase, dbConn *gorm.DB, logger *logger.Logger, migrate bool, isLocal bool) (*Database, error) {
	logger.Info().Msg("Initializing Database")
	db, err := dbConn.DB()
	if err != nil {
		return nil, errors.Wrap(err, connectToDbError)
	}

	// configure Gorm connection pool to avoid reaching PostgreSQL limits
	if cfg.MaxOpenConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}

	if connLifetime, err := time.ParseDuration(cfg.MaxConnLifetime); err == nil {
		db.SetConnMaxLifetime(connLifetime)
	}

	database := &Database{
		db:     dbConn,
		cfg:    cfg,
		logger: logger.ComponentLogger("database"),
	}

	if migrate {
		logger.Info().Msg("Migrating database to latest version")
		dbConn.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
		// migrate DB scheme to models state
		models := []interface{}{
			&graph.User{},
			&graph.Team{},
			&TenantConfiguration{},
			&Invite{},
			&Role{},
		}
		for _, model := range models {
			logger.Info().Msg("Auto migration of model")
			err = dbConn.AutoMigrate(model)
			if err != nil {
				logger.Error().Err(err).Send()
				return nil, errors.Wrap(err, "Failed to migrate model")
			}

			if _, ok := model.(*Invite); ok {
				dbConn.Migrator().DropConstraint(&Invite{}, "invites_pkey")                                                            //nolint:all
				dbConn.Exec("ALTER TABLE invites ADD CONSTRAINT invites_pkey PRIMARY KEY (email, entity_type, entity_id, tenant_id);") //nolint:all
			}
		}
	}

	// initialize DB with bootstrap data for local mode
	if isLocal { // nolint: nestif
		users, err := database.LoadUserData(cfg.LocalData.DataPathUser)
		if err != nil {
			logger.Error().Err(err).Str("filePath", cfg.LocalData.DataPathUser).Msg("failed to load user data")
		}
		err = database.LoadInvitationData(cfg.LocalData.DataPathInvitations)
		if err != nil {
			logger.Error().Err(err).Str("filePath", cfg.LocalData.DataPathInvitations).Msg("failed to load invitation data")
		}
		err = database.LoadTeamData(cfg.LocalData.DataPathTeam, users)
		if err != nil {
			logger.Error().Err(err).Str("filePath", cfg.LocalData.DataPathTeam).Msg("failed to load team data")
		}
		err = database.LoadTenantConfigData(cfg.LocalData.DataPathTenantConfiguration)
		if err != nil {
			logger.Error().Err(err).Str("filePath", cfg.LocalData.DataPathTenantConfiguration).Msg("failed to load tenant config data")
		}
		err = database.LoadRoleData(cfg.LocalData.DataPathRoles)
		if err != nil {
			logger.Error().Err(err).Str("filePath", cfg.LocalData.DataPathRoles).Msg("failed to load role data")
		}
	}

	return database, nil
}

func (d *Database) SetUserHooks(hooks UserHooks) {
	d.userHooks = hooks
}

func (d *Database) GetUserHooks() UserHooks { // nolint: ireturn
	return d.userHooks
}

func (d *Database) SetConfig(cfg ConfigDatabase) {
	d.cfg = cfg
}

func (d *Database) GetGormDB() *gorm.DB {
	return d.db
}

// Close closes the database connection
func (d *Database) Close() error {
	db, err := d.db.DB()
	if err != nil {
		return err
	}

	return db.Close()
}

// TODO: TenantContfig update / Delete not yet implemented.
func (d *Database) LoadTenantConfigData(filePath string) error {
	dat, err := os.ReadFile(filePath)
	if err != nil {
		return errors.Wrapf(err, "%s %v", readDataFileError, filePath)
	}
	tenantConfigurations := TenantConfigurationsList{}
	err = yaml.Unmarshal(dat, &tenantConfigurations)
	if err != nil {
		return err
	}
	for _, tc := range tenantConfigurations.Configs {
		existingTenantConfig := &TenantConfiguration{}
		result := d.db.
			Where("tenant_id = ?", tc.TenantID).
			Where("issuer = ?", tc.Issuer).
			Where("audience = ?", tc.Audience).
			First(existingTenantConfig)

		if result.RowsAffected > 0 {
			continue
		}

		newTenantConfiguration := &TenantConfiguration{
			TenantID: tc.TenantID,
			Issuer:   tc.Issuer,
			Audience: tc.Audience,
		}
		result = d.db.Create(&newTenantConfiguration)
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

func (d *Database) LoadTeamData(filePath string, users []*graph.User) error {
	dat, err := os.ReadFile(filePath)
	if err != nil {
		return errors.Wrapf(err, "%s %v", readDataFileError, filePath)
	}
	teamList := TeamList{}
	err = yaml.Unmarshal(dat, &teamList)
	if err != nil {
		return err
	}
	processedTeams := []graph.Team{}
	for _, team := range teamList.Team {
		existingTeam := &graph.Team{}
		result := d.db.Where("name = ?", team.Name).First(existingTeam)

		if result.RowsAffected > 0 {
			processedTeams = append(processedTeams, *existingTeam)
			continue
		}

		newTeam := &graph.Team{
			Name:     team.Name,
			TenantID: team.TenantID,
		}

		if team.ParentTeam != nil {
			for _, processedTeam := range processedTeams {
				if processedTeam.Name == *team.ParentTeam {
					newTeam.ParentTeam = &processedTeam
				}
			}
		}

		result = d.db.Create(&newTeam)
		if result.Error != nil {
			return result.Error
		}
		processedTeams = append(processedTeams, *newTeam)
	}
	if len(processedTeams) == 0 {
		return fmt.Errorf("no teams where loaded into the DB, validate that file at path '%s' has teams information", filePath)
	}
	return nil
}

func (d *Database) LoadUserData(filePath string) ([]*graph.User, error) {
	dat, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "%s %v", readDataFileError, filePath)
	}
	userList := UserList{}
	err = yaml.Unmarshal(dat, &userList)
	if err != nil {
		return nil, err
	}

	newUserList := []*graph.User{}
	for _, user := range userList.User {
		existingUser := &graph.User{}
		result := d.db.Where("user_id = ? AND tenant_id = ?", user.UserID, user.TenantID).First(existingUser)
		if result.RowsAffected == 0 {
			newUser := &graph.User{
				TenantID:              user.TenantID,
				UserID:                user.UserID,
				Email:                 user.Email,
				FirstName:             user.FirstName,
				LastName:              user.LastName,
				InvitationOutstanding: user.InvitationOutstanding,
			}

			result = d.db.Create(&newUser)
			if result.Error != nil {
				return nil, result.Error
			}
			newUserList = append(newUserList, newUser)
		} else {
			newUserList = append(newUserList, existingUser)
		}
	}
	if len(newUserList) == 0 {
		return nil, fmt.Errorf("no users where loaded into the DB, validate that file at path '%s' has users information", filePath)
	}
	return newUserList, nil
}

func (d *Database) LoadInvitationData(filePath string) error {
	invitationData, err := os.ReadFile(filePath)
	if err != nil {
		return errors.Wrapf(err, "%s %v", readDataFileError, filePath)
	}
	inviteList := InviteList{}
	err = yaml.Unmarshal(invitationData, &inviteList)
	if err != nil {
		return err
	}
	processedInvitations := []Invite{}
	for _, invite := range inviteList.Invitations {
		existingInvite := &Invite{}
		result := d.db.Where("email = ?", invite.Email).First(existingInvite)

		if result.RowsAffected > 0 {
			processedInvitations = append(processedInvitations, *existingInvite)
			continue
		}

		newInvite := &Invite{
			TenantID:   invite.TenantID,
			Email:      invite.Email,
			EntityType: invite.EntityType,
			EntityID:   invite.EntityID,
			Roles:      invite.Roles,
		}

		result = d.db.Create(&newInvite)
		if result.Error != nil {
			return result.Error
		}
		processedInvitations = append(processedInvitations, *newInvite)
	}
	if len(processedInvitations) == 0 {
		return fmt.Errorf("no invitations where loaded into the DB, validate that file at path '%s' has invitations information", filePath)
	}
	return nil
}

func (d *Database) LoadRoleData(filePath string) error {
	if filePath == "" {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var roles []Role
	err = yaml.Unmarshal(data, &roles)
	if err != nil {
		return err
	}

	for _, role := range roles {

		var existingRole Role
		res := d.db.Where(&Role{TechnicalName: role.TechnicalName, EntityType: role.EntityType}).First(&existingRole)
		if res.Error != nil && !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return res.Error
		}
		if res.RowsAffected != 0 {
			existingRole.DisplayName = role.DisplayName
			existingRole.EntityType = role.EntityType
			existingRole.EntityID = role.EntityID
			res = d.db.Save(&existingRole)
			if res.Error != nil {
				return res.Error
			}
		} else {
			res = d.db.Create(&role)
			if res.Error != nil {
				return res.Error
			}
		}

	}

	return nil
}

type User struct {
	ID                    string  `json:"id"`
	TenantID              string  `json:"tenant_id"`
	UserID                string  `json:"user_id"`
	Email                 string  `json:"email"`
	FirstName             *string `json:"first_name"`
	LastName              *string `json:"last_name"`
	InvitationOutstanding bool
}
type UserList struct {
	User []User
}

type InviteList struct {
	Invitations []Invite
}

type TeamList struct {
	Team []struct {
		ID          string   `json:"id"`
		TenantID    string   `json:"tenantId"`
		Name        string   `json:"name"`
		DisplayName string   `json:"displayName"`
		Members     []string `json:"members"`
		ParentTeam  *string  `json:"parentTeam"`
	}
}

type TenantConfigurationsList struct {
	Configs []TenantConfiguration `json:"configs"`
}
