package models

import (
	"errors"
	"fmt"
	"github.com/stitchcula/openforecast"
	"math"
)

const (
	Tolerance = 1.0E-8
)

var (
	ErrUninitialized   = errors.New("NotInitialized")
	ErrUnimplemented   = errors.New("Unimplemented")
	ErrIllegalArgument = errors.New("IllegalArgument")
	ErrNotFound        = errors.New("NotFound")
)

type TimeBasedForecastingModel interface {
	openforecast.ForecastingModel
	ForecastTime(float64) (float64, error)
	NumberOfPeriods() int
}

type AbstractTimeBasedModel struct {
	*AbstractForecastingModel
	impl TimeBasedForecastingModel

	timeVariable   string
	timeDiff       float64
	observedValues *openforecast.DataSet
	forecastValues *openforecast.DataSet
	minTimeValue   float64
	maxTimeValue   float64
}

func (at *AbstractTimeBasedModel) TimeVariable() string  { return at.timeVariable }
func (at *AbstractTimeBasedModel) TimeInterval() float64 { return at.timeDiff }

func NewAbstractTimeBasedModel(impl TimeBasedForecastingModel) *AbstractTimeBasedModel {
	return &AbstractTimeBasedModel{
		AbstractForecastingModel: NewAbstractForecastingModel(impl),
		impl:                     impl,
	}
}

// TODO(StitchCula): 没看懂
func (at *AbstractTimeBasedModel) Train(dataSet *openforecast.DataSet) (err error) {
	if at.timeVariable, err = at.getTimeVariable(dataSet); err != nil {
		return
	}

	if len(dataSet.Points) < at.impl.NumberOfPeriods() {
		return ErrIllegalArgument
	}

	at.observedValues = openforecast.NewDataSetCopy(dataSet)
	at.observedValues.Sort(at.TimeVariable())
	lastValue, _ := at.observedValues.Points[0].IndependentValue(at.TimeVariable())
	currentValue, _ := at.observedValues.Points[1].IndependentValue(at.TimeVariable())
	at.forecastValues = openforecast.NewDataSet(at.TimeVariable(), 0, nil)
	at.timeDiff = currentValue - lastValue
	at.minTimeValue = lastValue

	for _, dp := range at.observedValues.Points[2:] {
		lastValue = currentValue
		currentValue, _ = dp.IndependentValue(at.TimeVariable())

		diff := currentValue - lastValue
		if math.Abs(at.timeDiff-diff) > Tolerance {
			return fmt.Errorf("inconsistent intervals found in time series, using variable '%s'", at.TimeVariable())
		}

		if _, err = at.initForecastValue(currentValue); err != nil {
			return err
		}
	}

	testDataSet := openforecast.NewDataSetCopy(at.observedValues)
	for var10 := 0; var10 < at.impl.NumberOfPeriods(); var10++ {
		testDataSet.Points = testDataSet.Points[1:]
	}

	return at.calculateAccuracyIndicators(testDataSet)
}

// Forecast -> GetForecastValue
func (at *AbstractTimeBasedModel) Forecast(dp openforecast.DataPoint) (float64, error) {
	if at.AccuracyIndicators == uninitializedAccuracyIndicators {
		return 0, ErrUninitialized
	}

	t, ok := dp.IndependentValue(at.TimeVariable())
	if !ok {
		return 0, ErrIllegalArgument
	}
	return at.GetForecastValue(t)
}

// GetForecastValue -> initForecastValue
func (at *AbstractTimeBasedModel) GetForecastValue(timeValue float64) (float64, error) {
	// find cache from forecastValues
	if timeValue >= at.minTimeValue-Tolerance && timeValue <= at.maxTimeValue+Tolerance {
		for _, dp := range at.forecastValues.Points {
			if t, ok := dp.IndependentValue(at.TimeVariable()); ok && math.Abs(t-timeValue) < Tolerance {
				return dp.DependentValue(), nil
			}
		}
	}

	return at.initForecastValue(timeValue)
}

// initForecastValue -> impl.ForecastTime
func (at *AbstractTimeBasedModel) initForecastValue(timeValue float64) (float64, error) {
	forecast, err := at.impl.ForecastTime(timeValue)
	if err != nil {
		return 0, err
	}
	dp := openforecast.NewObservation(forecast)
	dp.SetIndependentValue(at.TimeVariable(), timeValue)
	// TODO(StitchCula): out-of-order
	at.forecastValues.Points = append(at.forecastValues.Points, dp)
	if timeValue > at.maxTimeValue {
		at.maxTimeValue = timeValue
	}

	return forecast, nil
}

func (at *AbstractTimeBasedModel) GetObservedValue(timeValue float64) (float64, error) {
	for _, dp := range at.observedValues.Points {
		if t, ok := dp.IndependentValue(at.TimeVariable()); ok && math.Abs(t-timeValue) < Tolerance {
			return dp.DependentValue(), nil
		}
	}
	return 0, ErrNotFound
}

