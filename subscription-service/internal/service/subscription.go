package service

import (
	"context"
	"subscription-service/internal/models"
	"subscription-service/internal/repository"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type SubscriptionService interface {
	CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error)
	GetSubscription(ctx context.Context, id uuid.UUID) (*models.Subscription, error)
	UpdateSubscription(ctx context.Context, id uuid.UUID, req *models.UpdateSubscriptionRequest) error
	DeleteSubscription(ctx context.Context, id uuid.UUID) error
	ListSubscriptions(ctx context.Context, filter *models.SubscriptionFilter) ([]*models.Subscription, error)
	GetTotalCost(ctx context.Context, filter *models.SubscriptionFilter) (int, error)
}

type subscriptionService struct {
	repo repository.SubscriptionRepository
}

func NewSubscriptionService(repo repository.SubscriptionRepository) SubscriptionService {
	return &subscriptionService{repo: repo}
}

func (s *subscriptionService) CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error) {
	startDate, err := time.Parse("01-2006", req.StartDate)
	if err != nil {
		return nil, errors.Wrap(err, "invalid start date format")
	}

	var endDate *time.Time
	if req.EndDate != nil {
		parsedEndDate, err := time.Parse("01-2006", *req.EndDate)
		if err != nil {
			return nil, errors.Wrap(err, "invalid end date format")
		}
		endDate = &parsedEndDate
	}

	subscription := &models.Subscription{
		ID:          uuid.New(),
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      req.UserID,
		StartDate:   startDate,
		EndDate:     endDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, subscription); err != nil {
		return nil, errors.Wrap(err, "failed to create subscription in repository")
	}

	return subscription, nil
}

func (s *subscriptionService) GetSubscription(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	subscription, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get subscription from repository")
	}
	if subscription == nil {
		return nil, errors.New("subscription not found")
	}
	return subscription, nil
}

func (s *subscriptionService) UpdateSubscription(ctx context.Context, id uuid.UUID, req *models.UpdateSubscriptionRequest) error {
	if _, err := s.GetSubscription(ctx, id); err != nil {
		return err
	}

	return s.repo.Update(ctx, id, req)
}

func (s *subscriptionService) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	if _, err := s.GetSubscription(ctx, id); err != nil {
		return err
	}

	return s.repo.Delete(ctx, id)
}

func (s *subscriptionService) ListSubscriptions(ctx context.Context, filter *models.SubscriptionFilter) ([]*models.Subscription, error) {
	return s.repo.List(ctx, filter)
}

func (s *subscriptionService) GetTotalCost(ctx context.Context, filter *models.SubscriptionFilter) (int, error) {
	return s.repo.GetTotalCost(ctx, filter)
}
