package aws

import (
	"io/ioutil"
	"log"
	"math"
	"testing"
	"time"
)

func init() {
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
}

func TestS3ToUnix(t *testing.T) {
	form := "2006-01-02T15:04:05Z"
	timestamp := "2018-07-01T13:14:15Z"
	actual := s3ToUnix(timestamp, form)
	expected := 1530450855
	if actual != expected {
		t.Errorf("Expected value %d and actual value %d are not the same!", expected, actual)
	}
}

func TestCalculateRatio(t *testing.T) {
	start := time.Date(2018, time.June, 10, 0, 0, 0, 0, time.UTC)
	stop := time.Date(2018, time.June, 20, 0, 0, 0, 0, time.UTC)
	date := time.Date(2018, time.June, 12, 0, 0, 0, 0, time.UTC)
	expected := 0.1
	actual := calculateRatio(start, stop, date)
	if math.Abs((actual - expected)) > 1e-9 {
		t.Errorf("Expected value %f and actual value %f are not the same!", expected, actual)
	}
}

func TestCalculateRatioShort(t *testing.T) {
	start := time.Date(2018, time.June, 10, 12, 0, 0, 0, time.UTC)
	stop := time.Date(2018, time.June, 10, 13, 0, 0, 0, time.UTC)
	date := time.Date(2018, time.June, 10, 0, 0, 0, 0, time.UTC)
	expected := 1.0
	actual := calculateRatio(start, stop, date)
	if math.Abs((actual - expected)) > 1e-9 {
		t.Errorf("Expected value %f and actual value %f are not the same!", expected, actual)
	}
}

func TestCalculateRatioLong(t *testing.T) {
	start := time.Date(2018, time.July, 1, 0, 0, 0, 0, time.UTC)
	stop := time.Date(2018, time.September, 1, 0, 0, 0, 0, time.UTC)
	date := time.Date(2018, time.August, 10, 0, 0, 0, 0, time.UTC)
	expected := 1.0 / 62.0
	actual := calculateRatio(start, stop, date)
	if math.Abs((actual - expected)) > 1e-9 {
		t.Errorf("Expected value %f and actual value %f are not the same!", expected, actual)
	}
}

func TestOverlapSimple(t *testing.T) {
	expected := 1.0
	actual := overlap(0, 1, 0, 2)
	if actual != expected {
		t.Errorf("Expected value %f and actual value %f are not the same!", expected, actual)
	}
}

func TestOverlapNone(t *testing.T) {
	expected := 0.0
	actual := overlap(-10, 10, 10, 20)
	if actual != expected {
		t.Errorf("Expected value %f and actual value %f are not the same!", expected, actual)
	}
}

func TestOverlapContained(t *testing.T) {
	expected := 5.0
	actual := overlap(0, 100, 50, 55)
	if actual != expected {
		t.Errorf("Expected value %f and actual value %f are not the same!", expected, actual)
	}
}
