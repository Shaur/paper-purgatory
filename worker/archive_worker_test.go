package worker

import (
	"context"
	"fmt"
	"io"
	"os"
	"paper/purgatory/model"
	"paper/purgatory/service"
	"paper/purgatory/utils"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	postgresContainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type WorkerTestSuite struct {
	suite.Suite
	pgContainer testcontainers.Container
	db          *gorm.DB
	worker      *Worker
	filesPath   string
}

func (s *WorkerTestSuite) SetupSuite() {
	ctx := context.Background()

	var err error
	s.pgContainer, err = postgresContainer.Run(
		ctx,
		"postgres:18-alpine",
		postgresContainer.WithDatabase("testdb"),
		postgresContainer.WithUsername("testuser"),
		postgresContainer.WithPassword("testpassword"),
		postgresContainer.BasicWaitStrategies(),
	)

	s.Require().NoError(err, "Failed to start PostgreSQL container")

	mappedPort, err := s.pgContainer.MappedPort(ctx, "5432")
	s.Require().NoError(err, "Failed to get mapped port")

	host, err := s.pgContainer.Host(ctx)
	s.Require().NoError(err, "Failed to get host")

	dsn := fmt.Sprintf("host=%s user=testuser password=testpassword dbname=testdb port=%s sslmode=disable",
		host, mappedPort.Port())

	s.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	s.Require().NoError(err, "Failed to connect to database")

	err = s.db.AutoMigrate(&model.Job{})
	s.Require().NoError(err, "Failed to migrate database schema")

	s.Require().NoError(s.db.Exec(`alter table job add constraint job_status_valid check (status in ('pending','started','done','failed'))`).Error)

	err = s.db.AutoMigrate(&model.PurgatoryItem{})
	s.Require().NoError(err, "Failed to migrate database schema")

	s.filesPath = s.T().TempDir()

	purgatoryService := service.Init(s.db, s.filesPath)

	s.worker = NewWorker(s.db, purgatoryService, 2*time.Minute)
}

func (s *WorkerTestSuite) SetupTest() {
	s.Require().NoError(s.db.Exec("delete from job").Error)
	s.Require().NoError(s.db.Exec("delete from purgatory").Error)
}

func (s *WorkerTestSuite) TestHappyPath() {
	job := model.Job{
		FileName:     "comics.cbz",
		FilePath:     "testdata/test.cbz",
		Status:       "pending",
		UpdateTime:   time.Now(),
		UploadedBy:   "testuser",
		ErrorMessage: nil,
		Attempts:     0,
	}

	s.db.Create(&job)

	processed := s.worker.ProcessOnce()
	s.Require().True(processed)

	s.db.Find(&job)

	s.Require().Equal(job.Status, "done")
	s.Require().Equal(job.Attempts, 1)

	var item model.PurgatoryItem
	s.db.Find(&item)

	s.Require().NotNil(item.Meta)
	s.Require().Equal(item.Meta.PagesCount, 1)
	s.Require().Equal(item.Meta.Number, "1")
	s.Require().Equal(item.Meta.SeriesName, "Jean Grey")
}

func (s *WorkerTestSuite) TestBrokenArchive() {
	job := model.Job{
		FileName:     "comics.cbz",
		FilePath:     s.filesPath + "/broken-archive.cbz",
		Status:       "pending",
		UpdateTime:   time.Now(),
		UploadedBy:   "testuser",
		ErrorMessage: nil,
		Attempts:     0,
	}

	s.db.Create(&job)

	brokenArchive, err := os.Create(s.filesPath + "/broken-archive.cbz")
	s.Require().NoError(err, "Failed to create broken archive")

	_, err = brokenArchive.Write([]byte("test"))
	s.Require().NoError(err, "Failed to fill broken archive")

	utils.HandleClose(brokenArchive.Close)

	processed := s.worker.ProcessOnce()
	s.Require().True(processed)

	s.db.Find(&job)

	s.Require().Equal("failed", job.Status)
	s.Require().Equal(1, job.Attempts)
}

func (s *WorkerTestSuite) TestEmptyQueue() {
	processed := s.worker.ProcessOnce()
	s.Require().False(processed)
}

func (s *WorkerTestSuite) TestGCHappyPath() {
	job := model.Job{
		FileName:     "comics.cbz",
		FilePath:     s.filesPath + "/comics.cbz",
		Status:       "done",
		UpdateTime:   time.Now(),
		UploadedBy:   "testuser",
		ErrorMessage: nil,
		Attempts:     0,
	}

	res := s.db.Create(&job)
	s.Require().NoError(res.Error)

	dst, err := os.Create(s.filesPath + "/comics.cbz")
	s.Require().NoError(err, "Failed to create comics.cbz")
	defer utils.HandleClose(dst.Close)

	src, err := os.Open("testdata/test.cbz")
	s.Require().NoError(err, "Failed to open test.cbz")
	defer utils.HandleClose(src.Close)

	_, err = io.Copy(dst, src)
	s.Require().NoError(err, "Failed to copy comics.cbz")

	s.worker.collectAndDelete()

	var rowsCount int64
	res = s.db.Model(model.Job{}).Count(&rowsCount)

	s.Require().Equal(int64(0), rowsCount)
	_, err = os.Open(s.filesPath + "/comics.cbz")
	s.Require().True(os.IsNotExist(err))
}

func (s *WorkerTestSuite) TestRecoveryStartedJob() {
	job := model.Job{
		FileName:     "comics.cbz",
		FilePath:     s.filesPath + "/comics.cbz",
		Status:       "started",
		UpdateTime:   time.Now().Add(-2 * time.Hour),
		UploadedBy:   "testuser",
		ErrorMessage: nil,
		Attempts:     1,
	}

	res := s.db.Create(&job)
	s.Require().NoError(res.Error)

	err := s.worker.recovery()
	s.Require().NoError(err, "Failed to recover comics.cbz")

	s.db.Find(&job)

	s.Require().Equal("pending", job.Status)
	s.Require().Equal(1, job.Attempts)
}

func (s *WorkerTestSuite) TestRecoveryStartedJobFailed() {
	job := model.Job{
		FileName:     "comics.cbz",
		FilePath:     s.filesPath + "/comics.cbz",
		Status:       "started",
		UpdateTime:   time.Now().Add(-2 * time.Hour),
		UploadedBy:   "testuser",
		ErrorMessage: nil,
		Attempts:     3,
	}

	res := s.db.Create(&job)
	s.Require().NoError(res.Error)

	err := s.worker.recovery()
	s.Require().NoError(err, "Failed to recover comics.cbz")

	s.db.Find(&job)

	s.Require().Equal("failed", job.Status)
	s.Require().Equal(3, job.Attempts)
	s.Require().NotNil(job.ErrorMessage)
	s.Require().Equal("max attempts exceeded", *job.ErrorMessage)
}

func (s *WorkerTestSuite) TestDeleteJobWithoutFile() {
	job := model.Job{
		FileName:     "comics.cbz",
		FilePath:     s.filesPath + "/file-not-found.cbz",
		Status:       "done",
		UpdateTime:   time.Now().Add(-2 * time.Hour),
		UploadedBy:   "testuser",
		ErrorMessage: nil,
		Attempts:     1,
	}

	res := s.db.Create(&job)
	s.Require().NoError(res.Error)

	s.worker.collectAndDelete()

	var rowsCount int64
	res = s.db.Model(model.Job{}).Count(&rowsCount)

	s.Require().Equal(int64(0), rowsCount)
	_, err := os.Open(s.filesPath + "/file-not-found.cbz")
	s.Require().True(os.IsNotExist(err))
}

func (s *WorkerTestSuite) TestDontRecoveryNewJobs() {
	pendingJob := model.Job{
		FileName:     "comics.cbz",
		FilePath:     s.filesPath + "/file-not-found.cbz",
		Status:       "pending",
		UpdateTime:   time.Now().Add(-1 * time.Minute),
		UploadedBy:   "testuser",
		ErrorMessage: nil,
		Attempts:     1,
	}

	startedJob := pendingJob
	startedJob.Status = "started"

	res := s.db.CreateInBatches(&[]model.Job{pendingJob, startedJob}, 2)
	s.Require().NoError(res.Error)

	err := s.worker.recovery()
	s.Require().NoError(err, "Failed to recover comics.cbz")

	s.Require().NoError(s.db.Find(&pendingJob, "status = 'pending'").Error)
	s.Require().NoError(s.db.Find(&startedJob, "status = 'started'").Error)

	s.Require().Equal("pending", pendingJob.Status)
	s.Require().Equal("started", startedJob.Status)

}

func TestArchiveWorker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	suite.Run(t, new(WorkerTestSuite))
}
