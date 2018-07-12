package azure

//go:generate mockgen -destination=./client_mock.go -package=azure -source=client.go

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-05-31/consumption"
	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"
	"github.com/Azure/azure-sdk-for-go/services/preview/subscription/mgmt/2018-03-01-preview/subscription"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// Client interface is made to simplify testing
type Client interface {
	getPeriodIterator(subscriptionID, filter string) (periodsIterator, error)
	getUsageIterator(subscriptionID, billingPeriod, filter string) (usageIterator, error)
	getSubscriptionIterator() (subscriptionIterator, error)
}

// RestClient is a simple implementation of Client
type RestClient struct {
	authorizer             autorest.Authorizer
	newSubscriptionsClient func() subscriptionClient
	newPeriodsClient       func(input string) billingClient
	newUsageDetailsClient  func(input string) consumptionClient
}

type billingClient interface {
	ListComplete(ctx context.Context, filter string, skiptoken string, top *int32) (result billing.PeriodsListResultIterator, err error)
}

type consumptionClient interface {
	ListByBillingPeriodComplete(ctx context.Context, billingPeriodName string, expand string, filter string, apply string, skiptoken string, top *int32) (result consumption.UsageDetailsListResultIterator, err error)
}

type subscriptionClient interface {
	ListComplete(ctx context.Context) (result subscription.ListResultIterator, err error)
}

type usageIterator interface {
	Next() error
	NotDone() bool
	Value() consumption.UsageDetail
}

type periodsIterator interface {
	Next() error
	NotDone() bool
	Value() billing.Period
}

type subscriptionIterator interface {
	Next() error
	NotDone() bool
	Value() subscription.Model
}

// NewRestClient returns a RestClient for the given subscription ID.
func NewRestClient() Client {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	restClient := RestClient{}

	restClient.newSubscriptionsClient = func() subscriptionClient {
		client := subscription.NewSubscriptionsClient()
		client.Authorizer = authorizer
		return client
	}
	restClient.newPeriodsClient = func(input string) billingClient {
		client := billing.NewPeriodsClient(input)
		client.Authorizer = authorizer
		return client
	}
	restClient.newUsageDetailsClient = func(input string) consumptionClient {
		client := consumption.NewUsageDetailsClient(input)
		client.Authorizer = authorizer
		return client
	}

	return restClient
}

func (c RestClient) getPeriodIterator(subscriptionID, filter string) (periodsIterator, error) {
	periodsClient := c.newPeriodsClient(subscriptionID)
	var top int32 = 100
	result, err := periodsClient.ListComplete(context.Background(), filter, "", &top)
	return &result, err
}

func (c RestClient) getUsageIterator(subscriptionID, billingPeriod, filter string) (usageIterator, error) {
	usageClient := c.newUsageDetailsClient(subscriptionID)
	var top int32 = 100
	result, err := usageClient.ListByBillingPeriodComplete(context.Background(), billingPeriod, "", filter, "", "", &top)
	return &result, err
}

func (c RestClient) getSubscriptionIterator() (subscriptionIterator, error) {
	subClient := c.newSubscriptionsClient()
	result, err := subClient.ListComplete(context.Background())
	return &result, err
}
