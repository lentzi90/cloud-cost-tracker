package cct_azure

import (
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-05-31/consumption"
	"github.com/Azure/azure-sdk-for-go/services/preview/billing/mgmt/2018-03-01-preview/billing"
	"github.com/golang/mock/gomock"
	"github.com/lentzi90/cct-azure/internal/cct/cct_azure/mocks"
)

func TestGetPeriodIterator(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBilling := mocks.NewMockbillingClient(mockCtrl)
	mockConsumption := mocks.NewMockconsumptionClient(mockCtrl)

	t.Run("Error from ListComplete", func(t *testing.T) {
		err0 := errors.New("error")
		expected := billing.PeriodsListResultIterator{}
		mockBilling.EXPECT().ListComplete(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(expected, err0)
		client := RestClient{mockBilling, mockConsumption}

		_, err := client.GetPeriodIterator("")

		if err == nil {
			t.Errorf("Expected error but got none!")
		}
	})

	t.Run("Get iterator with filter", func(t *testing.T) {
		filter := "filter"
		expected := billing.PeriodsListResultIterator{}
		mockBilling.EXPECT().ListComplete(gomock.Any(), filter, gomock.Any(), gomock.Any()).Return(expected, nil)
		client := RestClient{mockBilling, mockConsumption}

		actual, err := client.GetPeriodIterator(filter)
		if err != nil {
			t.Errorf("Caught error: %s", err)
		}
		if actual.Value() != expected.Value() {
			t.Errorf("Wanted and actual value differs!")
		}
	})
}

func TestGetUsageIterator(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBilling := mocks.NewMockbillingClient(mockCtrl)
	mockConsumption := mocks.NewMockconsumptionClient(mockCtrl)

	billingPeriod := "201809-1"
	filter := "filter"

	t.Run("Error from REST API", func(t *testing.T) {
		err0 := errors.New("error")
		expected := consumption.UsageDetailsListResultIterator{}
		mockConsumption.EXPECT().ListByBillingPeriodComplete(gomock.Any(), billingPeriod, gomock.Any(), filter, gomock.Any(), gomock.Any(), gomock.Any()).Return(expected, err0)
		client := RestClient{mockBilling, mockConsumption}

		_, err := client.GetUsageIterator(billingPeriod, filter)

		if err == nil {
			t.Errorf("Expected error but got none!")
		}
	})

	t.Run("Get usage with filter", func(t *testing.T) {
		expected := consumption.UsageDetailsListResultIterator{}
		mockConsumption.EXPECT().ListByBillingPeriodComplete(gomock.Any(), billingPeriod, gomock.Any(), filter, gomock.Any(), gomock.Any(), gomock.Any()).Return(expected, nil)
		client := RestClient{mockBilling, mockConsumption}

		actual, err := client.GetUsageIterator(billingPeriod, filter)

		if err != nil {
			t.Errorf("Caught error: %s", err)
		}

		if actual.Value().ID != expected.Value().ID {
			t.Errorf("Wanted and actual value differs!")
		}
	})
}
