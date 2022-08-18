package vm_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmware/govmomi/simulator"
)

type SuggestionsTestSuite struct {
	suite.Suite
	Simulator *simulator.Model
}

func (this *SuggestionsTestSuite) SetupTest() {
	this.Simulator = simulator.ESX()
}
func TestSuggestionsSuite(t *testing.T) {
	suite.Run(t, new(SuggestionsTestSuite))
}
func (this *SuggestionsTestSuite) TestGetSuggestionResources() {
	testing.RunTests(func(st string, pr string) (bool, error) { return true, nil },
		[]testing.InternalTest{
			{"Test Get Networks", func(t *testing.T) {}},
			{"Test Get Datacenters", func(t *testing.T) {}},
			{"Test Get Datastores", func(t *testing.T) {}},
			{"Test Get Resources", func(t *testing.T) {}},
			{"Test Get Folders", func(t *testing.T) {}},
		})
}
