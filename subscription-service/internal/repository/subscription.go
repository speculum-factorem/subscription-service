package repository

import (
	"context"
	"database/sql"
	"fmt"
	"subscription-service/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, sub *models.Subscription) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error)
	Update(ctx context.Context, id uuid.UUID, req *models.UpdateSubscriptionRequest) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *models.SubscriptionFilter) ([]*models.Subscription, error)
	GetTotalCost(ctx context.Context, filter *models.SubscriptionFilter) (int, error)
}

type subscriptionRepo struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) SubscriptionRepository {
	return &subscriptionRepo{db: db}
}

func (r *subscriptionRepo) Create(ctx context.Context, sub *models.Subscription) error {
	query := `
        INSERT INTO subscriptions (id, service_name, price, user_id, start_date, end_date, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `

	_, err := r.db.ExecContext(ctx, query,
		sub.ID, sub.ServiceName, sub.Price, sub.UserID, sub.StartDate, sub.EndDate, sub.CreatedAt, sub.UpdatedAt)

	return errors.Wrap(err, "failed to create subscription")
}

func (r *subscriptionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	query := `
        SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
        FROM subscriptions WHERE id = $1
    `

	var sub models.Subscription
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID, &sub.StartDate, &sub.EndDate, &sub.CreatedAt, &sub.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &sub, errors.Wrap(err, "failed to get subscription by id")
}

func (r *subscriptionRepo) Update(ctx context.Context, id uuid.UUID, req *models.UpdateSubscriptionRequest) error {
	query := "UPDATE subscriptions SET "
	args := []interface{}{}
	argPos := 1

	if req.ServiceName != nil {
		query += fmt.Sprintf("service_name = $%d, ", argPos)
		args = append(args, *req.ServiceName)
		argPos++
	}

	if req.Price != nil {
		query += fmt.Sprintf("price = $%d, ", argPos)
		args = append(args, *req.Price)
		argPos++
	}

	if req.StartDate != nil {
		startDate, err := time.Parse("01-2006", *req.StartDate)
		if err != nil {
			return errors.Wrap(err, "invalid start date format")
		}
		query += fmt.Sprintf("start_date = $%d, ", argPos)
		args = append(args, startDate)
		argPos++
	}

	if req.EndDate != nil {
		if *req.EndDate == "" {
			query += "end_date = NULL, "
		} else {
			endDate, err := time.Parse("01-2006", *req.EndDate)
			if err != nil {
				return errors.Wrap(err, "invalid end date format")
			}
			query += fmt.Sprintf("end_date = $%d, ", argPos)
			args = append(args, endDate)
			argPos++
		}
	}

	query += fmt.Sprintf("updated_at = $%d WHERE id = $%d", argPos, argPos+1)
	args = append(args, time.Now(), id)

	_, err := r.db.ExecContext(ctx, query, args...)
	return errors.Wrap(err, "failed to update subscription")
}

func (r *subscriptionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM subscriptions WHERE id = $1"
	_, err := r.db.ExecContext(ctx, query, id)
	return errors.Wrap(err, "failed to delete subscription")
}

func (r *subscriptionRepo) List(ctx context.Context, filter *models.SubscriptionFilter) ([]*models.Subscription, error) {
	query := `
        SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
        FROM subscriptions WHERE 1=1
    `
	args := []interface{}{}
	argPos := 1

	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argPos)
		args = append(args, *filter.UserID)
		argPos++
	}

	if filter.ServiceName != nil {
		query += fmt.Sprintf(" AND service_name ILIKE $%d", argPos)
		args = append(args, "%"+*filter.ServiceName+"%")
		argPos++
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list subscriptions")
	}
	defer rows.Close()

	var subscriptions []*models.Subscription
	for rows.Next() {
		var sub models.Subscription
		err := rows.Scan(
			&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID, &sub.StartDate, &sub.EndDate, &sub.CreatedAt, &sub.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan subscription")
		}
		subscriptions = append(subscriptions, &sub)
	}

	return subscriptions, nil
}

func (r *subscriptionRepo) GetTotalCost(ctx context.Context, filter *models.SubscriptionFilter) (int, error) {
	query := "SELECT COALESCE(SUM(price), 0) FROM subscriptions WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argPos)
		args = append(args, *filter.UserID)
		argPos++
	}

	if filter.ServiceName != nil {
		query += fmt.Sprintf(" AND service_name ILIKE $%d", argPos)
		args = append(args, "%"+*filter.ServiceName+"%")
		argPos++
	}

	if filter.StartDate != nil {
		startDate, err := time.Parse("01-2006", *filter.StartDate)
		if err != nil {
			return 0, errors.Wrap(err, "invalid start date format")
		}
		query += fmt.Sprintf(" AND start_date >= $%d", argPos)
		args = append(args, startDate)
		argPos++
	}

	if filter.EndDate != nil {
		endDate, err := time.Parse("01-2006", *filter.EndDate)
		if err != nil {
			return 0, errors.Wrap(err, "invalid end date format")
		}
		nextMonth := endDate.AddDate(0, 1, 0)
		query += fmt.Sprintf(" AND (start_date < $%d OR end_date IS NULL OR end_date < $%d)", argPos, argPos)
		args = append(args, nextMonth)
		argPos++
	}

	var totalCost int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&totalCost)
	return totalCost, errors.Wrap(err, "failed to calculate total cost")
}
