package service

import (
	"context"
	"database/sql"
	"errors"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *OpsService) ListUserErrorRequests(ctx context.Context, userID int64, filter *OpsErrorLogFilter) (*UserErrorRequestList, error) {
	if filter == nil {
		filter = &OpsErrorLogFilter{}
	}
	f := *filter
	filter = &f
	uid := userID
	filter.UserID = &uid

	filter.MatchDeletedKeyOwner = true

	filter.View = "all"
	filter.ExcludeCountTokens = true
	filter.ModelFuzzy = true

	filter.UserQuery = ""
	filter.Owner = ""
	filter.Source = ""

	filter.Phase = ""

	list, err := s.opsRepo.ListErrorLogs(ctx, filter)
	if err != nil {
		return nil, err
	}
	items := make([]*UserErrorRequest, 0, len(list.Errors))
	for _, e := range list.Errors {
		if r := ToUserErrorRequest(e); r != nil {
			items = append(items, r)
		}
	}
	return &UserErrorRequestList{
		Items:    items,
		Total:    list.Total,
		Page:     list.Page,
		PageSize: list.PageSize,
	}, nil
}

func (s *OpsService) GetUserErrorRequestDetail(ctx context.Context, userID, id int64) (*UserErrorRequestDetail, error) {
	if s.opsRepo == nil {
		return nil, infraerrors.NotFound("OPS_ERROR_NOT_FOUND", "ops error log not found")
	}
	if id <= 0 {
		return nil, infraerrors.BadRequest("OPS_ERROR_INVALID_ID", "invalid error id")
	}
	detail, err := s.opsRepo.GetErrorLogByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.NotFound("OPS_ERROR_NOT_FOUND", "ops error log not found")
		}
		return nil, infraerrors.InternalServer("OPS_ERROR_LOAD_FAILED", "Failed to load ops error log").WithCause(err)
	}

	ownedDirectly := detail.UserID != nil && *detail.UserID == userID
	ownedViaDeletedKey := detail.DeletedKeyOwnerUserID != nil && *detail.DeletedKeyOwnerUserID == userID
	if !ownedDirectly && !ownedViaDeletedKey {
		return nil, infraerrors.NotFound("OPS_ERROR_NOT_FOUND", "ops error log not found")
	}
	return ToUserErrorRequestDetail(detail), nil
}
