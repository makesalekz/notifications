package data

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	jwtv4 "github.com/golang-jwt/jwt/v4"
)

type JwtProcessor struct {
	jwtSecret []byte
}

type TenantClaims struct {
	jwtv4.RegisteredClaims

	TenantId  int64    `json:"tenant,omitempty"`
	MemberId  string   `json:"member,omitempty"`
	GroupsIds []string `json:"groups,omitempty"`
}

// NewJwtProcessor .
func NewJwtProcessor(c *Config) (*JwtProcessor, error) {
	secret, err := c.ReadGlobalSecretsFor(context.Background(), "jwt")
	if err != nil {
		return nil, fmt.Errorf("jwt secret not found, error: %w", err)
	}

	return &JwtProcessor{
		jwtSecret: []byte(secret["data"].(string)),
	}, nil
}

func (j *JwtProcessor) GetSecret() []byte {
	return j.jwtSecret
}

func (j *JwtProcessor) GetUserIdFromContext(ctx context.Context) (int64, bool) {
	claims := j.GetClaimsFromContext(ctx)
	if claims == nil {
		return 0, false
	}

	userId, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, false
	}

	return userId, true
}

func (j *JwtProcessor) GetClaimsFromContext(ctx context.Context) *jwtv4.RegisteredClaims {
	token, ok := jwt.FromContext(ctx)
	if !ok {
		return nil
	}

	claims, ok := token.(*jwtv4.RegisteredClaims)
	if !ok {
		return nil
	}

	return claims
}

func (j *JwtProcessor) GetTenantClaimsFromContext(ctx context.Context) (*TenantClaims, bool) {
	token, ok := jwt.FromContext(ctx)
	if !ok {
		return nil, false
	}

	claims, ok := token.(*TenantClaims)
	if !ok {
		return nil, false
	}

	return claims, true
}

func (tc *TenantClaims) GetUserId() int64 {
	userId, err := strconv.ParseInt(tc.Subject, 10, 64)
	if err != nil {
		return 0
	}

	return userId
}

func (tc *TenantClaims) GetTenantId() int64 {
	return tc.TenantId
}

func (tc *TenantClaims) GetIdentities() []string {
	return append(tc.GroupsIds, tc.MemberId)
}
