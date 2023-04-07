package cmd

type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationUrl         string `json:"verification_uri"`
	VerificationUrlComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type DeviceCodeRequest struct {
	ClientId string `json:"client_id"`
	Scope    string `json:"scope"`
	Audience string `json:"audience"`
}
type ResponseAccessToken struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	IdToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpireIn    string `json:"expire_in"`
}
type RequestAccessToken struct {
	GrantType  string `json:"grant_type"`
	DeviceCode string `json:"device_code"`
	ClientId   string `json:"client_id"`
}
type DataStoredInFile struct {
	AccessToken string `json:"access_token"`
}
type ResponseAbout struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}
