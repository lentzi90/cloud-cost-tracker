package azure

import (
	"errors"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-05-31/consumption"
	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"
	"github.com/Azure/azure-sdk-for-go/services/preview/subscription/mgmt/2018-03-01-preview/subscription"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/golang/mock/gomock"
	"github.com/lentzi90/cloud-cost-tracker/internal/cct/dbclient"
	"github.com/shopspring/decimal"
)

func init() {
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
	// log.SetOutput(os.Stdout)
}

type inputData struct {
	subscription  subscription.Model
	billingPeriod billing.Period
	usageDetails  []consumption.UsageDetail
}

// Base data for building more complex data types
var (
	subscriptionID  = "abcdefgh-1234-1234-abcd-abcdefghijkl"
	subscriptionID2 = "hgfedcba-4321-4321-dcba-lkjihgfedcba"
	periodID        = "id"
	periodName      = "period-name"
	period          = billing.Period{ID: &periodID, Name: &periodName}
	resourceGroup   = "group-name"
	provider        = "Microsoft.ContainerRegistry/registries"
	provider2       = "Microsoft.Compute/disks"
	instance        = "instance1"
	instance2       = "instance2"
	cost            = 10.50
	currency        = "SEK"
	currency2       = "USD"
	usageDate       = time.Date(2018, time.July, 3, 00, 0, 0, 0, time.UTC)
	instanceID      = "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/" + provider + "/" + instance
	instanceID2     = "/subscriptions/" + subscriptionID2 + "/resourceGroups/" + resourceGroup + "/providers/" + provider2 + "/" + instance2
	instanceID3     = "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/" + provider2 + "/" + instance2
	labels          = map[string]string{"cloud": "azure", "subscription": subscriptionID, "resource_group": resourceGroup, "service": provider, "instance": instance, "currency": currency}
	labels2         = map[string]string{"cloud": "azure", "subscription": subscriptionID2, "resource_group": resourceGroup, "service": provider2, "instance": instance2, "currency": currency2}
	labels3         = map[string]string{"cloud": "azure", "subscription": subscriptionID, "resource_group": resourceGroup, "service": provider2, "instance": instance2, "currency": currency}
	usageData       = dbclient.UsageData{Cost: cost, Date: usageDate, Labels: labels}
	usageData2      = dbclient.UsageData{Cost: cost, Date: usageDate, Labels: labels2}
	usageData3      = dbclient.UsageData{Cost: cost, Date: usageDate, Labels: labels3}
)

// Output usageData
var (
	// Simplest case: a single subscription with a single instance
	cloudCost = []dbclient.UsageData{usageData}
	// Two subscriptions with one instance each
	cloudCost2 = []dbclient.UsageData{usageData, usageData2}
	// A single subscription with two instances from two different providers
	cloudCost3 = []dbclient.UsageData{usageData, usageData3}
	emptyCost  = []dbclient.UsageData{}
)

// Faked data from API
var (
	subscription1 = subscription.Model{SubscriptionID: &subscriptionID}
	subscription2 = subscription.Model{SubscriptionID: &subscriptionID2}
	usageDetail   = fakeUsageDetail(usageDate, cost, currency, instanceID)
	usageDetail2  = fakeUsageDetail(usageDate, cost, currency2, instanceID2)
	usageDetail3  = fakeUsageDetail(usageDate, cost, currency, instanceID3)
	// Properties are missing
	partialUsageDetail = consumption.UsageDetail{ID: &periodID, Name: &periodName, UsageDetailProperties: nil}
	usageSlice         = []consumption.UsageDetail{usageDetail}
	usageSlice2        = []consumption.UsageDetail{usageDetail, usageDetail2}
	usageSlice3        = []consumption.UsageDetail{usageDetail, usageDetail3}
	partialUsageSlice  = []consumption.UsageDetail{partialUsageDetail}
)

// Input data for setting up mocks
var (
	// These inputs correspond to the cloud costs above
	input        = []inputData{{subscription: subscription1, billingPeriod: period, usageDetails: usageSlice}}
	input2       = []inputData{{subscription: subscription2, billingPeriod: period, usageDetails: usageSlice2}}
	input3       = []inputData{{subscription: subscription1, billingPeriod: period, usageDetails: usageSlice3}}
	partialInput = []inputData{{subscription: subscription1, billingPeriod: period, usageDetails: partialUsageSlice}}
)

