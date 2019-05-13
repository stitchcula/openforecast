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
	ErrUnimplemented   = errors.New("ErrUnimplemented")
	ErrIllegalArgument = errors.New("IllegalArgument")
)

type AbstractTimeBasedModel struct {
	*AbstractForecastingModel
	impl openforecast.ForecastingModel

	timeVariable   string
	timeDiff       float64
	minPeriods     int
	observedValues *openforecast.DataSet
	forecastValues *openforecast.DataSet
	minTimeValue   float64
	maxTimeValue   float64
}

func NewAbstractTimeBasedModel(impl openforecast.ForecastingModel, minPeriods int) *AbstractTimeBasedModel {
	return &AbstractTimeBasedModel{
		AbstractForecastingModel: NewAbstractForecastingModel(impl),
		impl:                     impl,
		minPeriods:               minPeriods,
	}
}

// TODO(StitchCula): 没看懂
func (at *AbstractTimeBasedModel) Train(dataSet *openforecast.DataSet) (err error) {
	if at.timeVariable, err = at.getTimeVariable(dataSet); err != nil {
		return
	}
	if len(dataSet.Points) < at.minPeriods {
		return ErrIllegalArgument
	}

	at.observedValues = openforecast.NewDataSetCopy(dataSet)
	at.observedValues.Sort(at.timeVariable)
	lastValue, _ := at.observedValues.Points[0].IndependentValue(at.timeVariable)
	currentValue, _ := at.observedValues.Points[1].IndependentValue(at.timeVariable)
	at.forecastValues = openforecast.NewDataSet(at.timeVariable, 0, nil)
	at.timeDiff = currentValue - lastValue
	at.minTimeValue = lastValue

	for _, dp := range at.observedValues.Points[2:] {
		lastValue = currentValue
		currentValue, _ = dp.IndependentValue(at.timeVariable)

		diff := currentValue - lastValue
		if math.Abs(at.timeDiff-diff) > Tolerance {
			return fmt.Errorf("inconsistent intervals found in time series, using variable '%s'", at.timeVariable)
		}

		if _, err = at.forecastValue(currentValue); err != nil {
			return err
		}
	}

	testDataSet := openforecast.NewDataSetCopy(at.observedValues)
	for var10 := 0; var10 < at.minPeriods; var10++ {
		testDataSet.Points = testDataSet.Points[1:]
	}

	return at.calculateAccuracyIndicators(testDataSet)
}

func (at *AbstractTimeBasedModel) forecastValue(timeValue float64) (float64, error) {
	dp := openforecast.NewObservation(0)
	forecast, err := at.impl.Forecast(dp)
	if err != nil {
		return 0, err
	}
	dp.SetIndependentValue(at.timeVariable, forecast)
	at.forecastValues.Points = append(at.forecastValues.Points, dp)

	if timeValue > at.maxTimeValue {
		at.maxTimeValue = timeValue
	}

	return forecast, nil
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
