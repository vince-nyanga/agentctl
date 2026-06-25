package core

import (
	"context"
	"fmt"
	"time"
)

func (s *Store) CreateApproval(approval Approval) (Approval, error) {
	if err := s.Init(); err != nil {
		return Approval{}, err
	}
	db, err := s.open()
	if err != nil {
		return Approval{}, err
	}
	defer db.Close()
	if approval.CreatedAt.IsZero() {
		approval.CreatedAt = time.Now()
	}
	if approval.State == "" {
		approval.State = "pending"
	}
	result, err := db.ExecContext(context.Background(), `INSERT INTO approvals(task_id, agent_name, type, title, description, risk, recommended_action, state, resolution, created_at, resolved_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, approval.TaskID, approval.AgentName, approval.Type, approval.Title, approval.Description, approval.Risk, approval.RecommendedAction, approval.State, approval.Resolution, formatTime(approval.CreatedAt), formatOptionalTime(approval.ResolvedAt))
	if err != nil {
		return Approval{}, err
	}
	approval.ID, err = result.LastInsertId()
	if err != nil {
		return Approval{}, err
	}
	return approval, nil
}

func (s *Store) ListApprovals(taskID, state string) ([]Approval, error) {
	if err := s.Init(); err != nil {
		return nil, err
	}
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, task_id, agent_name, type, title, description, risk, recommended_action, state, resolution, created_at, resolved_at FROM approvals WHERE 1=1`
	args := []any{}
	if taskID != "" {
		query += ` AND task_id = ?`
		args = append(args, taskID)
	}
	if state != "" {
		query += ` AND state = ?`
		args = append(args, state)
	}
	query += ` ORDER BY id DESC`

	rows, err := db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var approvals []Approval
	for rows.Next() {
		var approval Approval
		var createdAt, resolvedAt string
		if err := rows.Scan(&approval.ID, &approval.TaskID, &approval.AgentName, &approval.Type, &approval.Title, &approval.Description, &approval.Risk, &approval.RecommendedAction, &approval.State, &approval.Resolution, &createdAt, &resolvedAt); err != nil {
			return nil, err
		}
		approval.CreatedAt = parseTime(createdAt)
		approval.ResolvedAt = parseOptionalTime(resolvedAt)
		approvals = append(approvals, approval)
	}
	return approvals, rows.Err()
}

func (s *Store) ResolveApproval(id int64, resolution string) (Approval, error) {
	if err := s.Init(); err != nil {
		return Approval{}, err
	}
	db, err := s.open()
	if err != nil {
		return Approval{}, err
	}
	defer db.Close()
	now := time.Now()
	result, err := db.ExecContext(context.Background(), `UPDATE approvals SET state = 'resolved', resolution = ?, resolved_at = ? WHERE id = ? AND state = 'pending'`, resolution, formatTime(now), id)
	if err != nil {
		return Approval{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Approval{}, err
	}
	if rows == 0 {
		return Approval{}, fmt.Errorf("approval %d not found or already resolved", id)
	}
	approvals, err := s.ListApprovals("", "")
	if err != nil {
		return Approval{}, err
	}
	for _, approval := range approvals {
		if approval.ID == id {
			return approval, nil
		}
	}
	return Approval{}, fmt.Errorf("approval %d not found after resolving", id)
}
