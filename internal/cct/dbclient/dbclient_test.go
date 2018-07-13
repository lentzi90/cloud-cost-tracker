package dbclient

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	client "github.com/influxdata/influxdb/client/v2"
)

var (
	dbConfig = Config{
		DBName:   "TestDB",
		Username: "testUser",
		Password: "testPassword",
		Address:  "TestAddress",
	}
	usageData1 = UsageData{
		Cost:     111,
		Currency: "SEK",
		Date:     time.Date(2001, time.January, 1, 1, 1, 1, 1, time.UTC),
		Labels: map[string]string{
			"l1": "test1",
			"l2": "test2",
		},
	}
	usageData2 = UsageData{
		Cost:     222,
		Currency: "USD",
		Date:     time.Date(2002, time.February, 2, 2, 2, 2, 2, time.UTC),
		Labels: map[string]string{
			"l3": "test3",
			"l4": "test4",
		},
	}
	usageData3 = UsageData{
		Cost:     333,
		Currency: "USD",
		Date:     time.Date(2003, time.March, 3, 3, 3, 3, 3, time.UTC),
		Labels:   nil,
	}
	usageDataArray = []UsageData{usageData1, usageData2, usageData3}
)

func TestConfig(t *testing.T) {
	dbClient := NewDBClient(dbConfig)

	expected := dbConfig
	actual := dbClient.GetConfig()

	if actual != expected {
		t.Errorf("Wanted: %s got: %s", expected, actual)
	}
}

// Tests that everything works if everythings goes well
func TestAddUsageData(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Create mocked points and BatchPoints
	testPoint1 := &client.Point{}
	testPoint2 := &client.Point{}
	testPoint3 := &client.Point{}
	mockedBP := NewMockbp(mockCtrl)
	mockedBP.EXPECT().AddPoint(testPoint1).Times(1)
	mockedBP.EXPECT().AddPoint(testPoint2).Times(1)
	mockedBP.EXPECT().AddPoint(testPoint3).Times(1)

	// Mocked Client that NewHTTPClient will return
	mockConnection := createWorkingConClient(mockCtrl, 2, 3)

	// Mocked influxInterface that will return the mocked Client
	mockinfluxInterface := createWorkinginfluxInterface(mockCtrl, mockConnection)

	// Mocked NewBatchPoints that will return the mocked BatchPoint
	mockinfluxInterface.EXPECT().NewBatchPoints(client.BatchPointsConfig{
		Database:  dbConfig.DBName,
		Precision: "h",
	}).
		Times(3).
		DoAndReturn(func(conf client.BatchPointsConfig) (client.BatchPoints, error) {
			return mockedBP, nil
		})

	expectedFields1 := map[string]interface{}{"cost": usageData1.Cost}
	expectedLabels1 := usageData1.Labels
	expectedLabels1["currency"] = usageData1.Currency

	expectedFields2 := map[string]interface{}{"cost": usageData2.Cost}
	expectedLabels2 := usageData2.Labels
	expectedLabels2["currency"] = usageData2.Currency

	expectedFields3 := map[string]interface{}{"cost": usageData3.Cost}
	expectedLabels3 := map[string]string{"currency": usageData3.Currency}

	// Mocked NewPoint that will return the mocked Point
	mockinfluxInterface.EXPECT().NewPoint("cost", expectedLabels1, expectedFields1, usageData1.Date).
		Times(1).
		DoAndReturn(func(name string, tags map[string]string, fields map[string]interface{}, t time.Time) (*client.Point, error) {
			return testPoint1, nil
		})

	mockinfluxInterface.EXPECT().NewPoint("cost", expectedLabels2, expectedFields2, usageData2.Date).
		Times(1).
		DoAndReturn(func(name string, tags map[string]string, fields map[string]interface{}, t time.Time) (*client.Point, error) {
			return testPoint2, nil
		})

	mockinfluxInterface.EXPECT().NewPoint("cost", expectedLabels3, expectedFields3, usageData3.Date).
		Times(1).
		DoAndReturn(func(name string, tags map[string]string, fields map[string]interface{}, t time.Time) (*client.Point, error) {
			return testPoint3, nil
		})

	dbClient := NewDBClient(dbConfig)
	dbClient.influxInterface = mockinfluxInterface

	actual := dbClient.AddUsageData(usageDataArray)

	if actual != nil {
		t.Errorf("Wanted: AddUsageData to return nil but got %v", actual)
	}
}

// Tests that AddUsageData fails if httpClient fails
func TestHttpClientFail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	// Mocked influxInterface that will return an error
	mockinfluxInterface := NewMockinfluxInterface(mockCtrl)
	mockinfluxInterface.EXPECT().NewHTTPClient(client.HTTPConfig{
		Addr:     dbConfig.Address,
		Username: dbConfig.Username,
		Password: dbConfig.Password,
	}).
		Times(1).
		DoAndReturn(func(conf client.HTTPConfig) (client.Client, error) {
			return nil, errors.New("testHTTPClientError")
		})

	dbClient := NewDBClient(dbConfig)
	dbClient.influxInterface = mockinfluxInterface

	actual := dbClient.AddUsageData(usageDataArray)
	expected := "testHTTPClientError"

	if actual.Error() != expected {
		t.Errorf("Wanted: AddUsageData to return %v but got %v", expected, actual)
	}
}

