package main

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/dustin/go-couchstore"
)

var testInput = []*string{nil}
var nextValue = "29"

var bigInput []byte

func init() {
	s := []string{"31", "63", "foo", "17"}
	for i := range s {
		testInput = append(testInput, &s[i])
	}

	var err error
	bigInput, err = ioutil.ReadFile("sample.json")
	if err != nil {
		panic("Couldn't read sample.json")
	}
}

func streamCollection(s []*string) chan ptrval {
	ch := make(chan ptrval)
	go func() {
		defer close(ch)
		t := time.Unix(1347255646, 418514126).UTC()
		for _, r := range s {
			t = t.Add(time.Second)
			ts := t.Format(time.RFC3339Nano)
			ch <- ptrval{couchstore.NewDocInfo(ts, 0), r, true}
		}
		t = t.Add(time.Second)
		ts := t.Format(time.RFC3339Nano)
		ch <- ptrval{couchstore.NewDocInfo(ts, 0), &nextValue, false}
	}()
	return ch
}

func TestEmptyRateConversion(t *testing.T) {
	ch := make(chan ptrval)
	rch := convertTofloat64Rate(ch)
	close(ch)
	val, got := <-rch
	if got {
		t.Fatalf("Expected empty channel, got %v", val)
	}
}

func TestSingleRateConversion(t *testing.T) {
	ch := make(chan ptrval, 1)
	rch := convertTofloat64Rate(ch)
	ch <- ptrval{nil, &nextValue, true}
	close(ch)
	val, got := <-rch
	if got {
		t.Fatalf("Expected empty channel, got %v", val)
	}
}

func TestPairRateConversion(t *testing.T) {
	ch := make(chan ptrval, 2)
	rch := convertTofloat64Rate(ch)

	tm := time.Now().UTC()
	val1 := "20"
	ch <- ptrval{couchstore.NewDocInfo(tm.Format(time.RFC3339Nano), 0),
		&val1, true}

	tm = tm.Add(5 * time.Second)
	val2 := "25"
	ch <- ptrval{couchstore.NewDocInfo(tm.Format(time.RFC3339Nano), 0),
		&val2, false}

	close(ch)
	exp := 1.0
	val, got := <-rch
	if !got {
		t.Fatalf("Expected value, got empty channel")
	}
	if val != exp {
		t.Fatalf("Expected %v, got %v", exp, val)
	}
}

func TestReducers(t *testing.T) {
	tests := []struct {
		reducer string
		exp     interface{}
	}{
		{"any", "31"},
		{"count", 4},
		{"sum", float64(111)},
		{"sumsq", float64(5219)},
		{"max", float64(63)},
		{"min", float64(17)},
		{"avg", float64(37)},
		{"c_min", float64(-23)},
		{"c_avg", float64(7)},
		{"c_max", float64(32)},
		{"identity", testInput},
	}

	for _, test := range tests {
		got := reducers[test.reducer](streamCollection(testInput))
		if !reflect.DeepEqual(got, test.exp) {
			t.Errorf("Expected %v for %v, got %v",
				test.exp, test.reducer, got)
			t.Fail()
		}
	}
}

func TestEmptyReducers(t *testing.T) {
	emptyInput := []*string{}
	tests := []struct {
		reducer string
		exp     interface{}
	}{
		{"any", nil},
		{"count", 0},
		{"sum", 0.0},
		{"sumsq", 0.0},
		{"max", math.NaN()},
		{"min", math.NaN()},
		{"avg", math.NaN()},
		{"c_min", math.NaN()},
		{"c_avg", math.NaN()},
		{"c_max", math.NaN()},
		{"identity", emptyInput},
	}

	eq := func(a, b interface{}) bool {
		if !reflect.DeepEqual(a, b) {
			af, aok := a.(float64)
			bf, bok := b.(float64)
			return aok && bok && (math.IsNaN(af) == math.IsNaN(bf))
		}
		return true
	}

	for _, test := range tests {
		got := reducers[test.reducer](streamCollection(emptyInput))
		if !eq(got, test.exp) {
			t.Errorf("Expected %v for %v, got %v",
				test.exp, test.reducer, got)
			t.Fail()
		}
	}
}

func TestNilReducers(t *testing.T) {
	emptyInput := []*string{nil}
	tests := []struct {
		reducer string
		exp     interface{}
	}{
		{"any", nil},
		{"count", 0},
		{"sum", 0.0},
		{"sumsq", 0.0},
		{"max", math.NaN()},
		{"min", math.NaN()},
		{"avg", math.NaN()},
		{"c_min", math.NaN()},
		{"c_avg", math.NaN()},
		{"c_max", math.NaN()},
		{"identity", emptyInput},
	}

	eq := func(a, b interface{}) bool {
		if !reflect.DeepEqual(a, b) {
			af, aok := a.(float64)
			bf, bok := b.(float64)
			return aok && bok && (math.IsNaN(af) == math.IsNaN(bf))
		}
		return true
	}

	for _, test := range tests {
		got := reducers[test.reducer](streamCollection(emptyInput))
		if !eq(got, test.exp) {
			t.Errorf("Expected %v for %v, got %v",
				test.exp, test.reducer, got)
			t.Fail()
		}
	}
}

func BenchmarkJSONParser(b *testing.B) {
	b.SetBytes(int64(len(bigInput)))
	for i := 0; i < b.N; i++ {
		m := map[string]interface{}{}
		err := json.Unmarshal(bigInput, &m)
		if err != nil {
			b.Fatalf("Error unmarshaling json: %v", err)
		}
	}
}
