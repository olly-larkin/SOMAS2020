package server

import (
	"fmt"
	"log"

	"github.com/SOMAS2020/SOMAS2020/internal/common/baseclient"
	"github.com/SOMAS2020/SOMAS2020/internal/common/config"
	"github.com/SOMAS2020/SOMAS2020/internal/common/disasters"
	"github.com/SOMAS2020/SOMAS2020/internal/common/foraging"
	"github.com/SOMAS2020/SOMAS2020/internal/common/gamestate"
	"github.com/SOMAS2020/SOMAS2020/internal/common/rules"
	"github.com/SOMAS2020/SOMAS2020/internal/common/shared"
	"github.com/pkg/errors"
)

// Server represents the primary server interface exposed to the simulation.
type Server interface {
	// EntryPoint function that returns a list of historic gamestate.ClientInfos until the
	// game ends.
	EntryPoint() ([]gamestate.GameState, error)
}

// SOMASServer implements Server.
type SOMASServer struct {
	gameState gamestate.GameState

	gameConfig config.Config

	// ClientMap maps from the ClientID to the Client object.
	// We don't store this in gameState--gameState is shared to clients and should
	// not contain pointers to other clients!
	clientMap map[shared.ClientID]baseclient.Client
}

// NewSOMASServer returns an instance of the main server we use.
func NewSOMASServer(gameConfig config.Config) Server {
	clientInfos, clientMap := getClientInfosAndMapFromRegisteredClients(
		baseclient.RegisteredClients,
		gameConfig.InitialResources,
	)
	return createSOMASServer(clientInfos, clientMap, gameConfig)
}

// createSOMASServer creates the main server given initial data about the
// clients. Extracted from SOMASServerFactory for testing purposes.
func createSOMASServer(
	clientInfos map[shared.ClientID]gamestate.ClientInfo,
	clientMap map[shared.ClientID]baseclient.Client,
	gameConfig config.Config,
) Server {
	clientIDs := make([]shared.ClientID, 0, len(clientMap))
	for k := range clientMap {
		clientIDs = append(clientIDs, k)
	}

	forageHistory := map[shared.ForageType][]foraging.ForagingReport{}
	for _, t := range shared.AllForageTypes() {
		forageHistory[t] = make([]foraging.ForagingReport, 0)
	}

	if gameConfig.IIGOConfig.StartWithRulesInPlay {
		for ruleName := range rules.AvailableRules {
			// Result is ignored since we know that the RulesInPlay cache cannot contain any of
			// these rules (the only error case)
			_ = rules.PullRuleIntoPlay(ruleName)
		}
	}

	server := &SOMASServer{
		clientMap:  clientMap,
		gameConfig: gameConfig,
		gameState: gamestate.GameState{
			Season:          1,
			Turn:            1,
			ClientInfos:     clientInfos,
			Environment:     disasters.InitEnvironment(clientIDs, gameConfig.DisasterConfig),
			ForagingHistory: forageHistory,
			IIGOHistory:     map[uint][]shared.Accountability{},
			IIGORolesBudget: map[shared.Role]shared.Resources{
				shared.President: 0,
				shared.Judge:     0,
				shared.Speaker:   0,
			},
			IIGOTurnsInPower: map[shared.Role]uint{
				shared.President: 0,
				shared.Judge:     0,
				shared.Speaker:   0,
			},
			SpeakerID:   shared.Team1,
			JudgeID:     shared.Team2,
			PresidentID: shared.Team3,
			CommonPool:  gameConfig.InitialCommonPool,
			RulesInfo: gamestate.RulesContext{
				AvailableRules:     rules.AvailableRules,
				CurrentRulesInPlay: rules.RulesInPlay,
				VariableMap:        rules.InitialVarRegistration(),
			},
		},
	}

	server.gameState.DeerPopulation = foraging.CreateDeerPopulationModel(gameConfig.ForagingConfig.DeerHuntConfig, server.logf)

	for _, client := range clientMap {
		client.Initialise(ServerForClient{
			clientID: client.GetID(),
			server:   server,
		})
	}

	return server
}

// EntryPoint function that returns a list of historic gamestate.GameState until the
// game ends.
func (s *SOMASServer) EntryPoint() ([]gamestate.GameState, error) {
	states := []gamestate.GameState{s.gameState.Copy()}

	for !s.gameOver(s.gameConfig.MaxTurns, s.gameConfig.MaxSeasons) {
		if err := s.runTurn(); err != nil {
			return states, err
		}
		states = append(states, s.gameState.Copy())
	}
	return states, nil
}

// getEcho retrieves an echo from all the clients and make sure they are the same.
func (s *SOMASServer) getEcho(str string) error {
	for _, c := range s.clientMap {
		got := c.Echo(str)
		if str != got {
			return errors.Errorf("Echo error: want '%v' got '%v' from %v",
				str, got, c.GetID())
		}
		s.logf("Received echo `%v` from %v", str, c.GetID())
	}
	return nil
}

// logf is the server's default logger.
func (s *SOMASServer) logf(format string, a ...interface{}) {
	log.Printf("[SERVER]: %v", fmt.Sprintf(format, a...))
}

// ServerForClient is a reference to the server for particular client. It implements baseclient.ServerReadHandle
type ServerForClient struct {
	clientID shared.ClientID
	server   *SOMASServer
}

// GetGameState gets the ClientGameState for the client matching s.clientID in
// s.server
func (s ServerForClient) GetGameState() gamestate.ClientGameState {
	return s.server.gameState.GetClientGameStateCopy(s.clientID)
}

// GetGameConfig returns ClientConfig which is a subset of the entire Config that is visible to clients.
func (s ServerForClient) GetGameConfig() config.ClientConfig {
	return s.server.gameConfig.GetClientConfig()
}
