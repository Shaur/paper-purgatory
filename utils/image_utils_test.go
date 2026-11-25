package utils

import (
	"bytes"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/image/webp"
)

type UtilsTestSuite struct {
	suite.Suite
}

func TestUtils(t *testing.T) {
	suite.Run(t, new(UtilsTestSuite))
}

func (s *UtilsTestSuite) TestConvertWebp() {
	src, err := os.ReadFile("testdata/test.webp")
	s.Require().NoError(err)

	dest := filepath.Join(s.T().TempDir(), "test.jpeg")

	err = NormalizePage(bytes.NewBuffer(src), dest)
	s.Require().NoError(err)

	file, err := os.ReadFile(dest)
	s.Require().NoError(err)

	s.Assert().True(bytes.HasPrefix(file, []byte{0xFF, 0xD8, 0xFF}), "output is not JPEG")
	s.Assert().False(bytes.Contains(file, []byte("WEBP")))

	srcImg, err := webp.Decode(bytes.NewReader(src))
	s.Require().NoError(err)
	outImg, err := jpeg.Decode(bytes.NewReader(file))
	s.Require().NoError(err)
	s.Assert().Equal(srcImg.Bounds(), outImg.Bounds())
}
