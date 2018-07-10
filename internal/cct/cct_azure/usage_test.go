package cct_azure

import (
	"errors"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-05-31/consumption"
	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/golang/mock/gomock"
	"github.com/shopspring/decimal"
)

func TestSplitString(t *testing.T) {
	instanceID := "/subscriptions/abcdefgh-1234-1234-abcd-abcdefghijkl/resourceGroups/elastisys-container-registry/providers/Microsoft.ContainerRegistry/registries/elastisys"
	expected := "Microsoft.ContainerRegistry/registries"
	actual := getProvider(instanceID)

	if actual != expected {
		t.Errorf("Wanted: %s got: %s", expected, actual)
	}
}

func TestGetCloudCost(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPeriodsIter := NewMockperiodsIterator(mockCtrl)
	mockUsageIter := NewMockusageIterator(mockCtrl)
	mockClient := NewMockClient(mockCtrl)
	ue := NewUsageExplorer(mockClient)

	t.Run("Fail to get cloud cost", func(t *testing.T) {
		err0 := errors.New("error")
		periodIter := mockPeriodsIter
		mockClient.EXPECT().getPeriodIterator(gomock.Any()).Return(periodIter, err0)
		date := time.Date(2018, time.July, 3, 00, 0, 0, 0, time.UTC)

		_, err := ue.GetCloudCost(date)

		if err == nil {
			t.Errorf("Expected error but got none!")
		}
	})

	t.Run("Get cloud cost", func(t *testing.T) {
		// filter := "filter"
		id := "id"
		name := "name"
		instanceID := "/subscriptions/abcdefgh-1234-1234-abcd-abcdefghijkl/resourceGroups/elastisys-container-registry/providers/Microsoft.ContainerRegistry/registries/elastisys"
		pretaxCost := decimal.NewFromFloat(10.50)
		currency := "SEK"
		usageDate := time.Date(2018, time.July, 3, 00, 0, 0, 0, time.UTC)
		usageStart := date.Time{usageDate}
		usageEnd := date.Time{time.Date(2018, time.July, 4, 00, 0, 0, 0, time.UTC)}
		usageProps := consumption.UsageDetailProperties{InstanceID: &instanceID, PretaxCost: &pretaxCost, Currency: &currency, UsageStart: &usageStart, UsageEnd: &usageEnd}
		period := billing.Period{ID: &id, Name: &name}
		usage := consumption.UsageDetail{ID: &id, Name: &name, UsageDetailProperties: &usageProps}

		mockPeriodsIter.EXPECT().Value().Return(period)

		// We have a single item in the iterator so return true the first time, then false
		mockUsageIter.EXPECT().NotDone().Return(true)
		mockUsageIter.EXPECT().NotDone().Return(false)
		mockUsageIter.EXPECT().Value().Return(usage)
		mockUsageIter.EXPECT().Next()

		periodIter := mockPeriodsIter
		mockClient.EXPECT().getPeriodIterator(gomock.Any()).Return(periodIter, nil)
		mockClient.EXPECT().getUsageIterator(name, gomock.Any()).Return(mockUsageIter, nil)

		// labels := make(map[string]string)
		// data := db_client.UsageData{Cost: pretaxCost, Currency: currency, Date: usageDate, Labels: labels}
		// expected := []db_client.UsageData{data}
		actual, err := ue.GetCloudCost(usageDate)

		if err != nil {
			t.Errorf("Caught error: %s", err)
		}

		if actual == nil {
			t.Errorf("Expected and actual differ!")
		}
	})
}
