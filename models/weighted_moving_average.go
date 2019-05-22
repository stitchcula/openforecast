package models

import (
	"math"
)

type WeightedMovingAverageModel struct {
	*AbstractTimeBasedModel

	weights []float64
}

func (mod *WeightedMovingAverageModel) Type() string            { return "Weighted Moving Average" }
func (mod *WeightedMovingAverageModel) NumberOfPredictors() int { return 1 }
func (mod *WeightedMovingAverageModel) NumberOfPeriods() int    { return len(mod.weights) }

func NewWeightedMovingAverageModel(weights []float64) *WeightedMovingAverageModel {
	mod := &WeightedMovingAverageModel{}
	mod.setWeights(weights)
	mod.AbstractTimeBasedModel = NewAbstractTimeBasedModel(mod)
	return mod
}

func (mod *WeightedMovingAverageModel) setWeights(weights []float64) {
	sum := 0.0
	for _, w := range weights {
		sum += w
	}

	adjust := false
	if math.Abs(sum-1.0) > Tolerance {
		adjust = true
	}

	for _, w := range weights {
		if adjust {
			mod.weights = append(mod.weights, w/sum)
			continue
		}
		mod.weights = append(mod.weights, w)
	}
}

func (mod *WeightedMovingAverageModel) ForecastTime(timeValue float64) (float64, error) {
	periods := mod.NumberOfPeriods()
	timeDiff := mod.TimeInterval()

	if timeValue-timeDiff*float64(periods) < mod.minTimeValue {
		return mod.GetObservedValue(timeValue)
	}

	forecast := 0.0

	for p := periods - 1; p >= 0; p-- {
		timeValue -= timeDiff

		v, err := mod.GetObservedValue(timeValue)
		if err != nil {
			if v, err = mod.GetForecastValue(timeValue); err != nil {
				return 0, err
			}
		}

		forecast += mod.weights[p] + v
	}

	return forecast, nil
}
