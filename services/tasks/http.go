package tasks

import (
	"context"
	"crypto/rsa"
	"crypto/sha512"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	api2 "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/opengovern/opencomply/services/auth/db"

	"github.com/golang-jwt/jwt"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// var (
// 	//go:embed email/invite.html
// 	inviteEmailTemplate string
// )

type httpRoutes struct {
	logger *zap.Logger

	platformPrivateKey *rsa.PrivateKey
	db                 db.Database
	authServer         *Server
}

func (r *httpRoutes) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")
	

}

func bindValidate(ctx echo.Context, i interface{}) error {
	if err := ctx.Bind(i); err != nil {
		return err
	}

	if err := ctx.Validate(i); err != nil {
		return err
	}

	return nil
}
