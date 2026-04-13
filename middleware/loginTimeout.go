package middleware

import (
	"time"

	"github.com/engrotech/common-utils/utils"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// RequireSameDayTwoFA checks that the twoFAChallenges.updatedAt for (email, deviceId) is on the same calendar day as "now".
// If not same-day, it returns HTTP 408 (Request Timeout).
//
// This is intended to be used on OTP verification endpoints to prevent using older challenges.
func LoginTimeout(twoFAChallenges *mongo.Collection) fiber.Handler {
	type reqShape struct {
		Email    string `json:"email"`
		DeviceID string `json:"deviceId"`
	}
	return func(c *fiber.Ctx) error {
		userId := c.Locals("userId")
		var doc bson.M
		err := twoFAChallenges.FindOne(c.Context(), bson.M{
			"userId": userId,
		}).Decode(&doc)
		if err != nil {
			// If there's no challenge doc, let Verify2FA return its existing error.

			return requestTimeout(c)
		}
		updatedAt, ok := utils.BSONDateToTime(doc["updatedAt"])
		if !ok {

			return requestTimeout(c)
		}
		now := time.Now()
		y1, m1, d1 := now.Date()
		y2, m2, d2 := updatedAt.Date()
		if y1 != y2 || m1 != m2 || d1 != d2 {
			return requestTimeout(c)
		}
		return c.Next()
	}
}
func requestTimeout(c *fiber.Ctx) error {
	return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
		"success": false,
		"message": "Request timeout",
		"data":    nil,
	})
}
