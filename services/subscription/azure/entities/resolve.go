package entities

import "time"

type AzureSaaSUser struct {
	EmailId  string `json:"emailId"`
	ObjectId string `json:"objectId"`
	TenantId string `json:"tenantId"`
	Puid     string `json:"puid"`
}

type AzureSaaSTerm struct {
	TermUnit  string    `json:"termUnit"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
}

type AzureSaaSSubscription struct {
	Id                        string        `json:"id"`
	PublisherId               string        `json:"publisherId"`
	OfferId                   string        `json:"offerId"`
	Name                      string        `json:"name"`
	SaasSubscriptionStatus    string        `json:"saasSubscriptionStatus"`
	Beneficiary               AzureSaaSUser `json:"beneficiary"`
	Purchaser                 AzureSaaSUser `json:"purchaser"`
	PlanId                    string        `json:"planId"`
	Term                      AzureSaaSTerm `json:"term"`
	AutoRenew                 bool          `json:"autoRenew"`
	IsTest                    bool          `json:"isTest"`
	IsFreeTrial               bool          `json:"isFreeTrial"`
	AllowedCustomerOperations []string      `json:"allowedCustomerOperations"`
	SandboxType               string        `json:"sandboxType"`
	LastModified              string        `json:"lastModified"`
	Quantity                  int           `json:"quantity"`
	SessionMode               string        `json:"sessionMode"`
}

type AzureSaaSResolveResponse struct {
	Id               string                `json:"id"`
	SubscriptionName string                `json:"subscriptionName"`
	OfferId          string                `json:"offerId"`
	PlanId           string                `json:"planId"`
	Quantity         int                   `json:"quantity"`
	Subscription     AzureSaaSSubscription `json:"subscription"`
}

type AzureSaaSGetAllSubscriptionsResponse struct {
	Subscriptions []AzureSaaSSubscription `json:"subscriptions"`
	NextLink      string                  `json:"@nextLink"`
}

type AzureSaaSOperation struct {
	Id             string `json:"id"`
	ActivityId     string `json:"activityId"`
	SubscriptionId string `json:"subscriptionId"`
	OfferId        string `json:"offerId"`
	PublisherId    string `json:"publisherId"`
	PlanId         string `json:"planId"`
	Quantity       int    `json:"quantity"`
	Action         string `json:"action"`
	TimeStamp      string `json:"timeStamp"`
	Status         string `json:"status"`
}

type AzureSaaSListOutstandingOperations struct {
	Operations []AzureSaaSOperation `json:"operations"`
}

type AzureSaaSGetOperationStatus struct {
	Id              string `json:"id  "`
	ActivityId      string `json:"activityId"`
	SubscriptionId  string `json:"subscriptionId"`
	OfferId         string `json:"offerId"`
	PublisherId     string `json:"publisherId"`
	PlanId          string `json:"planId"`
	Quantity        int    `json:"quantity"`
	Action          string `json:"action"`
	TimeStamp       string `json:"timeStamp"`
	Status          string `json:"status"`
	ErrorStatusCode string `json:"errorStatusCode"`
	ErrorMessage    string `json:"errorMessage"`
}

type AzureSaaSUpdateResponse struct {
	Status string `json:"status"`
}

type AzureSaaSUsageEventRequest struct {
	ResourceId         string    `json:"resourceId"`
	Quantity           float64   `json:"quantity"`
	Dimension          string    `json:"dimension"`
	EffectiveStartTime time.Time `json:"effectiveStartTime"`
	PlanId             string    `json:"planId"`
}

type AzureSaaSUsageEventResponse struct {
	UsageEventId       string    `json:"usageEventId"`
	Status             string    `json:"status"`
	MessageTime        time.Time `json:"messageTime"`
	ResourceId         string    `json:"resourceId"`
	Quantity           float64   `json:"quantity"`
	Dimension          string    `json:"dimension"`
	EffectiveStartTime string    `json:"effectiveStartTime"`
	PlanId             string    `json:"planId"`
}

type AzureSaaSUsageEventErrorResponse struct {
	Message string `json:"message"`
	Target  string `json:"target"`
	Details []struct {
		Message string `json:"message"`
		Target  string `json:"target"`
		Code    string `json:"code"`
	} `json:"details"`
	Code           string `json:"code"`
	AdditionalInfo struct {
		AcceptedMessage struct {
			UsageEventId       string    `json:"usageEventId"`
			Status             string    `json:"status"`
			MessageTime        time.Time `json:"messageTime"`
			ResourceId         string    `json:"resourceId"`
			Quantity           float64   `json:"quantity"`
			Dimension          string    `json:"dimension"`
			EffectiveStartTime time.Time `json:"effectiveStartTime"`
			PlanId             string    `json:"planId"`
		} `json:"acceptedMessage"`
	} `json:"additionalInfo"`
}
