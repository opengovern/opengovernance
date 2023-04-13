package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/auth0"
)

type tokenStruct struct {
	AccessToken string
}

var testUsers = []auth0.User{
	{
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Email:         "user1@test.com",
		EmailVerified: true,
		FamilyName:    "testFamily",
		GivenName:     "testName",
		Locale:        "locale test",
		Name:          "testName",
		Nickname:      "testNick",
		Picture:       "testURL",
		UserId:        "test1",
		UserMetadata:  auth0.Metadata{},
		LastLogin:     time.Now(),
		LastIp:        "testIP",
		LoginsCount:   1,
		AppMetadata: auth0.Metadata{
			WorkspaceAccess: map[string]api.Role{"ws1": api.AdminRole, "ws2": api.EditorRole, "ws3": api.EditorRole, "ws4": api.ViewerRole},
		},
		Username:      "testUserName",
		PhoneNumber:   "testPhone",
		PhoneVerified: true,
		Multifactor:   []string{"testMultifactior"},
		Blocked:       false,
	},
	{
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Email:         "user1@test.com",
		EmailVerified: true,
		FamilyName:    "testFamily",
		GivenName:     "testName",
		Locale:        "locale test",
		Name:          "testName",
		Nickname:      "testNick",
		Picture:       "testURL",
		UserId:        "test2",
		UserMetadata:  auth0.Metadata{},
		LastLogin:     time.Now(),
		LastIp:        "testIP",
		LoginsCount:   1,
		AppMetadata: auth0.Metadata{
			WorkspaceAccess: map[string]api.Role{"ws1": api.EditorRole, "ws2": api.EditorRole},
		},
		Username:      "testUserName",
		PhoneNumber:   "testPhone",
		PhoneVerified: true,
		Multifactor:   []string{"testMultifactior"},
		Blocked:       false,
	},
	{
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Email:         "user1@test.com",
		EmailVerified: true,
		FamilyName:    "testFamily",
		GivenName:     "testName",
		Locale:        "locale test",
		Name:          "testName",
		Nickname:      "testNick",
		Picture:       "testURL",
		UserId:        "test3",
		UserMetadata:  auth0.Metadata{},
		LastLogin:     time.Now(),
		LastIp:        "testIP",
		LoginsCount:   1,
		AppMetadata: auth0.Metadata{
			WorkspaceAccess: map[string]api.Role{"ws1": api.ViewerRole, "ws2": api.EditorRole, "ws4": api.ViewerRole},
		},
		Username:      "testUserName",
		PhoneNumber:   "testPhone",
		PhoneVerified: true,
		Multifactor:   []string{"testMultifactior"},
		Blocked:       false,
	},
	{
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Email:         "user1@test.com",
		EmailVerified: true,
		FamilyName:    "testFamily",
		GivenName:     "testName",
		Locale:        "locale test",
		Name:          "testName",
		Nickname:      "testNick",
		Picture:       "testURL",
		UserId:        "test4",
		UserMetadata:  auth0.Metadata{},
		LastLogin:     time.Now(),
		LastIp:        "testIP",
		LoginsCount:   1,
		AppMetadata: auth0.Metadata{
			WorkspaceAccess: map[string]api.Role{"ws4": api.AdminRole},
		},
		Username:      "testUserName",
		PhoneNumber:   "testPhone",
		PhoneVerified: true,
		Multifactor:   []string{"testMultifactior"},
		Blocked:       false,
	},
	{
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Email:         "user1@test.com",
		EmailVerified: true,
		FamilyName:    "testFamily",
		GivenName:     "testName",
		Locale:        "locale test",
		Name:          "testName",
		Nickname:      "testNick",
		Picture:       "testURL",
		UserId:        "test5",
		UserMetadata:  auth0.Metadata{},
		LastLogin:     time.Now(),
		LastIp:        "testIP",
		LoginsCount:   1,
		AppMetadata: auth0.Metadata{
			WorkspaceAccess: map[string]api.Role{"ws2": api.AdminRole, "ws3": api.EditorRole, "ws4": api.ViewerRole},
		},
		Username:      "testUserName",
		PhoneNumber:   "testPhone",
		PhoneVerified: true,
		Multifactor:   []string{"testMultifactior"},
		Blocked:       false,
	},
	{
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Email:         "user1@test.com",
		EmailVerified: true,
		FamilyName:    "testFamily",
		GivenName:     "testName",
		Locale:        "locale test",
		Name:          "testName",
		Nickname:      "testNick",
		Picture:       "testURL",
		UserId:        "test6",
		UserMetadata:  auth0.Metadata{},
		LastLogin:     time.Now(),
		LastIp:        "testIP",
		LoginsCount:   1,
		AppMetadata: auth0.Metadata{
			WorkspaceAccess: map[string]api.Role{"ws1": api.ViewerRole, "ws2": api.ViewerRole, "ws3": api.ViewerRole, "ws4": api.ViewerRole},
		},
		Username:      "testUserName",
		PhoneNumber:   "testPhone",
		PhoneVerified: true,
		Multifactor:   []string{"testMultifactior"},
		Blocked:       false,
	},
}

func handlerFunc(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")
	fmt.Fprint(w, fmt.Sprintf("Welcome to the homepage! %s", id))
}

func mockDeleteUser(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func mockFillTocken(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	res := tokenStruct{
		AccessToken: "test",
	}
	json.NewEncoder(w).Encode(res)
}

func mockGetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := strings.TrimPrefix(r.URL.Path, "/api/v2/users/")
	fmt.Println("getuser: ", id)
	for _, user := range testUsers {
		if user.UserId == id {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(user)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func mockGetUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(testUsers)
}

func mockGetClient(w http.ResponseWriter, r *http.Request) {
	client := strings.TrimPrefix(r.URL.Path, "/api/v2/clients/")
	w.Header().Set("Content-Type", "application/json")
	tenant := map[string]string{
		"client": client,
		"tenant": "testTenant",
		"random": "123",
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tenant)
}

func mockPatchUser(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
