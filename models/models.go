package models

import (
	"errors"
	"fmt"
	"github.com/stitchcula/openforecast"
)

var (
	ErrUninitialized   = errors.New("NotInitialized")
	ErrUnimplemented   = errors.New("ErrUnimplemented")
	ErrIllegalArgument = errors.New("IllegalArgument")
)

type AbstractTimeBasedModel struct {
	timeVariable string
}

func NewAbstractTimeBasedModel() *AbstractTimeBasedModel {
	return &AbstractTimeBasedModel{}
}

func (at *AbstractTimeBasedModel) Train(dataSet *openforecast.DataSet) (err error) {
	if at.timeVariable, err = at.getTimeVariable(dataSet); err != nil {
		return
	}
	// TODO:
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

// TODO:
func (af *AbstractForecastingModel) calculateAccuracyIndicators(dataSet *openforecast.DataSet) error {
	/*
		forecastValues, err := af.ForecastAll(openforecast.NewDataSetCopy(dataSet))
		if err != nil {
			return err
		}

		var sumErr, sumAbsErr, sumAbsPercentErr, sumErrSquared float64

		af.AccuracyIndicators = NewAccuracyIndicators()
	*/
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
