package middleware

import (
	"strings"
	"time"

	"github.com/engrotech/common-utils/utils"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// RequireAccessToken validates Authorization: Bearer <accessToken> against the tokens collection.
// Only sessions with status "active" are accepted (use status "revoked" or similar to invalidate).
// On success sets c.Locals("userId") and c.Locals("session", bson.M). Returns 401 otherwise.
func RequireAccessToken(tokens *mongo.Collection) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := strings.TrimSpace(c.Get("Authorization"))
		if auth == "" {
			return unauthorized(c)
		}
		raw := auth
		if parts := strings.SplitN(auth, " ", 2); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			raw = strings.TrimSpace(parts[1])
		}
		if raw == "" {
			return unauthorized(c)
		}

		var doc bson.M
		err := tokens.FindOne(c.Context(), bson.M{
			"accessToken": raw,
			"status":      "active",
		}).Decode(&doc)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return unauthorized(c)
			}
			return unauthorized(c)
		}

		expiresAt, ok := utils.BSONDateToTime(doc["expiresAt"])
		if !ok || !time.Now().Before(expiresAt) {
			return unauthorized(c)
		}

		if uid, ok := doc["userId"].(string); ok && uid != "" {
			c.Locals("userId", uid)
		}
		c.Locals("session", doc)
		c.Locals("accessToken", auth)

		return c.Next()
	}
}

func unauthorized(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"success": false,
		"message": "Unauthorized",
		"data":    nil,
	})
}
