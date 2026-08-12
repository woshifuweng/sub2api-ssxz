package service

import (
	"context"
	"fmt"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	AdminAPIKeyAuditActionEnable      = "api_key.enable"
	AdminAPIKeyAuditActionDisable     = "api_key.disable"
	AdminAPIKeyAuditActionDelete      = "api_key.delete"
	AdminAPIKeyAuditActionChangeGroup = "api_key.change_group"
)

var (
	ErrAdminAPIKeyInvalidAction = infraerrors.BadRequest("INVALID_ADMIN_API_KEY_ACTION", "invalid admin API key action")
	ErrAdminActorRequired       = infraerrors.Forbidden("ADMIN_ACTOR_REQUIRED", "authenticated admin identity is required")
	ErrAdminAPIKeyGroupInactive = infraerrors.BadRequest("GROUP_NOT_ACTIVE", "target group is not active")
	ErrAdminAPIKeySubscription  = infraerrors.BadRequest("SUBSCRIPTION_REQUIRED", "user does not have an active subscription for this group")
)

// AdminAPIKeyMutationService is separate from the broad AdminService contract so
// privileged write operations remain explicit and independently testable.
type AdminAPIKeyMutationService interface {
	AdminSetAPIKeyEnabled(ctx context.Context, keyID int64, enabled bool, actorUserID int64) (*APIKey, error)
	AdminChangeAPIKeyGroup(ctx context.Context, keyID, groupID, actorUserID int64) (*AdminUpdateAPIKeyGroupIDResult, error)
	AdminDeleteAPIKey(ctx context.Context, keyID, actorUserID int64) error
}

type AdminAPIKeyMutationCommand struct {
	APIKeyID    int64
	ActorUserID int64
	Action      string
	GroupID     *int64
}

// AdminAPIKeyMutationRecord contains the internal post-commit state. The raw
// authentication key is used only for cache invalidation and must never be
// serialized or written to the audit record.
type AdminAPIKeyMutationRecord struct {
	APIKeyID               int64
	UserID                 int64
	AuthenticationKey      string
	Status                 string
	GroupID                *int64
	GroupName              string
	Deleted                bool
	AutoGrantedGroupAccess bool
}

type AdminAPIKeyMutationRepository interface {
	AdminMutateAPIKey(ctx context.Context, command AdminAPIKeyMutationCommand) (*AdminAPIKeyMutationRecord, error)
}

func (s *adminServiceImpl) adminAPIKeyMutationRepository() (AdminAPIKeyMutationRepository, error) {
	repo, ok := s.apiKeyRepo.(AdminAPIKeyMutationRepository)
	if !ok {
		return nil, fmt.Errorf("admin API key mutation repository is unavailable")
	}
	return repo, nil
}

func validateAdminAPIKeyMutationIdentity(keyID, actorUserID int64) error {
	if actorUserID <= 0 {
		return ErrAdminActorRequired
	}
	if keyID <= 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}

func (s *adminServiceImpl) AdminSetAPIKeyEnabled(ctx context.Context, keyID int64, enabled bool, actorUserID int64) (*APIKey, error) {
	if err := validateAdminAPIKeyMutationIdentity(keyID, actorUserID); err != nil {
		return nil, err
	}
	repo, err := s.adminAPIKeyMutationRepository()
	if err != nil {
		return nil, err
	}
	action := AdminAPIKeyAuditActionDisable
	if enabled {
		action = AdminAPIKeyAuditActionEnable
	}
	record, err := repo.AdminMutateAPIKey(ctx, AdminAPIKeyMutationCommand{
		APIKeyID: keyID, ActorUserID: actorUserID, Action: action,
	})
	if err != nil {
		return nil, err
	}
	s.invalidateAdminAPIKeyAuthentication(ctx, record.AuthenticationKey)
	return s.apiKeyRepo.GetByID(ctx, keyID)
}

func (s *adminServiceImpl) AdminChangeAPIKeyGroup(ctx context.Context, keyID, groupID, actorUserID int64) (*AdminUpdateAPIKeyGroupIDResult, error) {
	if err := validateAdminAPIKeyMutationIdentity(keyID, actorUserID); err != nil {
		return nil, err
	}
	if groupID < 0 {
		return nil, infraerrors.BadRequest("INVALID_GROUP_ID", "group_id must be non-negative")
	}
	repo, err := s.adminAPIKeyMutationRepository()
	if err != nil {
		return nil, err
	}
	record, err := repo.AdminMutateAPIKey(ctx, AdminAPIKeyMutationCommand{
		APIKeyID: keyID, ActorUserID: actorUserID,
		Action: AdminAPIKeyAuditActionChangeGroup, GroupID: &groupID,
	})
	if err != nil {
		return nil, err
	}
	s.invalidateAdminAPIKeyAuthentication(ctx, record.AuthenticationKey)
	apiKey, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	result := &AdminUpdateAPIKeyGroupIDResult{
		APIKey:                 apiKey,
		AutoGrantedGroupAccess: record.AutoGrantedGroupAccess,
	}
	if record.AutoGrantedGroupAccess && record.GroupID != nil {
		gid := *record.GroupID
		result.GrantedGroupID = &gid
		result.GrantedGroupName = record.GroupName
	}
	return result, nil
}

func (s *adminServiceImpl) AdminDeleteAPIKey(ctx context.Context, keyID, actorUserID int64) error {
	if err := validateAdminAPIKeyMutationIdentity(keyID, actorUserID); err != nil {
		return err
	}
	repo, err := s.adminAPIKeyMutationRepository()
	if err != nil {
		return err
	}
	record, err := repo.AdminMutateAPIKey(ctx, AdminAPIKeyMutationCommand{
		APIKeyID: keyID, ActorUserID: actorUserID, Action: AdminAPIKeyAuditActionDelete,
	})
	if err != nil {
		return err
	}
	s.invalidateAdminAPIKeyAuthentication(ctx, record.AuthenticationKey)
	return nil
}

func (s *adminServiceImpl) invalidateAdminAPIKeyAuthentication(ctx context.Context, key string) {
	if s.authCacheInvalidator != nil && key != "" {
		s.authCacheInvalidator.InvalidateAuthCacheByKey(ctx, key)
	}
}
