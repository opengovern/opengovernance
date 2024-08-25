package main

import (
	"os"

	_ "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	_ "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	_ "github.com/kaytu-io/kaytu-engine/pkg/docs" // docs is generated by Swag CLI, you have to import it.
	_ "github.com/kaytu-io/kaytu-util/pkg/api"
	_ "github.com/kaytu-io/kaytu-util/pkg/es"
	_ "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	_ "github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	swagger "github.com/swaggo/echo-swagger"
	_ "gorm.io/datatypes"
)

var (
	HttpAddress = os.Getenv("HTTP_ADDRESS")
)

//go:generate ../../scripts/generate_doc.sh

//	@title						Kaytu Service API
//	@version					1.0
//	@host						api.kaytu.io
//	@schemes					https
//	@securityDefinitions.apikey	BearerToken
//	@tokenUrl					https://example.com/oauth/token
//	@in							header
//	@name						Authorization
//	@description				Enter the token with the `Bearer` prefix.

func main() {
	e := echo.New()
	e.Logger.SetLevel(log.DEBUG) // TODO: change in prod
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Logger())

	e.GET("/swagger/*", swagger.WrapHandler)
	e.Logger.Fatal(e.Start(HttpAddress))
}
