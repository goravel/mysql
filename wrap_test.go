package mysql

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type WrapTestSuite struct {
	suite.Suite
	wrap *Wrap
}

func TestWrapSuite(t *testing.T) {
	suite.Run(t, new(WrapTestSuite))
}

func (s *WrapTestSuite) SetupTest() {
	s.wrap = NewWrap("prefix_")
}

func (s *WrapTestSuite) TestValue() {
	// With asterisk
	result := s.wrap.Value("*")
	s.Equal("*", result)

	// Without asterisk
	result = s.wrap.Value("value")
	s.Equal("`value`", result)
}
