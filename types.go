package openforecast

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type DataPoint interface {
	SetDependentValue(v float64)
	DependentValue() float64
	SetIndependentValue(k string, v float64)
	IndependentValue(k string) (v float64, ok bool)
	IndependentVariableNames() []string
	Equals(DataPoint) bool
	fmt.Stringer
}

type DataSet struct {
	Points         []DataPoint
	timeVariable   string
	periodsPerYear int
}

func (ds *DataSet) TimeVariable() string { return ds.timeVariable }
func (ds *DataSet) PeriodsPerYear() int  { return ds.periodsPerYear }

func NewDataSetCopy(dataSet *DataSet) *DataSet {
	return NewDataSet(dataSet.TimeVariable(), dataSet.PeriodsPerYear(), dataSet.Points)
}

func NewDataSet(timeVariable string, periodsPerYear int, points []DataPoint) *DataSet {
	ds := &DataSet{
		timeVariable:   timeVariable,
		periodsPerYear: periodsPerYear,
		Points:         make([]DataPoint, 0, len(points)),
	}
	// deep copy DataPoint which is reference
	for _, dp := range points {
		ds.Points = append(ds.Points, NewObservationCopy(dp))
	}
	return ds
}

func (ds *DataSet) IndependentVariables() []string {

	return nil
}

type ForecastingModel interface {
	Type() string

	Train(*DataSet) error
	Forecast(DataPoint) (float64, error)
	ForecastAll(*DataSet) (*DataSet, error)

	AIC() float64
	Bias() float64
	MAD() float64
	MAPE() float64
	MSE() float64
	SAE() float64

	NumberOfPredictors() int
}

type StreamingModel interface {
	// TODO(StitchCula): DataSet or DataPoint?
	Update(DataPoint) error
}

type Observation struct {
	dependentValue    float64
	independentValues map[string]float64
	mu                *sync.RWMutex
}

func NewObservation(dependentValue float64) *Observation {
	o := &Observation{
		dependentValue:    dependentValue,
		independentValues: make(map[string]float64),
		mu:                new(sync.RWMutex),
	}
	return o
}

func NewObservationCopy(dataPoint DataPoint) *Observation {
	o := NewObservation(dataPoint.DependentValue())
	for _, k := range dataPoint.IndependentVariableNames() {
		o.independentValues[k], _ = dataPoint.IndependentValue(k)
	}
	return o
}

func (o *Observation) SetDependentValue(v float64) {
	o.dependentValue = v
}

func (o *Observation) DependentValue() float64 {
	return o.dependentValue
}

func (o *Observation) SetIndependentValue(k string, v float64) {
	o.mu.Lock()
	o.independentValues[k] = v
	o.mu.Unlock()
}

func (o *Observation) IndependentValue(k string) (v float64, ok bool) {
	o.mu.RLock()
	v, ok = o.independentValues[k]
	o.mu.RUnlock()
	return
}

func (o *Observation) IndependentVariableNames() []string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	ks := make([]string, 0, len(o.independentValues))
	for k := range o.independentValues {
		ks = append(ks, k)
	}
	return ks
}

// Equals goroutine-unsafe
func (o *Observation) Equals(dataPoint DataPoint) bool {
	if o.DependentValue() != dataPoint.DependentValue() {
		return false
	}
	ks1 := o.IndependentVariableNames()
	ks2 := dataPoint.IndependentVariableNames()
	if len(ks1) != len(ks2) {
		return false
	}
	for _, k := range ks1 {
		v1, _ := o.IndependentValue(k)
		if v2, ok := dataPoint.IndependentValue(k); !ok || v1 != v2 {
			return false
		}
	}
	return true
}

func (o *Observation) String() string {
	buf := strings.Builder{}
	buf.WriteByte('(')

	o.mu.RLock()
	for k, v := range o.independentValues {
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
		buf.WriteString(", ")
	}
	o.mu.RUnlock()

	buf.WriteString("dependentValue=")
	buf.WriteString(strconv.FormatFloat(o.dependentValue, 'f', -1, 64))
	buf.WriteByte(')')

	return buf.String()
}
