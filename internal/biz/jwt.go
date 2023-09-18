package biz

import (
	"context"
	sender_v1 "notifications/api/send/v1"
	"os"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	jwtv4 "github.com/golang-jwt/jwt/v4"
)

type JwtProcessor struct {
	jwtSecret []byte
}

// NewJwtProcessor .
func NewJwtProcessor() (*JwtProcessor, error) {
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		if pair[0] == "JWT_SECRET" {
			jwtSecret := []byte(pair[1])
			return &JwtProcessor{
				jwtSecret: jwtSecret,
			}, nil
		}
	}

	return nil, sender_v1.ErrorInternal("JWT_SECRET not found")
}

func (j *JwtProcessor) GetSecret() []byte {
	return j.jwtSecret
}

func (j *JwtProcessor) GetUserIdFromContext(ctx context.Context) (int64, bool) {
	token, ok := jwt.FromContext(ctx)
	if !ok {
		return 0, false
	}

	claims, ok := token.(*jwtv4.RegisteredClaims)
	if !ok {
		return 0, false
	}

	userId, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, false
	}

	return userId, true
}
