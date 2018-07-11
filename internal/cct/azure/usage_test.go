package azure

import (
	"errors"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-05-31/consumption"
	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/golang/mock/gomock"
	"github.com/lentzi90/cloud-cost-tracker/internal/cct/dbclient"
	"github.com/shopspring/decimal"
)

func init() {
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
}

func TestSplitString(t *testing.T) {
	expected := "Microsoft.ContainerRegistry/registries"
	instanceID := "/subscriptions/abcdefgh-1234-1234-abcd-abcdefghijkl/resourceGroups/elastisys-container-registry/providers/" + expected + "/elastisys"
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
		mockClient.EXPECT().getPeriodIterator(gomock.Any()).Return(mockPeriodsIter, err0)
		date := time.Date(2018, time.July, 3, 00, 0, 0, 0, time.UTC)

		_, err := ue.GetCloudCost(date)

		if err == nil {
			t.Errorf("Expected error but got none!")
		}
	})

	t.Run("Get cloud cost", func(t *testing.T) {
		// filter := "filter"
		provider := "Microsoft.ContainerRegistry/registries"
		id := "id"
		name := "name"
		instanceID := "/subscriptions/abcdefgh-1234-1234-abcd-abcdefghijkl/resourceGroups/elastisys-container-registry/providers/" + provider + "/elastisys"
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

		labels := map[string]string{"provider": provider}
		cost, _ := pretaxCost.Float64()
		data := dbclient.UsageData{Cost: cost, Currency: currency, Date: usageDate, Labels: labels}
		expected := []dbclient.UsageData{data}
		actual, err := ue.GetCloudCost(usageDate)

		if err != nil {
			t.Errorf("Caught error: %s", err)
		}

		checkCloudCost(t, expected, actual)
	})
}

func checkCloudCost(t *testing.T, expected, actual []dbclient.UsageData) {
	if len(actual) != len(expected) {
		t.Errorf("UsageData slince lengths differ!")
	}
	for i := range expected {
		if (actual[i].Cost != expected[i].Cost) ||
			(actual[i].Currency != expected[i].Currency) ||
			(actual[i].Date != expected[i].Date) {
			t.Errorf("UsageData differ. Expected: %f %s %s, Actual: %f %s %s",
				expected[i].Cost, expected[i].Currency, expected[i].Date.Format("2006-01-02"),
				actual[i].Cost, actual[i].Currency, actual[i].Date.Format("2006-01-02"))
		}

		if len(expected[i].Labels) != len(actual[i].Labels) {
			t.Errorf("Expected %d labels, actual %d", len(expected[i].Labels), len(actual[i].Labels))
		}

		for k, v := range expected[i].Labels {
			if v != actual[i].Labels[k] {
				t.Errorf("Expected label: %s=%s, actual: %s=%s", k, v, k, actual[i].Labels[k])
			}
		}
	}
}