func TestGetCloudCost(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSubscriptionsIter := NewMocksubscriptionIterator(mockCtrl)
	mockPeriodsIter := NewMockperiodsIterator(mockCtrl)
	mockUsageIter := NewMockusageIterator(mockCtrl)
	mockClient := NewMockClient(mockCtrl)
	ue := UsageExplorer{client: mockClient}

	cases := []struct {
		in   []inputData
		want []dbclient.UsageData
	}{
		{input, cloudCost},        // Single subscription and instance
		{input2, cloudCost2},      // Two subscriptions and instances
		{input3, cloudCost3},      // Two instances and one subscription
		{partialInput, emptyCost}, // Missing properties: not abel to calculate cost
	}

	for _, c := range cases {
		setupClient(*mockClient, mockSubscriptionsIter, mockPeriodsIter, mockUsageIter, c.in)
		setupIterators(*mockSubscriptionsIter, *mockPeriodsIter, *mockUsageIter, c.in)

		actual, err := ue.GetCloudCost(usageDate)
		if err != nil {
			t.Errorf("Caught error: %s", err)
		}

		checkCloudCost(t, c.want, actual)
	}

	// Some special cases
	// ------------------

	t.Run("Fail to get subscriptions iterator", func(t *testing.T) {
		mockClient.EXPECT().getSubscriptionIterator().Return(mockSubscriptionsIter, errors.New("error"))

		_, err := ue.GetCloudCost(usageDate)

		if err == nil {
			t.Errorf("Expected error but got none!")
		}
	})

	t.Run("Fail to get period iterator", func(t *testing.T) {
		mockClient.EXPECT().getSubscriptionIterator().Return(mockSubscriptionsIter, nil)
		mockSubscriptionsIter.EXPECT().Next().AnyTimes()
		mockSubscriptionsIter.EXPECT().NotDone().Return(true)
		mockSubscriptionsIter.EXPECT().Value().Return(subscription1)
		mockSubscriptionsIter.EXPECT().NotDone().Return(false)
		mockClient.EXPECT().getPeriodIterator(subscriptionID, gomock.Any()).Return(mockPeriodsIter, errors.New("error"))

		_, err := ue.GetCloudCost(usageDate)

		if err == nil {
			t.Errorf("Expected error but got none!")
		}
	})

	t.Run("Fail to get usage iterator", func(t *testing.T) {
		mockPeriodsIter.EXPECT().Value().Return(period)
		mockClient.EXPECT().getSubscriptionIterator().Return(mockSubscriptionsIter, nil)
		mockSubscriptionsIter.EXPECT().Next().AnyTimes()
		mockSubscriptionsIter.EXPECT().NotDone().Return(true)
		mockSubscriptionsIter.EXPECT().Value().Return(subscription1)
		mockSubscriptionsIter.EXPECT().NotDone().Return(false)
		mockClient.EXPECT().getPeriodIterator(subscriptionID, gomock.Any()).Return(mockPeriodsIter, nil)
		mockClient.EXPECT().getUsageIterator(subscriptionID, periodName, gomock.Any()).Return(mockUsageIter, errors.New("error"))

		_, err := ue.GetCloudCost(usageDate)

		if err == nil {
			t.Errorf("Expected error but got none!")
		}
	})
}

// Helper functions
// ----------------

// Create a UsageDetail object from the provided input
func fakeUsageDetail(usageDate time.Time, cost float64, currency string, instanceID string) consumption.UsageDetail {
	id := "id"
	name := "name"
	pretaxCost := decimal.NewFromFloat(cost)
	usageStart := date.Time{Time: usageDate}
	// End after one day
	usageEnd := date.Time{Time: usageDate.AddDate(0, 0, 1)}
	usageProps := consumption.UsageDetailProperties{InstanceID: &instanceID, PretaxCost: &pretaxCost, Currency: &currency, UsageStart: &usageStart, UsageEnd: &usageEnd}
	return consumption.UsageDetail{ID: &id, Name: &name, UsageDetailProperties: &usageProps}
}

// Make the iterators iterate over the provided data
func setupIterators(subsIter MocksubscriptionIterator, periodsIter MockperiodsIterator, usageIter MockusageIterator, input []inputData) {
	subsIter.EXPECT().Next().AnyTimes()
	usageIter.EXPECT().Next().AnyTimes()
	periodsIter.EXPECT().Next().AnyTimes()
	for _, data := range input {
		subsIter.EXPECT().NotDone().Return(true)
		subsIter.EXPECT().Value().Return(data.subscription)
		periodsIter.EXPECT().Value().Return(data.billingPeriod)
		for _, usage := range data.usageDetails {
			usageIter.EXPECT().NotDone().Return(true)
			usageIter.EXPECT().Value().Return(usage)
		}
		usageIter.EXPECT().NotDone().Return(false)
	}
	subsIter.EXPECT().NotDone().Return(false)
}

// Make the mocked client return desired iterators
func setupClient(mock MockClient, subscriptionsIter subscriptionIterator, periodsIter periodsIterator, usageIter usageIterator, input []inputData) {
	mock.EXPECT().getSubscriptionIterator().Return(subscriptionsIter, nil)
	for _, data := range input {
		mock.EXPECT().getPeriodIterator(*data.subscription.SubscriptionID, gomock.Any()).Return(periodsIter, nil)
		mock.EXPECT().getUsageIterator(*data.subscription.SubscriptionID, gomock.Any(), gomock.Any()).Return(usageIter, nil)
	}
}

// Check that the actual data resambles the expected data
func checkCloudCost(t *testing.T, expected, actual []dbclient.UsageData) {
	if len(actual) != len(expected) {
		t.Log("UsageData slice lengths differ!")
		t.Log("Expected", len(expected), "but got", len(actual))
		t.FailNow()
	}
	for i := range expected {
		if (actual[i].Cost != expected[i].Cost) ||
			(actual[i].Date != expected[i].Date) {
			t.Errorf("UsageData differ. Expected: %f %s, Actual: %f %s",
				expected[i].Cost, expected[i].Date.Format("2006-01-02"),
				actual[i].Cost, actual[i].Date.Format("2006-01-02"))
		}

		if len(expected[i].Labels) != len(actual[i].Labels) {
			t.Errorf("Expected %d labels, actual %d", len(expected[i].Labels), len(actual[i].Labels))
		}

		for k, v := range expected[i].Labels {
			if v != actual[i].Labels[k] {
				t.Errorf("Expected: %s=%s, actual: %s=%s", k, v, k, actual[i].Labels[k])
			}
		}
	}
}
