package db

import (
	"context"
	"strings"

	"github.com/platform-mesh/golang-commons/errors"
	"github.com/platform-mesh/golang-commons/sentry"
	"github.com/platform-mesh/iam-service/pkg/graph"
	"gorm.io/gorm"
)

type Invite struct {
	TenantID   string `gorm:"primaryKey"`
	Email      string `gorm:"primaryKey"`
	EntityType string `gorm:"primaryKey"`
	EntityID   string `gorm:"primaryKey"`
	Roles      string
}

func (d *Database) createInvite(ctx context.Context, invite Invite) error {
	return d.db.WithContext(ctx).Create(invite).Error
}

func (d *Database) GetInvitesForEmail(ctx context.Context, tenantID, email string) ([]Invite, error) {
	return d.getInvites(ctx, Invite{TenantID: tenantID, Email: email})
}

func (d *Database) GetInvitesForEntity(ctx context.Context, tenantID, entityType, entityID string) ([]Invite, error) {
	var invites []Invite
	err := d.db.WithContext(ctx).Where(&Invite{TenantID: tenantID, EntityType: entityType, EntityID: entityID}).Find(&invites).Error

	return invites, err
}

func (d *Database) getInvites(ctx context.Context, criteria Invite) ([]Invite, error) {
	var invites []Invite
	err := d.db.WithContext(ctx).Where(&criteria).Find(&invites).Error

	return invites, err
}

func (d *Database) DeleteInvitesForEmail(ctx context.Context, tenantID, email string) error {
	return d.DeleteInvite(ctx, Invite{TenantID: tenantID, Email: email})
}

func (d *Database) DeleteInvite(ctx context.Context, criteria Invite) error {
	return d.db.WithContext(ctx).Where(&criteria).Delete(&Invite{}).Error
}

func (d *Database) updateInviteRoles(ctx context.Context, criteria Invite, roles string) error {
	var invite Invite
	return d.db.Model(&invite).WithContext(ctx).Where(&criteria).Update("roles", roles).Error
}

func (d *Database) RemoveRoleFromInvite(ctx context.Context, criteria Invite, roleToDelete string) error {
	var invite Invite
	err := d.db.WithContext(ctx).Where(&criteria).First(&invite).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	if err != nil {
		return err
	}

	roles := strings.Split(invite.Roles, ",")
	var newRoles []string
	for _, role := range roles {
		if role != roleToDelete {
			newRoles = append(newRoles, role)
		}
	}

	if len(newRoles) == 0 {
		return d.DeleteInvite(ctx, criteria)
	}

	updatedRoles := strings.Join(newRoles, ",")
	return d.updateInviteRoles(ctx, criteria, updatedRoles)
}

func (d *Database) InviteUser(ctx context.Context, tenantID string, invite graph.Invite, notifyByEmail bool) error {
	newRoles := strings.Join(invite.Roles, ",")
	byEmailAndEntity := Invite{TenantID: tenantID, Email: invite.Email, EntityType: invite.Entity.EntityType, EntityID: invite.Entity.EntityID}

	invites, err := d.getInvites(ctx, byEmailAndEntity)
	if err != nil {
		d.logger.Error().Err(err).Str("email", invite.Email).Msg("Failed to get invites")
		sentry.CaptureError(err, sentry.Tags{"email": invite.Email})
		return err
	}

	userWasAlreadyInvited := len(invites) > 0
	if userWasAlreadyInvited {
		existingInvite := invites[0]

		if existingInvite.Roles != newRoles {
			err = d.updateInviteRoles(ctx, byEmailAndEntity, newRoles)
			if err != nil {
				d.logger.Error().Err(err).Str("email", invite.Email).Msg("Could not update invite roles")
				sentry.CaptureError(err, sentry.Tags{"email": invite.Email})
				return err
			}
		}

		d.logger.Log().Str("email", invite.Email).Msg("User has already been invited")
		return nil
	}

	inv := Invite{
		TenantID:   tenantID,
		Email:      invite.Email,
		EntityType: invite.Entity.EntityType,
		EntityID:   invite.Entity.EntityID,
		Roles:      strings.Join(invite.Roles, ","),
	}
	err = d.createInvite(ctx, inv)
	if err != nil {
		d.logger.Error().Err(err).Str("email", invite.Email).Msg("Could not create invite")
		sentry.CaptureError(err, sentry.Tags{"email": invite.Email})
		return err
	}

	if d.userHooks != nil {
		d.userHooks.UserInvited(ctx, &graph.User{Email: invite.Email}, tenantID, invite.Entity.EntityID, notifyByEmail)
	}

	return err
}
