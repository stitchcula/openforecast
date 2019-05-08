package openforecast

import "sync"

type DataPoint interface {
	SetDependentValue(v float64)
	DependentValue()
	SetIndependentValue(k string, v float64)
	IndependentValue(k string)
	IndependentVariableNames() []string
	Equals(DataPoint) bool
}

type Observation struct {
	dependentValue    float64
	independentValues map[string]float64
	mu                *sync.RWMutex
}

type DataSet []DataPoint

type ForecastingModel interface {
	Type() string

	Train(DataSet) error
	Forecast(DataSet) (DataSet, error)

	AIC() float64
	Bias() float64
	MAD() float64
	MAPE() float64
	MSE() float64
	SAE() float64

	NumberOfPredictors() int
}

type StreamingModel interface {
	Update(DataSet)
}
