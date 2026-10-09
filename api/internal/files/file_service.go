package files

import (
	"atonixcorp/api/internal/models"
	"mime/multipart"
)

type FileService struct {
	repo *FileRepository
}

func NewFileService(repo *FileRepository) *FileService {
	return &FileService{repo}
}

func (s *FileService) Upload(file multipart.File, header *multipart.FileHeader) (*models.File, error) {
	path, err := SaveToDisk(file, header.Filename)
	if err != nil {
		return nil, err
	}

	f := &models.File{
		FileName: header.Filename,
		FilePath: path,
		Size:     header.Size,
		MimeType: header.Header.Get("Content-Type"),
	}

	err = s.repo.Save(f)
	return f, err
}
