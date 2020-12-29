package foraging

import (
	"github.com/SOMAS2020/SOMAS2020/internal/common/config"
	"github.com/SOMAS2020/SOMAS2020/internal/common/shared"
	"github.com/pkg/errors"
)

// CreateDeerHunt receives hunt participants and their contributions and returns a DeerHunt
func CreateDeerHunt(teamResourceInputs map[shared.ClientID]shared.Resources, dhConf config.DeerHuntConfig) (DeerHunt, error) {
	if len(teamResourceInputs) == 0 {
		return DeerHunt{}, errors.Errorf("No resource contributions specified!")
	}
	params := deerHuntParams{p: dhConf.BernoulliProb, lam: dhConf.ExponentialRate}
	return DeerHunt{ParticipantContributions: teamResourceInputs, params: params}, nil // returning error too for future use
}

// CreateFishingExpedition sees the participants and their contributions and returns the value of FishHunt
func CreateFishingExpedition(teamResourceInputs map[shared.ClientID]shared.Resources) (FishingExpedition, error) {
	if len(teamResourceInputs) == 0 {
		return FishingExpedition{}, errors.Errorf("No resource contributions specified - %v")
	}
	fConf := config.GameConfig().ForagingConfig.FishingConfig
	params := fishingParams{Mu: fConf.Mean, Sigma: fConf.Variance}
	return FishingExpedition{ParticipantContributions: teamResourceInputs, params: params}, nil // returning error too for future use
}

// CreateDeerPopulationModel returns the target population model. The formulation of this model should be changed here before runtime
func CreateDeerPopulationModel(dhConf config.DeerHuntConfig) DeerPopulationModel {
	return createBasicDeerPopulationModel(dhConf)
}
