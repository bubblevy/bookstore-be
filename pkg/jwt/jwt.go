package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService interface {
	GenerateToken(userID uint, email string, role string) (string, error)
	GenerateRefreshToken(userID uint) (string, error)
	ValidateToken(tokenString string) (*jwt.Token, error)
	ValidateRefreshToken(tokenString string) (*jwt.Token, error)
}

type jwtService struct {
	secretKey []byte
	issuer    string
}

func NewJWTService(secretKey string) JWTService {
	return &jwtService{
		secretKey: []byte(secretKey),
		issuer:    "bookstore-api",
	}
}

func (s *jwtService) GenerateToken(userID uint, email string, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"type":    "access",
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"iss":     s.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("Failed to sign token: %w", err)
	}

	return signedToken, nil
}

func (s *jwtService) GenerateRefreshToken(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    "refresh",
		"exp":     time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iss":     s.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("Failed to sign refresh token: %w", err)
	}

	return signedToken, nil
}

func (s *jwtService) ValidateToken(tokenString string) (*jwt.Token, error) {
	return s.validate(tokenString, "access")
}

func (s *jwtService) ValidateRefreshToken(tokenString string) (*jwt.Token, error) {
	return s.validate(tokenString, "refresh")
}

func (s *jwtService) validate(tokenString string, tokenType string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf(
				"Unexpected signing method: %v",
				token.Header["alg"],
			)
		}

		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("Invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("Invalid token claims")
	}

	if claims["type"] != tokenType {
		return nil, fmt.Errorf("Invalid token type")
	}

	if claims["iss"] != s.issuer {
		return nil, fmt.Errorf("Invalid token issuer")
	}

	return token, nil
}
