package authz

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AccessType string

const (
	AccessPublic        AccessType = "public"
	AccessAuthenticated AccessType = "authenticated"
	AccessSubscription  AccessType = "subscription"
)

type OperationPermission struct {
	Operation       string     `bson:"operation"`
	RequiredFeature string     `bson:"requiredFeature"`
	Access          AccessType `bson:"access"`
	IsActive        bool       `bson:"isActive"`
}

type PlanFeature struct {
	FeatureKey string `bson:"featureKey"`
	Enabled    bool   `bson:"enabled"`
	Limit      int    `bson:"limit"`
}

type PlanEntitlement struct {
	Enabled *bool  `bson:"enabled,omitempty" json:"enabled,omitempty"`
	Level   string `bson:"level" json:"level"`
	Limit   int    `bson:"limit" json:"limit"`
}

func (e PlanEntitlement) IsEnabled() bool {
	if e.Enabled == nil {
		return true
	}
	return *e.Enabled
}

type SubscriptionPlan struct {
	Features     []PlanFeature              `bson:"features"`
	Entitlements map[string]PlanEntitlement `bson:"entitlements"`
}

type Subscription struct {
	Status             string            `bson:"status"`
	CurrentPeriodStart *time.Time        `bson:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd   *time.Time        `bson:"currentPeriodEnd,omitempty"`
	StartDate          *time.Time        `bson:"startDate,omitempty"`
	EndDate            *time.Time        `bson:"endDate,omitempty"`
	PlanSnapshot       *SubscriptionPlan `bson:"planSnapshot,omitempty"`
}

type UsageCounter struct {
	Used      int       `bson:"used"`
	Limit     int       `bson:"limit"`
	PeriodEnd time.Time `bson:"periodEnd"`
}

type Config struct {
	MongoClient             *mongo.Client
	DatabaseName            string
	PermissionsCollection   string
	SubscriptionsCollection string
	UsageCollection         string
	ServiceName             string
}

type SubscriptionGuard struct {
	config Config
	cache  map[string]OperationPermission
	mu     sync.RWMutex
}

func NewSubscriptionGuard(cfg Config) *SubscriptionGuard {
	return &SubscriptionGuard{
		config: cfg,
		cache:  make(map[string]OperationPermission),
	}
}

// Reload loads operation permissions from MongoDB into the memory cache
func (g *SubscriptionGuard) Reload(ctx context.Context) error {
	coll := g.config.MongoClient.Database(g.config.DatabaseName).Collection(g.config.PermissionsCollection)
	
	cursor, err := coll.Find(ctx, bson.M{"service": g.config.ServiceName, "isActive": true})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	temp := make(map[string]OperationPermission)
	for cursor.Next(ctx) {
		var p OperationPermission
		if err := cursor.Decode(&p); err == nil {
			temp[p.Operation] = p
		}
	}

	g.mu.Lock()
	g.cache = temp
	g.mu.Unlock()
	return nil
}

func (g *SubscriptionGuard) RequireOperation(operation string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		g.mu.RLock()
		perm, ok := g.cache[operation]
		g.mu.RUnlock()

		if !ok || perm.Access == AccessPublic {
			return c.Next()
		}

		userId, _ := c.Locals("userId").(string)
		if userId == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "User context is required",
			})
		}

		if perm.Access == AccessSubscription {
			ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
			defer cancel()

			db := g.config.MongoClient.Database(g.config.DatabaseName)

			// 1. Fetch active subscriptions for user
			now := time.Now().UTC()
			subFilter := bson.M{
				"userId": userId,
				"status": "active",
			}
			
			cursor, err := db.Collection(g.config.SubscriptionsCollection).Find(ctx, subFilter)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "Failed to read subscriptions"})
			}
			defer cursor.Close(ctx)

			var maxLimit int = 0
			var hasAccess bool = false
			var unlimited bool = false

			for cursor.Next(ctx) {
				var sub Subscription
				if err := cursor.Decode(&sub); err != nil {
					continue
				}

				// Check active time window
				if sub.StartDate != nil && sub.EndDate != nil {
					if now.Before(*sub.StartDate) || now.After(*sub.EndDate) {
						continue
					}
				} else if sub.CurrentPeriodStart != nil && sub.CurrentPeriodEnd != nil {
					if now.Before(*sub.CurrentPeriodStart) || now.After(*sub.CurrentPeriodEnd) {
						continue
					}
				}

				if sub.PlanSnapshot == nil {
					continue
				}

				// Resolve entitlement from snapshot
				var ent PlanEntitlement
				found := false

				// Check V2 Entitlements map
				if sub.PlanSnapshot.Entitlements != nil {
					if e, ok := sub.PlanSnapshot.Entitlements[perm.RequiredFeature]; ok {
						ent = e
						found = true
					}
				}

				// Fallback to legacy features array
				if !found && sub.PlanSnapshot.Features != nil {
					for _, f := range sub.PlanSnapshot.Features {
						if f.FeatureKey == perm.RequiredFeature && f.Enabled {
							limitVal := f.Limit
							if limitVal == 0 {
								limitVal = -1
							}
							ent = PlanEntitlement{
								Limit: limitVal,
							}
							found = true
							break
						}
					}
				}

				if found && ent.IsEnabled() {
					hasAccess = true
					if ent.Limit == -1 {
						unlimited = true
					} else if ent.Limit > maxLimit {
						maxLimit = ent.Limit
					}
				}
			}

			if !hasAccess {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"success": false,
					"message": fmt.Sprintf("Access denied. Plan subscription required for: %s", perm.RequiredFeature),
				})
			}

			// 2. Validate usage limit if not unlimited
			if !unlimited {
				var counter UsageCounter
				err := db.Collection(g.config.UsageCollection).FindOne(ctx, bson.M{
					"userId":      userId,
					"featureCode": perm.RequiredFeature,
				}).Decode(&counter)
				
				if err == nil {
					if now.Before(counter.PeriodEnd) && counter.Used >= maxLimit {
						return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
							"success": false,
							"message": "Usage limit exceeded for this feature",
						})
					}
				}
			}
		}

		return c.Next()
	}
}
