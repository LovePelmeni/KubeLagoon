package resource_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmware/govmomi/object"
)

type ResourceTestSuite struct {
	suite.Suite
	Resources []object.Reference
}

func TestResourceSuite(t *testing.T) {
	suite.Run(t, new(ResourceTestSuite))
}

func (this *ResourceTestSuite) SetupTest() {
	this.Resources = []object.Reference{}
}
func (this *ResourceTestSuite) TestResources() {
	testing.RunTests(func(str string, path string) (bool, error) { return true, nil },
		[]testing.InternalTest{
			{"Creating Resources using Valid Data", func(T *testing.T) {
				// Creates New Resources
			}},

			{"Getting Resources using Resource Item Paths", func(T *testing.T) {
				// Getting Existing Resources
			}},

			{"Deleting Resources using", func(T *testing.T) {
				// Deleting Resources
			}},
		})
}
