package authV2

import (
	"context"
	"crypto/rsa"
	"crypto/sha512"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	dexApi "github.com/dexidp/dex/api/v2"
	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	api2 "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/opengovernance/pkg/authV2/utils"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/opengovern/opengovernance/pkg/authV2/db"

	"github.com/golang-jwt/jwt"

	"github.com/labstack/echo/v4"
	"github.com/opengovern/opengovernance/pkg/authV2/api"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// var (
// 	//go:embed email/invite.html
// 	inviteEmailTemplate string
// )

type httpRoutes struct {
	logger *zap.Logger
	
	kaytuPrivateKey   *rsa.PrivateKey
	db                db.Database
	authServer        *Server
}

func (r *httpRoutes) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")
	v1.GET("/check", r.Check)
	v1.GET("/users", httpserver.AuthorizeHandler(r.GetUsers, api2.EditorRole)) //checked
	v1.GET("/user/:id", httpserver.AuthorizeHandler(r.GetUserDetails, api2.EditorRole)) //checked
	v1.GET("/me", httpserver.AuthorizeHandler(r.GetMe, api2.EditorRole)) //checked
	v1.POST("/keys", httpserver.AuthorizeHandler(r.CreateAPIKey, api2.EditorRole)) //checked
	v1.GET("/keys", httpserver.AuthorizeHandler(r.ListAPIKeys, api2.EditorRole)) //checked
	v1.DELETE("/keys/:id", httpserver.AuthorizeHandler(r.DeleteAPIKey, api2.EditorRole))
	// TODO: API FOR Edit keys
	v1.POST("/user/create", httpserver.AuthorizeHandler(r.CreateUser, api2.EditorRole))
	v1.POST("/user/update", httpserver.AuthorizeHandler(r.UpdateUser, api2.EditorRole))
	v1.GET("/user/password/check", httpserver.AuthorizeHandler(r.CheckUserPasswordChangeRequired, api2.ViewerRole))
	v1.POST("/user/password/reset", httpserver.AuthorizeHandler(r.ResetUserPassword, api2.ViewerRole))
	v1.DELETE("/user/:email_address", httpserver.AuthorizeHandler(r.DeleteUser, api2.AdminRole))

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

func (r *httpRoutes) Check(ctx echo.Context) error {
	checkRequest := envoyauth.CheckRequest{
		Attributes: &envoyauth.AttributeContext{
			Request: &envoyauth.AttributeContext_Request{
				Http: &envoyauth.AttributeContext_HttpRequest{
					Headers: make(map[string]string),
				},
			},
		},
	}

	for k, v := range ctx.Request().Header {
		if len(v) == 0 {
			checkRequest.Attributes.Request.Http.Headers[k] = ""
		} else {
			checkRequest.Attributes.Request.Http.Headers[k] = v[0]
		}
	}
	originalUri, err := url.Parse(ctx.Request().Header.Get("X-Original-URI"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid original uri")
	}
	checkRequest.Attributes.Request.Http.Path = originalUri.Path
	checkRequest.Attributes.Request.Http.Method = ctx.Request().Header.Get("X-Original-Method")

	res, err := r.authServer.Check(ctx.Request().Context(), &checkRequest)
	if err != nil {
		return err
	}

	if res.Status.Code != int32(codes.OK) {
		return echo.NewHTTPError(http.StatusForbidden, res.Status.Message)
	}

	if res.GetOkResponse() == nil {
		return echo.NewHTTPError(http.StatusForbidden, "no ok response")
	}

	for _, header := range res.GetOkResponse().GetHeaders() {
		if header == nil || header.Header == nil {
			continue
		}
		ctx.Response().Header().Set(header.Header.Key, header.Header.Value)
	}

	return ctx.NoContent(http.StatusOK)
}


// GetUsers godoc
//
//	@Summary		List Users
//	@Description	Retrieves a list of users who are members of the workspace.
//	@Security		BearerToken
//	@Tags			users
//	@Produce		json
//	@Param			request	body	api.GetUsersRequest	false	"Request Body"
//	@Success		200		{array}	api.GetUsersResponse
//	@Router			/auth/api/v1/users [get]
func (r *httpRoutes) GetUsers(ctx echo.Context) error {
	
	var req api.GetUsersRequest
	if err := ctx.Bind(&req); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	users, err := r.db.GetUsers()
	if err != nil {
		return err
	}
	var resp []api.GetUsersResponse
	for _, u := range users {

		resp = append(resp, api.GetUsersResponse{
			UserID:        u.ID,
			UserName:      u.Username,
			Email:         u.Email,
			EmailVerified: u.EmailVerified,
			ExternalId:  u.ExternalId,
			LastActivity:  u.LastLogin,
			RoleName:      u.Role,
		})
	}
	return ctx.JSON(http.StatusOK, resp)
}

// GetUserDetails godoc
//
//	@Summary		Get User details
//	@Description	Returns user details by specified user id.
//	@Security		BearerToken
//	@Tags			users
//	@Produce		json
//	@Param			userId	path		string	true	"User ID"
//	@Success		200		{object}	api.GetUserResponse
//	@Router			/auth/api/v1/user/{userId} [get]
func (r *httpRoutes) GetUserDetails(ctx echo.Context) error {
	
	userID := ctx.Param("id")
	userID, err := url.QueryUnescape(userID)
	if err != nil {
		return err
	}
	user, err := r.db.GetUser(userID)
	if err != nil {
		return err
	}
	
	status := api.InviteStatus_PENDING
	if user.EmailVerified {
		status = api.InviteStatus_ACCEPTED
	}
	resp := api.GetUserResponse{
		UserID:        user.ID,
		UserName:      user.Username,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Status:        status,
		LastActivity:  user.LastLogin,
		CreatedAt:     user.CreatedAt,
		Blocked:       user.IsActive,
		RoleName:      user.Role,
	}
	// check if LastLogin is Default go time value remove it 
	if (user.LastLogin.IsZero()) {
		resp.LastActivity = nil
	}





	return ctx.JSON(http.StatusOK, resp)

}

// GetMe godoc
//
//	@Summary		Get Me
//	@Description	Returns my user details
//	@Security		BearerToken
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	api.GetMeResponse
//	@Router			/auth/api/v1/me [get]
func (r *httpRoutes) GetMe(ctx echo.Context) error {
	userID := httpserver.GetUserID(ctx)

	user, err := utils.GetUser(userID,r.db)
	if err != nil {
		return err
	}

	status := api.InviteStatus_PENDING
	if user.EmailVerified {
		status = api.InviteStatus_ACCEPTED
	}
	resp := api.GetMeResponse{
		UserID:          user.ID,
		UserName:        user.Username,
		Email:           user.Email,
		EmailVerified:   user.EmailVerified,
		Status:          status,
		LastActivity:    user.LastLogin,
		CreatedAt:       user.CreatedAt,
		Blocked:         user.IsActive,
		Role: user.Role,
		MemberSince:     user.CreatedAt,
		LastLogin:       user.LastLogin,
	}
	if (user.LastLogin.IsZero()) {
		resp.LastLogin = nil
		resp.LastActivity = nil

	}



	return ctx.JSON(http.StatusOK, resp)

}




// CreateAPIKey godoc
//
//	@Summary		Create Workspace Key
//	@Description	Creates workspace key for the defined role with the defined name in the workspace.
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Param			request	body		api.CreateAPIKeyRequest	true	"Request Body"
//	@Success		200		{object}	api.CreateAPIKeyResponse
//	@Failure		406		{object}	echo.HTTPError
//	@Router			/auth/api/v1/key/create [post]
func (r *httpRoutes) CreateAPIKey(ctx echo.Context) error {
	userID := httpserver.GetUserID(ctx)
	var req api.CreateAPIKeyRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	usr, err := utils.GetUser(userID,r.db)
	if err != nil {
		r.logger.Error("failed to get user", zap.Error(err))
		return err
	}

	if usr == nil {
		return errors.New("failed to find user in auth0")
	}

	u := userClaim{
		Role: api2.EditorRole,
		
		Email:          usr.Email,
		ExternalUserID: usr.ExternalId,
	}

	if r.kaytuPrivateKey == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "kaytu api key is disabled")
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, &u).SignedString(r.kaytuPrivateKey)
	if err != nil {
		r.logger.Error("failed to create token", zap.Error(err))
		return err
	}

	masked := fmt.Sprintf("%s...%s", token[:10], token[len(token)-10:])

	hash := sha512.New()
	_, err = hash.Write([]byte(token))
	if err != nil {
		r.logger.Error("failed to hash token", zap.Error(err))
		return err
	}
	keyHash := hex.EncodeToString(hash.Sum(nil))
	r.logger.Info("hashed token")

	currentKeyCount, err := r.db.CountApiKeysForUser(userID)
	if err != nil {
		r.logger.Error("failed to get user API Keys count", zap.Error(err))
		return err
	}
	if currentKeyCount > 5 {
		return echo.NewHTTPError(http.StatusNotAcceptable, "maximum number of keys for user reached")
	}
	r.logger.Info("creating API Key")
	apikey := db.ApiKey{
		Name:          req.Name,
		Role:          req.Role,
		CreatorUserID: userID,
		
		IsActive:        true,
		IsDeleted:       false,
		MaskedKey:     masked,
		KeyHash:       keyHash,
	}

	r.logger.Info("adding API Key")
	err = r.db.AddApiKey(&apikey)
	if err != nil {
		r.logger.Error("failed to add API Key", zap.Error(err))
		return err
	}

	return ctx.JSON(http.StatusOK, api.CreateAPIKeyResponse{
		ID:        apikey.ID,
		Name:      apikey.Name,
		Active:    apikey.IsActive,
		CreatedAt: apikey.CreatedAt,
		RoleName:  apikey.Role,
		Token:     token,
	})
}

