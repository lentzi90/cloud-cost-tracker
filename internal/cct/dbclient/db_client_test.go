package dbclient

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	client "github.com/influxdata/influxdb/client/v2"
)

var (
	dbConfig = DBClientConfig{
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
	usageDataArray = []UsageData{usageData1, usageData2}
)

func TestConfig(t *testing.T) {
	dbClient := NewDBClient(dbConfig)

	expected := dbConfig
	actual := dbClient.GetConfig()

	if actual != expected {
		t.Errorf("Wanted: %s got: %s", expected, actual)
	}
}

func TestAddUsageData(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Create mocked points and BatchPoints
	testPoint1 := &client.Point{}
	testPoint2 := &client.Point{}
	mockedBP := NewMockbp(mockCtrl)
	mockedBP.EXPECT().AddPoint(testPoint1).Times(1)
	mockedBP.EXPECT().AddPoint(testPoint2).Times(1)

	// Mocked Client that NewHTTPClient will return
	mockConnection := createWorkingConnection(mockCtrl, 2, 2)

	// Mocked HTTPClient that will return the mocked Client
	mockHTTPClient := createWorkingHTTPClient(mockCtrl, mockConnection)

	// Mocked NewBatchPoints that will return the mocked BatchPoint
	mockBatchPoints := createWorkingBatchPoints(mockCtrl, mockedBP, 2)

	expectedFields1 := map[string]interface{}{"cost": usageData1.Cost}
	expectedLabels1 := usageData1.Labels
	expectedLabels1["currency"] = usageData1.Currency

	expectedFields2 := map[string]interface{}{"cost": usageData2.Cost}
	expectedLabels2 := usageData2.Labels
	expectedLabels2["currency"] = usageData2.Currency

	// Mocked NewPoint that will return the mocked Point
	mockPoint := NewMockpoint(mockCtrl)
	mockPoint.EXPECT().NewPoint("cost", expectedLabels1, expectedFields1, usageData1.Date).
		Times(1).
		DoAndReturn(func(name string, tags map[string]string, fields map[string]interface{}, t time.Time) (*client.Point, error) {
			return testPoint1, nil
		})

	mockPoint.EXPECT().NewPoint("cost", expectedLabels2, expectedFields2, usageData2.Date).
		Times(1).
		DoAndReturn(func(name string, tags map[string]string, fields map[string]interface{}, t time.Time) (*client.Point, error) {
			return testPoint2, nil
		})

	dbClient := NewDBClient(dbConfig)
	dbClient.httpClient = mockHTTPClient
	dbClient.batchPoints = mockBatchPoints
	dbClient.point = mockPoint

	if !dbClient.AddUsageData(usageDataArray) {
		t.Fail()
	}
}

func TestAddUsageDataHTTPClientFail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	// Mocked HTTPClient that will return an error
	mockHTTPClient := NewMockhttpClient(mockCtrl)
	mockHTTPClient.EXPECT().NewHTTPClient(client.HTTPConfig{
		Addr:     dbConfig.Address,
		Username: dbConfig.Username,
		Password: dbConfig.Password,
	}).
		Times(1).
		DoAndReturn(func(conf client.HTTPConfig) (client.Client, error) {
			return nil, errors.New("testError")
		})

	dbClient := NewDBClient(dbConfig)
	dbClient.httpClient = mockHTTPClient

	if dbClient.AddUsageData(usageDataArray) {
		t.Fail()
	}
}

func TestBatchPointFail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Mocked Client that NewHTTPClient will return
	mockConnection := createWorkingConnection(mockCtrl, 1, 0)

	// Mocked HTTPClient that will return the mocked Client
	mockHTTPClient := createWorkingHTTPClient(mockCtrl, mockConnection)

	// Mocked NewBatchPoints that will return an error
	mockBatchPoints := NewMockbatchPoints(mockCtrl)
	mockBatchPoints.EXPECT().NewBatchPoints(client.BatchPointsConfig{
		Database:  dbConfig.DBName,
		Precision: "h",
	}).
		Times(1).
		DoAndReturn(func(conf client.BatchPointsConfig) (client.BatchPoints, error) {
			return nil, errors.New("testError")
		})

	dbClient := NewDBClient(dbConfig)
	dbClient.httpClient = mockHTTPClient
	dbClient.batchPoints = mockBatchPoints

	if dbClient.AddUsageData(usageDataArray) {
		t.Fail()
	}
}

func TestPointFail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Create mocked points and BatchPoints
	mockedBP := NewMockbp(mockCtrl)
	mockedBP.EXPECT().AddPoint(gomock.Any()).Times(0)

	// Mocked Client that NewHTTPClient will return
	mockConnection := createWorkingConnection(mockCtrl, 1, 0)

	// Mocked HTTPClient that will return the mocked Client
	mockHTTPClient := createWorkingHTTPClient(mockCtrl, mockConnection)

	// Mocked NewBatchPoints that will return the mocked BatchPoint
	mockBatchPoints := createWorkingBatchPoints(mockCtrl, mockedBP, 1)

	// Mocked NewPoint that will return the mocked Point
	mockPoint := NewMockpoint(mockCtrl)
	mockPoint.EXPECT().NewPoint("cost", gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(name string, tags map[string]string, fields map[string]interface{}, t time.Time) (*client.Point, error) {
			return nil, errors.New("testError")
		})

	dbClient := NewDBClient(dbConfig)
	dbClient.httpClient = mockHTTPClient
	dbClient.batchPoints = mockBatchPoints
	dbClient.point = mockPoint

	if dbClient.AddUsageData(usageDataArray) {
		t.Fail()
	}
}

// Creates a working HTTPClient for testing
func createWorkingHTTPClient(mockCtrl *gomock.Controller, mockConnection conClient) httpClient {
	mockHTTPClient := NewMockhttpClient(mockCtrl)
	mockHTTPClient.EXPECT().NewHTTPClient(client.HTTPConfig{
		Addr:     dbConfig.Address,
		Username: dbConfig.Username,
		Password: dbConfig.Password,
	}).
		Times(1).
		DoAndReturn(func(conf client.HTTPConfig) (client.Client, error) {
			return mockConnection, nil
		})
	return mockHTTPClient
}

// Creates a working BatchPoint for testing
func createWorkingBatchPoints(mockCtrl *gomock.Controller, mockedBP bp, times int) batchPoints {
	mockBatchPoints := NewMockbatchPoints(mockCtrl)
	mockBatchPoints.EXPECT().NewBatchPoints(client.BatchPointsConfig{
		Database:  dbConfig.DBName,
		Precision: "h",
	}).
		Times(times).
		DoAndReturn(func(conf client.BatchPointsConfig) (client.BatchPoints, error) {
			return mockedBP, nil
		})
	return mockBatchPoints
}

func createWorkingConnection(mockCtrl *gomock.Controller, timesClose int, timesWrite int) conClient {
	mockConnection := NewMockconClient(mockCtrl)
	mockConnection.EXPECT().Close().Times(timesClose)
	mockConnection.EXPECT().Write(gomock.Any()).Times(timesWrite)
	return mockConnection
}