func (at *AbstractTimeBasedModel) getTimeVariable(dataSet *openforecast.DataSet) (timeVariable string, err error) {
	if timeVariable = dataSet.TimeVariable(); timeVariable != "" {
		return
	}
	if independentVars := dataSet.IndependentVariables(); len(independentVars) == 1 {
		return independentVars[0], nil
	}
	return "", ErrIllegalArgument
}

type AbstractForecastingModel struct {
	*AccuracyIndicators
	impl openforecast.ForecastingModel
}

func NewAbstractForecastingModel(impl openforecast.ForecastingModel) *AbstractForecastingModel {
	return &AbstractForecastingModel{
		AccuracyIndicators: uninitializedAccuracyIndicators,
		impl:               impl,
	}
}

// ForecastAll -> impl.Forecast
func (af *AbstractForecastingModel) ForecastAll(dataSet *openforecast.DataSet) (*openforecast.DataSet, error) {
	if af.AccuracyIndicators == uninitializedAccuracyIndicators {
		return nil, ErrUninitialized
	}
	for _, dp := range dataSet.Points {
		v, err := af.impl.Forecast(dp)
		if err != nil {
			return nil, errors.New("forecast " + dp.String() + " " + err.Error())
		}
		dp.SetDependentValue(v)
	}
	return dataSet, nil
}

func (af *AbstractForecastingModel) calculateAccuracyIndicators(dataSet *openforecast.DataSet) error {
	af.AccuracyIndicators = NewAccuracyIndicators()

	forecastValues, err := af.ForecastAll(openforecast.NewDataSetCopy(dataSet))
	if err != nil {
		return err
	}

	var sumErr, sumAbsErr, sumAbsPercentErr, sumErrSquared float64
	for i, fdp := range forecastValues.Points {
		dp := dataSet.Points[i]
		x0 := dp.DependentValue()
		x1 := fdp.DependentValue()

		deta := x1 - x0
		sumErr += deta
		sumAbsErr += math.Abs(deta)
		sumAbsPercentErr += math.Abs(deta / x0)
		sumErrSquared += deta * deta
	}

	n := float64(len(dataSet.Points))
	p := float64(af.impl.NumberOfPredictors())
	af.AccuracyIndicators.SetAIC(n*math.Log(6.283185307179586) + math.Log(sumErrSquared/n) + (2 * (p + 2)))
	af.AccuracyIndicators.SetBias(sumErr / n)
	af.AccuracyIndicators.SetMAD(sumAbsErr / n)
	af.AccuracyIndicators.SetMAPE(sumAbsPercentErr / n)
	af.AccuracyIndicators.SetMSE(sumErrSquared / n)
	af.AccuracyIndicators.SetSAE(sumAbsErr)
	return nil
}

var uninitializedAccuracyIndicators = &AccuracyIndicators{aic: -1, bias: -1, mad: -1, mape: -1, mse: -1, sae: -1}

type AccuracyIndicators struct {
	aic  float64
	bias float64
	mad  float64
	mape float64
	mse  float64
	sae  float64
}

func NewAccuracyIndicators() *AccuracyIndicators {
	const def = 1.7976931348623157E308
	return &AccuracyIndicators{
		aic:  def,
		bias: def,
		mad:  def,
		mape: def,
		mse:  def,
		sae:  def,
	}
}

func (ai *AccuracyIndicators) AIC() float64         { return ai.aic }
func (ai *AccuracyIndicators) Bias() float64        { return ai.bias }
func (ai *AccuracyIndicators) MAD() float64         { return ai.mad }
func (ai *AccuracyIndicators) MAPE() float64        { return ai.mape }
func (ai *AccuracyIndicators) MSE() float64         { return ai.mse }
func (ai *AccuracyIndicators) SAE() float64         { return ai.sae }
func (ai *AccuracyIndicators) SetAIC(aic float64)   { ai.aic = aic }
func (ai *AccuracyIndicators) SetBias(bias float64) { ai.bias = bias }
func (ai *AccuracyIndicators) SetMAD(mad float64)   { ai.mad = mad }
func (ai *AccuracyIndicators) SetMAPE(mape float64) { ai.mape = mape }
func (ai *AccuracyIndicators) SetMSE(mse float64)   { ai.mse = mse }
func (ai *AccuracyIndicators) SetSAE(sae float64)   { ai.sae = sae }

func (ai *AccuracyIndicators) String() string {
	return fmt.Sprintf("AIC=%f, Bias=%f, MAD=%f, MAPE=%f, MSE=%f, SAE=%f", ai.aic, ai.bias, ai.mad, ai.mape, ai.mse, ai.sae)
}