// DeleteAPIKey godoc
//
//	@Summary		Delete Workspace Key
//	@Description	Deletes the specified workspace key by ID.
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Param			id	path		string	true	"Key ID"
//	@Success		200	{object}	nil
//	@Router			/auth/api/v1/key/{id}/delete [delete]
func (r *httpRoutes) DeleteAPIKey(ctx echo.Context) error {
	// userId := httpserver.GetUserID(ctx)
	id := ctx.Param("id")

	// keys, err := r.db.ListApiKeysForUser(userId)
	// if err != nil {
	// 	return err
	// }

	
	exists := false
	// for _, key := range keys {
	// 	if key.Name == name {
	// 		keyId = key.ID
	// 		exists = true
	// 	}
	// }

	if !exists {
		return echo.NewHTTPError(http.StatusBadRequest, "key not found")
	}
	integer_id, err :=(strconv.ParseUint(id, 10, 32))
	err = r.db.DeleteAPIKey(integer_id)
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// ListAPIKeys godoc
//
//	@Summary		Get Workspace Keys
//	@Description	Gets list of all keys in the workspace.
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Success		200	{object}	[]api.WorkspaceApiKey
//	@Router			/auth/api/v1/keys [get]
func (r *httpRoutes) ListAPIKeys(ctx echo.Context) error {
	userID := httpserver.GetUserID(ctx)
	keys, err := r.db.ListApiKeysForUser(userID)
	if err != nil {
		return err
	}

	var resp []api.WorkspaceApiKey
	for _, key := range keys {
		resp = append(resp, api.WorkspaceApiKey{
			ID:            key.ID,
			CreatedAt:     key.CreatedAt,
			Name:          key.Name,
			RoleName:      key.Role,
			CreatorUserID: key.CreatorUserID,
			Active:        key.IsActive,
			MaskedKey:     key.MaskedKey,
		})
	}

	return ctx.JSON(http.StatusOK, resp)
}


// CreateUser godoc
//
//	@Summary		Create User
//	@Description	Creates User.
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Param			request	body	api.CreateUserRequest	true	"Request Body"
//	@Success		200
//	@Router			/auth/api/v3/user/create [post]
func (r *httpRoutes) CreateUser(ctx echo.Context) error {

	var req api.CreateUserRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	err := r.DoCreateUser(req)
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

func (r *httpRoutes) DoCreateUser(req api.CreateUserRequest) error {

	if req.EmailAddress == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email address is required")
	}

	user, err := r.db.GetUserByEmail(req.EmailAddress)
	if user != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "email already used")
	}

	count, err := r.db.GetUsersCount()
	if err != nil {
		r.logger.Error("failed to get users count", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get users count")
	}
	adminAccount := false
	var firstUser *db.User


	if count == 1 {
		firstUser, err = r.db.GetFirstUser()
		if err != nil {
			r.logger.Error("failed to get first user", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to get first user")
		}
		
	} else if count == 0 {
		adminAccount = true
	}
	if adminAccount && (req.Role == nil || *req.Role != api2.AdminRole) {
		return echo.NewHTTPError(http.StatusBadRequest, "You should define admin role")
	}

	if adminAccount && firstUser != nil {
		err = r.DoDeleteUser(firstUser.Email)
		if err != nil {
			r.logger.Error("failed to delete first user", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete first user")
		}
	}

	connector := ""
	userId := fmt.Sprintf("dex|%s", req.EmailAddress)
	if req.Password != nil {
		connector = "local"
		dexClient, err := newDexClient(dexGrpcAddress)
		if err != nil {
			r.logger.Error("failed to create dex client", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex client")
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			r.logger.Error("failed to hash token", zap.Error(err))
			return err
		}

		dexReq := &dexApi.CreatePasswordReq{
			Password: &dexApi.Password{
				UserId: userId,
				Email:  req.EmailAddress,
				Hash:   hashedPassword,
			},
		}

		resp, err := dexClient.CreatePassword(context.TODO(), dexReq)
		if err != nil {
			r.logger.Error("failed to create dex password", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex password")
		}
		if resp.AlreadyExists {
			dexReq := &dexApi.UpdatePasswordReq{
				Email:   req.EmailAddress,
				NewHash: hashedPassword,
			}

			_, err = dexClient.UpdatePassword(context.TODO(), dexReq)
			if err != nil {
				r.logger.Error("failed to update dex password", zap.Error(err))
				return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex password")
			}
		}
	}

	

	role := api2.ViewerRole
	if req.Role != nil {
		role = *req.Role
	}

	

	


	requirePasswordChange := true
	if adminAccount {
		requirePasswordChange = false
	}

	newUser := &db.User{
		Email:                 req.EmailAddress,
		Username:              req.EmailAddress,
		Role:                  role,
		EmailVerified:         false,
		Connector:             connector,
		ExternalId: userId,
		RequirePasswordChange: requirePasswordChange,
	}
	err = r.db.CreateUser(newUser)
	if err != nil {
		r.logger.Error("failed to create user", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create user")
	}
	return nil
}

// UpdateUser godoc
//
//	@Summary		Update User
//	@Description	Updates User.
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Param			request	body	api.UpdateUserRequest	true	"Request Body"
//	@Success		200
//	@Router			/auth/api/v3/user/update [post]
func (r *httpRoutes) UpdateUser(ctx echo.Context) error {
	var req api.UpdateUserRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if req.EmailAddress == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email address is required")
	}

	user, err := r.db.GetUserByEmail(req.EmailAddress)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
	}
	if user == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "user not found")
	}

	if req.Password != nil && user.Connector == "local" {
		dexClient, err := newDexClient(dexGrpcAddress)
		if err != nil {
			r.logger.Error("failed to create dex client", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex client")
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			r.logger.Error("failed to hash token", zap.Error(err))
			return err
		}

		dexReq := &dexApi.UpdatePasswordReq{
			Email:   req.EmailAddress,
			NewHash: hashedPassword,
		}

		resp, err := dexClient.UpdatePassword(context.TODO(), dexReq)
		if err != nil {
			r.logger.Error("failed to update dex password", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex password")
		}
		if resp.NotFound {
			dexReq := &dexApi.CreatePasswordReq{
				Password: &dexApi.Password{
					UserId: fmt.Sprintf("dex|%s", req.EmailAddress),
					Email:  req.EmailAddress,
					Hash:   hashedPassword,
				},
			}

			_, err = dexClient.CreatePassword(context.TODO(), dexReq)
			if err != nil {
				r.logger.Error("failed to create dex password", zap.Error(err))
				return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex password")
			}
		}

		err = r.db.UserPasswordUpdate(user.ID)
		if err != nil {
			r.logger.Error("failed to update user", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to update user")
		}
	}

	

	if req.Role != nil {
		 update_user :=&db.User{
			Model: gorm.Model{
				ID: user.ID,
			},
			Role: *req.Role,
			IsActive: req.IsActive,

		}
		err = r.db.UpdateUser(update_user)
		if err != nil {
			r.logger.Error("failed to update user", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to update user")
		}
	}

	return ctx.NoContent(http.StatusOK)
}

// DeleteUser godoc
//
//	@Summary		Delete User
//	@Description	Delete User.
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Param			email_address	path	string	true	"Request Body"
//	@Success		200
//	@Router			/auth/api/v3/user/{email_address}/delete [delete]
func (r *httpRoutes) DeleteUser(ctx echo.Context) error {
	emailAddress := ctx.Param("email_address")
	if emailAddress == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email address is required")
	}

	err := r.DoDeleteUser(emailAddress)
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

func (r *httpRoutes) DoDeleteUser(emailAddress string) error {
	dexClient, err := newDexClient(dexGrpcAddress)
	if err != nil {
		r.logger.Error("failed to create dex client", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex client")
	}

	dexReq := &dexApi.DeletePasswordReq{
		Email: emailAddress,
	}

	_, err = dexClient.DeletePassword(context.TODO(), dexReq)
	if err != nil {
		r.logger.Error("failed to create dex password", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex password")
	}

	user, err := r.db.GetUserByEmail(emailAddress)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "user does not exist")
	}
	if user == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "user does not exist")
	}

	
	err = r.db.DeleteUser(user.ID)
	if err != nil {
		r.logger.Error("failed to delete user", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create user")
	}
	return nil
}

func newDexClient(hostAndPort string) (dexApi.DexClient, error) {
	conn, err := grpc.NewClient(hostAndPort, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("dial: %v", err)
	}
	return dexApi.NewDexClient(conn), nil
}

// CheckUserPasswordChangeRequired godoc
//
//	@Summary		Delete User
//	@Description	Delete User.
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Success		200
//	@Router			/auth/api/v3/user/password/check [get]
func (r *httpRoutes) CheckUserPasswordChangeRequired(ctx echo.Context) error {
	userId := httpserver.GetUserID(ctx)

	user, err := r.db.GetUser(userId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
	}
	if user == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "user not found")
	}

	if user.RequirePasswordChange {
		return ctx.String(http.StatusOK, "CHANGE_REQUIRED")
	} else {
		return ctx.String(http.StatusOK, "CHANGE_NOT_REQUIRED")
	}
}

// ResetUserPassword godoc
//
//	@Summary		Reset current user password
//	@Description	Reset current user password
//	@Security		BearerToken
//	@Tags			keys
//	@Produce		json
//	@Success		200
//	@Router			/auth/api/v3/user/password/reset [post]
func (r *httpRoutes) ResetUserPassword(ctx echo.Context) error {
	userId := httpserver.GetUserID(ctx)

	user, err := r.db.GetUser(userId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
	}
	if user == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "user not found")
	}

	var req api.ResetUserPasswordRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if user.Connector != "local" {
		return echo.NewHTTPError(http.StatusBadRequest, "user connector should be local")
	}

	dexClient, err := newDexClient(dexGrpcAddress)
	if err != nil {
		r.logger.Error("failed to create dex client", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex client")
	}

	dexReq := &dexApi.VerifyPasswordReq{
		Email:    user.Email,
		Password: req.CurrentPassword,
	}

	resp, err := dexClient.VerifyPassword(context.TODO(), dexReq)
	if err != nil {
		r.logger.Error("failed to validate dex password", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to validate dex password")
	}
	if resp.NotFound {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}
	if !resp.Verified {
		return echo.NewHTTPError(http.StatusUnauthorized, "current password is not correct")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		r.logger.Error("failed to hash token", zap.Error(err))
		return err
	}

	passwordUpdateReq := &dexApi.UpdatePasswordReq{
		Email:   user.Email,
		NewHash: hashedPassword,
	}

	passwordUpdateResp, err := dexClient.UpdatePassword(context.TODO(), passwordUpdateReq)
	if err != nil {
		r.logger.Error("failed to update dex password", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to update dex password")
	}
	if passwordUpdateResp.NotFound {
		dexReq := &dexApi.CreatePasswordReq{
			Password: &dexApi.Password{
				UserId: fmt.Sprintf("dex|%s", user.Email),
				Email:  user.Email,
				Hash:   hashedPassword,
			},
		}

		_, err = dexClient.CreatePassword(context.TODO(), dexReq)
		if err != nil {
			r.logger.Error("failed to create dex password", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to create dex password")
		}
	}

	err = r.db.UserPasswordUpdate(user.ID)
	if err != nil {
		r.logger.Error("failed to update user", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to update user")
	}

	return ctx.NoContent(http.StatusOK)
}
