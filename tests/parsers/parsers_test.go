package parsers_test

import (
	"encoding/json"
	"testing"

	"github.com/LovePelmeni/Infrastructure/parsers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ParseTestSuite struct {
	suite.Suite
	Parsers []interface{}
}

func TestParseTestSuite(t *testing.T) {
	suite.Run(t, new(ParseTestSuite))
}

func (this *ParseTestSuite) SetupTest() {
	this.Parsers = append(this.Parsers, []interface{}{parsers.DatacenterConfig{}, parsers.VirtualMachineCustomSpec{}}...)
}
func (this *ParseTestSuite) TestParsers() {
	testing.RunTests(func(st string, pa string) (bool, error) { return true, nil },
		[]testing.InternalTest{

			{"Testing Datacenter Configuration Parser using Valid Configuration", func(t *testing.T) {
				SerializedConfig, _ := json.Marshal(struct {
					Datacenter struct {
					  ItemPath string `json:"ItemPath"`
					} `json:"Datacenter"`
				}{})
				Configuration, ParserError := parsers.NewHardwareConfig(string(SerializedConfig))
				assert.NoError(this.T(), ParserError, "Error should be Nil, Because has been Passed Valid Config")
				assert.Nil(this.T(), Configuration, "Datacenter info should be Empty Because, Invalid Configuration has been Passed.")
			}},

			{"Testing Datacenter Configuration Parser using Invalid Configuration", func(t *testing.T) {
				SerializedInvalidConfig, _ := json.Marshal(struct{}{})
				Configuration, ParserError := parsers.NewHardwareConfig(string(SerializedInvalidConfig))
				assert.Error(this.T(), ParserError, "Error should be not None, Because config Is Not Valid.")
				assert.NotNil(this.T(), Configuration.Datacenter.ItemPath, "Failed to Parse Valid Configuration")
			}},

			{"Testing Customized Configuration Parser, using Invalid Customized Configuration", func(t *testing.T) {

			}},
			{"Testing Customized Configuration Parser, using Valid Customized Configuration", func(t *testing.T) {

			}},
		})
}