// Tests that AddUsageData fails if batchPoints fails
func TestBatchPointFail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Mocked Client that NewHTTPClient will return
	mockConnection := createWorkingConClient(mockCtrl, 1, 0)

	// Mocked influxInterface that will return the mocked Client
	mockinfluxInterface := createWorkinginfluxInterface(mockCtrl, mockConnection)

	// Mocked NewBatchPoints that will return an error
	mockinfluxInterface.EXPECT().NewBatchPoints(client.BatchPointsConfig{
		Database:  dbConfig.DBName,
		Precision: "h",
	}).
		Times(1).
		DoAndReturn(func(conf client.BatchPointsConfig) (client.BatchPoints, error) {
			return nil, errors.New("testBatchPointsError")
		})

	dbClient := NewDBClient(dbConfig)
	dbClient.influxInterface = mockinfluxInterface

	actual := dbClient.AddUsageData(usageDataArray)
	expected := "testBatchPointsError"

	if actual.Error() != expected {
		t.Errorf("Wanted: AddUsageData to return %v but got %v", expected, actual)
	}
}

// Tests that AddUsageData fails if point fails
func TestPointFail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Create mocked points and BatchPoints
	mockedBP := NewMockbp(mockCtrl)
	mockedBP.EXPECT().AddPoint(gomock.Any()).Times(0)

	// Mocked Client that NewHTTPClient will return
	mockConnection := createWorkingConClient(mockCtrl, 1, 0)

	// Mocked influxInterface that will return the mocked Client
	mockinfluxInterface := createWorkinginfluxInterface(mockCtrl, mockConnection)

	mockinfluxInterface.EXPECT().NewBatchPoints(client.BatchPointsConfig{
		Database:  dbConfig.DBName,
		Precision: "h",
	}).
		Times(1).
		DoAndReturn(func(conf client.BatchPointsConfig) (client.BatchPoints, error) {
			return mockedBP, nil
		})

	mockinfluxInterface.EXPECT().NewPoint("cost", gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(name string, tags map[string]string, fields map[string]interface{}, t time.Time) (*client.Point, error) {
			return nil, errors.New("testPointError")
		})

	dbClient := NewDBClient(dbConfig)
	dbClient.influxInterface = mockinfluxInterface

	actual := dbClient.AddUsageData(usageDataArray)
	expected := "testPointError"

	if actual.Error() != expected {
		t.Errorf("Wanted: AddUsageData to return %v but got %v", expected, actual)
	}
}

// Tests that AddUsageData fails if Write fails
func TestWriteFail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Create mocked points and BatchPoints
	testPoint := &client.Point{}
	mockedBP := NewMockbp(mockCtrl)
	mockedBP.EXPECT().AddPoint(gomock.Any()).Times(1)

	// Mocked Client that NewHTTPClient will return
	mockConnection := NewMockconClient(mockCtrl)
	mockConnection.EXPECT().Close().Times(1)
	mockConnection.EXPECT().Write(mockedBP).Times(1).DoAndReturn(func(client.BatchPoints) error {
		return errors.New("testWriteError")
	})

	// Mocked influxInterface that will return the mocked Client
	mockinfluxInterface := createWorkinginfluxInterface(mockCtrl, mockConnection)

	mockinfluxInterface.EXPECT().NewBatchPoints(client.BatchPointsConfig{
		Database:  dbConfig.DBName,
		Precision: "h",
	}).
		Times(1).
		DoAndReturn(func(conf client.BatchPointsConfig) (client.BatchPoints, error) {
			return mockedBP, nil
		})

	// Mocked NewPoint that will return the mocked Point
	mockinfluxInterface.EXPECT().NewPoint("cost", gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(name string, tags map[string]string, fields map[string]interface{}, t time.Time) (*client.Point, error) {
			return testPoint, nil
		})

	dbClient := NewDBClient(dbConfig)
	dbClient.influxInterface = mockinfluxInterface

	actual := dbClient.AddUsageData(usageDataArray)
	expected := "testWriteError"

	if actual.Error() != expected {
		t.Errorf("Wanted: AddUsageData to return %v but got %v", expected, actual)
	}
}

// Creates a working influxInterface mock for testing
func createWorkinginfluxInterface(mockCtrl *gomock.Controller, mockConnection conClient) *MockinfluxInterface {
	mockinfluxInterface := NewMockinfluxInterface(mockCtrl)
	mockinfluxInterface.EXPECT().NewHTTPClient(client.HTTPConfig{
		Addr:     dbConfig.Address,
		Username: dbConfig.Username,
		Password: dbConfig.Password,
	}).
		Times(1).
		DoAndReturn(func(conf client.HTTPConfig) (client.Client, error) {
			return mockConnection, nil
		})
	return mockinfluxInterface
}

// Create a working conClient mock for testing
func createWorkingConClient(mockCtrl *gomock.Controller, timesClose int, timesWrite int) conClient {
	mockConnection := NewMockconClient(mockCtrl)
	mockConnection.EXPECT().Close().Times(timesClose)
	mockConnection.EXPECT().Write(gomock.Any()).Times(timesWrite)
	return mockConnection
}
