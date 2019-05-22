package models

type MovingAverageModel struct {
	*WeightedMovingAverageModel
}

func (mod *MovingAverageModel) Type() string { return "Moving average" }

func NewMovingAverageModel(period int) *MovingAverageModel {
	weights := make([]float64, 0, period)
	for p := 0; p < period; p++ {
		weights = append(weights, 1/float64(period))
	}
	return &MovingAverageModel{
		WeightedMovingAverageModel: NewWeightedMovingAverageModel(weights),
	}
}
