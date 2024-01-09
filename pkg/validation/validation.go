package validation

import (
	"fmt"
	"os"
	"path/filepath"
)

type AgentExists func(string) (bool, error)

type validateAgentExistence struct {
	agentExists AgentExists
}

func agentFileExists(agent string) (bool, error) {
	directory := "/usr/sbin/"
	fullPath := filepath.Join(directory, agent)
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("error checking file: %w", err)
}

type AgentValidator interface {
	ValidateAgentName(agent string) (bool, error)
}

func NewAgentValidator() AgentValidator {
	return &validateAgentExistence{agentExists: agentFileExists}
}

func NewCustomAgentValidator(agentExists AgentExists) AgentValidator {
	return &validateAgentExistence{agentExists: agentExists}
}

func (vfe *validateAgentExistence) ValidateAgentName(agent string) (bool, error) {
	return vfe.agentExists(agent)
}
