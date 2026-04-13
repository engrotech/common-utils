package utils

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func SuccessResponse(c *fiber.Ctx, msg string, data interface{}) error {
	return c.JSON(fiber.Map{
		"success": true,
		"message": msg,
		"data":    data,
	})
}

func ErrorResponse(c *fiber.Ctx, msg string) error {
	return c.Status(400).JSON(fiber.Map{
		"success": false,
		"message": msg,
		"data":    nil,
	})
}
func IsValidEmail(email string) bool {
	var re = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	return re.MatchString(email)
}
func IsStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	var (
		hasUpper   = regexp.MustCompile(`[A-Z]`).MatchString(password)
		hasLower   = regexp.MustCompile(`[a-z]`).MatchString(password)
		hasNumber  = regexp.MustCompile(`[0-9]`).MatchString(password)
		hasSpecial = regexp.MustCompile(`[\W_]`).MatchString(password)
	)

	return hasUpper && hasLower && hasNumber && hasSpecial
}
func GenerateOTP() string {
	// 6-digit numeric OTP
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		// fallback, still deterministic length
		return "000000"
	}
	n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	return fmt.Sprintf("%06d", n%1000000)
}
func BSONDateToTime(v interface{}) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case primitive.DateTime:
		return t.Time(), true
	default:
		return time.Time{}, false
	}
}
