package extauth

type AzureADUser struct {
	ID                string          `json:"id,omitempty"`
	DisplayName       string          `json:"displayName,omitempty"`
	GivenName         string          `json:"givenName,omitempty"`
	Mail              string          `json:"mail,omitempty"`
	MobilePhone       string          `json:"mobilePhone,omitempty"`
	Surname           string          `json:"surname,omitempty"`
	UserPrincipalName string          `json:"userPrincipalName,omitempty"`
	AccountEnabled    bool            `json:"accountEnabled,omitempty"`
	Identities        []Identities    `json:"identities,omitempty"`
	PasswordProfile   PasswordProfile `json:"passwordProfile"`
	PasswordPolicies  string          `json:"passwordPolicies,omitempty"`
	Creationtype      string          `json:"creationType,omitempty"`
}

type Identities struct {
	SignInType       string `json:"signInType"`
	Issuer           string `json:"issuer"`
	IssuerAssignedId string `json:"issuerAssignedId"`
}

type PasswordProfile struct {
	ForceChangePasswordNextSignIn        bool   `json:"forceChangePasswordNextSignIn"`
	ForceChangePasswordNextSignInWithMfa bool   `json:"forceChangePasswordNextSignInWithMfa"`
	Password                             string `json:"password"`
}

type AzureErrors struct {
	Error Error `json:"error"`
}

type Error struct {
	Code       string     `json:"code"`
	Message    string     `json:"message"`
	Details    []Details  `json:"details"`
	InnerError InnerError `json:"innerError"`
}

type Details struct {
	Code    string `json:"code"`
	Target  string `json:"target"`
	Message string `json:"message"`
}

type InnerError struct {
	Date            string `json:"date"`
	RequestID       string `json:"request-id"`
	ClientRequestID string `json:"client-request-id"`
}
