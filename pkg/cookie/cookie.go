package cookie

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
)

const (
	AccessTokenName    = "access_token"
	RefreshTokenName   = "refresh_token"
	AccessTokenExpiry  = 15 * time.Minute
	RefreshTokenExpiry = 7 * 24 * time.Hour
	RefreshPath        = "/api/v1/auth/refresh-token"
	RootPath           = "/"
)

func isProduction() bool {
	return os.Getenv("APP_ENV") == "production"
}

func SetAccessToken(c fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     AccessTokenName,
		Value:    token,
		Expires:  time.Now().Add(AccessTokenExpiry),
		HTTPOnly: true,
		Secure:   isProduction(),
		SameSite: fiber.CookieSameSiteLaxMode,
		Path:     RootPath,
	})
}

func SetRefreshToken(c fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     RefreshTokenName,
		Value:    token,
		Expires:  time.Now().Add(RefreshTokenExpiry),
		HTTPOnly: true,
		Secure:   isProduction(),
		SameSite: fiber.CookieSameSiteLaxMode,
		Path:     RefreshPath,
	})
}

func SetAuthCookies(c fiber.Ctx, accessToken, refreshToken string) {
	SetAccessToken(c, accessToken)
	SetRefreshToken(c, refreshToken)
}

func ClearAuthCookies(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     AccessTokenName,
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Path:     RootPath,
	})
	c.Cookie(&fiber.Cookie{
		Name:     RefreshTokenName,
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Path:     RefreshPath,
	})
}

func GetRefreshToken(c fiber.Ctx) string {
	return c.Cookies(RefreshTokenName)
}
