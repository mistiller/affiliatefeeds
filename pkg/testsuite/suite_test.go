// +build !unit
// +build integration

package testsuite

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestFeedTestSuite(t *testing.T) {
	suite.Run(t, new(FeedTestSuite))
}
