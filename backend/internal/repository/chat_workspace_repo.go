package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type chatWorkspaceRepository struct {
	db *sql.DB
}

func NewChatWorkspaceRepository(db *sql.DB) service.ChatWorkspaceRepository {
	return &chatWorkspaceRepository{db: db}
}

func (r *chatWorkspaceRepository) ListConversations(ctx context.Context, userID int64) ([]service.WorkspaceConversation, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, user_id, title, status, last_message_at, created_at, updated_at
FROM workspace_conversations
WHERE user_id = $1
ORDER BY COALESCE(last_message_at, updated_at) DESC, id DESC
LIMIT 100`, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspace conversations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.WorkspaceConversation, 0)
	for rows.Next() {
		conversation, err := scanWorkspaceConversation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *conversation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *chatWorkspaceRepository) CreateConversation(ctx context.Context, userID int64, title string) (*service.WorkspaceConversation, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO workspace_conversations (user_id, title, status, created_at, updated_at)
VALUES ($1, $2, 'active', NOW(), NOW())
RETURNING id, user_id, title, status, last_message_at, created_at, updated_at`, userID, title)
	return scanWorkspaceConversation(row)
}

func (r *chatWorkspaceRepository) GetConversation(ctx context.Context, userID, conversationID int64) (*service.WorkspaceConversation, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, user_id, title, status, last_message_at, created_at, updated_at
FROM workspace_conversations
WHERE id = $1 AND user_id = $2
LIMIT 1`, conversationID, userID)
	conversation, err := scanWorkspaceConversation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrWorkspaceConversationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get workspace conversation: %w", err)
	}
	return conversation, nil
}

func (r *chatWorkspaceRepository) ListMessages(ctx context.Context, userID, conversationID int64) ([]service.WorkspaceMessage, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, conversation_id, user_id, message_type, role, content, model, intent, status, metadata, created_at, updated_at
FROM workspace_messages
WHERE conversation_id = $1 AND user_id = $2
ORDER BY created_at ASC, id ASC`, conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspace messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.WorkspaceMessage, 0)
	for rows.Next() {
		msg, err := scanWorkspaceMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *chatWorkspaceRepository) AppendMessage(ctx context.Context, userID int64, input service.WorkspaceAppendMessageInput, titleIfEmpty string) (*service.WorkspaceMessage, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin workspace append: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var exists int
	if err := tx.QueryRowContext(ctx, `
SELECT 1
FROM workspace_conversations
WHERE id = $1 AND user_id = $2
FOR UPDATE`, input.ConversationID, userID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrWorkspaceConversationNotFound
		}
		return nil, fmt.Errorf("lock workspace conversation: %w", err)
	}

	metadataJSON, err := marshalWorkspaceMessageMetadata(input.Metadata)
	if err != nil {
		return nil, err
	}

	row := tx.QueryRowContext(ctx, `
INSERT INTO workspace_messages (
	conversation_id, user_id, message_type, role, content, model, intent, status, metadata, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
RETURNING id, conversation_id, user_id, message_type, role, content, model, intent, status, metadata, created_at, updated_at`,
		input.ConversationID,
		userID,
		input.MessageType,
		input.Role,
		input.Content,
		input.Model,
		input.Intent,
		input.Status,
		metadataJSON,
	)
	msg, err := scanWorkspaceMessage(row)
	if err != nil {
		return nil, fmt.Errorf("insert workspace message: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE workspace_conversations
SET last_message_at = $2,
    updated_at = NOW(),
    title = CASE WHEN title = '' AND $3 <> '' THEN $3 ELSE title END
WHERE id = $1 AND user_id = $4`, input.ConversationID, msg.CreatedAt, titleIfEmpty, userID); err != nil {
		return nil, fmt.Errorf("update workspace conversation activity: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit workspace append: %w", err)
	}
	return msg, nil
}

type workspaceConversationScanner interface {
	Scan(dest ...any) error
}

func scanWorkspaceConversation(row workspaceConversationScanner) (*service.WorkspaceConversation, error) {
	var out service.WorkspaceConversation
	var lastMessageAt sql.NullTime
	if err := row.Scan(
		&out.ID,
		&out.UserID,
		&out.Title,
		&out.Status,
		&lastMessageAt,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if lastMessageAt.Valid {
		t := lastMessageAt.Time
		out.LastMessageAt = &t
	}
	return &out, nil
}

type workspaceMessageScanner interface {
	Scan(dest ...any) error
}

func scanWorkspaceMessage(row workspaceMessageScanner) (*service.WorkspaceMessage, error) {
	var out service.WorkspaceMessage
	var metadataBytes []byte
	if err := row.Scan(
		&out.ID,
		&out.ConversationID,
		&out.UserID,
		&out.MessageType,
		&out.Role,
		&out.Content,
		&out.Model,
		&out.Intent,
		&out.Status,
		&metadataBytes,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(metadataBytes) > 0 {
		var metadata map[string]any
		if err := json.Unmarshal(metadataBytes, &metadata); err == nil {
			out.Metadata = metadata
		}
	}
	return &out, nil
}

func marshalWorkspaceMessageMetadata(metadata map[string]any) (any, error) {
	if len(metadata) == 0 {
		return nil, nil
	}
	body, err := json.Marshal(metadata)
	if err != nil {
		return nil, service.ErrWorkspaceInvalidMessage
	}
	return body, nil
}

var _ service.ChatWorkspaceRepository = (*chatWorkspaceRepository)(nil)
