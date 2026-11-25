package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *MainTestSuite) uploadRouter(fileSizeLimit int64) *gin.Engine {
	router := gin.New()
	router.POST("/purgatory", limitBody(fileSizeLimit), s.controller.UploadFile)

	return router
}

func (s *MainTestSuite) multipartBody(filename string, content []byte) (*bytes.Buffer, string, string) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	part, err := w.CreateFormFile("file", filename)
	s.Require().NoError(err)
	_, err = part.Write(content)
	s.Require().NoError(err)
	s.Require().NoError(w.Close())
	return body, w.FormDataContentType(), w.Boundary()
}

func (s *MainTestSuite) TestUploadRejectsFakeArchive() {
	body, contentType, _ := s.multipartBody("evil.cbz", []byte("hello, i definitely not a zip file"))

	req := httptest.NewRequest(http.MethodPost, "/purgatory", body)
	req.Header.Set("Content-Type", contentType)

	resp := httptest.NewRecorder()
	s.uploadRouter(1024).ServeHTTP(resp, req)

	s.Assert().Equal(http.StatusUnsupportedMediaType, resp.Code)
	s.Assert().Contains(resp.Body.String(), "unsupported file type")
}

func (s *MainTestSuite) TestUploadRejectsTraversalName() {
	body, _, boundary := s.multipartBody("../../evil.cbz", append([]byte("PK\x03\x04"), []byte("garbage-but-prefixed")...))

	form, err := multipart.NewReader(body, boundary).ReadForm(32 * MiB)
	s.Require().NoError(err)
	s.Require().Len(form.File["file"], 1)

	dest, destPath, err := s.service.UploadTempFile(form.File["file"][0])
	s.Require().NoError(err)

	defer func() { _ = dest.Close() }()
	defer func() { _ = os.Remove(destPath) }()

	s.Assert().True(
		strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(s.filesPath)+string(os.PathSeparator)),
		"destPath escaped files dir: %s", destPath,
	)
	s.Assert().FileExists(destPath)
}

func (s *MainTestSuite) TestUploadLimitReturns413() {
	token := s.createTestToken("testuser")

	fileContent := append([]byte("PK\x03\x04"), bytes.Repeat([]byte("x"), 2048)...)
	body, contentType, _ := s.multipartBody("evil.cbz", fileContent)

	req := httptest.NewRequest(http.MethodPost, "/purgatory", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)

	resp := httptest.NewRecorder()

	router := s.uploadRouter(1024)
	router.ServeHTTP(resp, req)

	s.Assert().Equal(http.StatusRequestEntityTooLarge, resp.Code)
	s.Assert().Equal("{\"error\":\"Content too large\"}", resp.Body.String())
}
