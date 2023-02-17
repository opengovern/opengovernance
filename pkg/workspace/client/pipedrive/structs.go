package pipedrive

import (
	"strconv"
	"strings"
)

type ValueWithPrimary struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

func GetPrimaryValue(values []ValueWithPrimary) string {
	for _, value := range values {
		if value.Primary {
			return value.Value
		}
	}
	return ""
}

type ContactPerson struct {
	OwnerID int                `json:"owner_id"`
	Name    string             `json:"name"`
	Emails  []ValueWithPrimary `json:"email"`
	Phones  []ValueWithPrimary `json:"phone"`
}

type PlatformAccessString string
type PlatformAccess int

const (
	PlatformAccessAllowDemoAccess         PlatformAccess = 16
	PlatformAccessAllowInstanceManagement PlatformAccess = 17
	PlatformAccessAllowInstanceCreation   PlatformAccess = 18
)

func (p PlatformAccessString) ParseIntoPlatformAccess() []PlatformAccess {
	platformAccess := make([]PlatformAccess, 0)
	platformAccessString := string(p)
	if platformAccessString == "" {
		return platformAccess
	}
	platformAccessStrings := strings.Split(platformAccessString, ",")
	for _, platformAccessString := range platformAccessStrings {
		intPlatformAccess, err := strconv.Atoi(platformAccessString)
		if err != nil {
			continue
		}
		platformAccess = append(platformAccess, PlatformAccess(intPlatformAccess))
	}
	return platformAccess
}

type Organization struct {
	ID                     int           `json:"id"`
	CompanyID              int           `json:"company_id"`
	Name                   string        `json:"name"`
	URL                    string        `json:"14292011fbc15ac15d9965bf7906d47acc74c62c"`
	AddressCountry         string        `json:"address_country"`
	AddressAdminAreaLevel1 string        `json:"address_admin_area_level_1"`
	AddressAdminAreaLevel2 string        `json:"address_admin_area_level_2"`
	AddressLocality        string        `json:"address_locality"`
	AddressPostalCode      string        `json:"address_postal_code"`
	Address                string        `json:"address"`
	PlatformAccessString   string        `json:"41a9dc0f3a7456f2be4214cdb84f77bf8d6b22aa"`
	Contact                ContactPerson `json:"5d41d55253c85327a14cbc7ca990c3ae4177220d"`
}
