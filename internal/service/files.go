package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	files_v1 "github.com/voice-chat-team/contracts/gen/go/files/v1"
	"github.com/voice-chat-team/services.files/internal/repository"
	"github.com/voice-chat-team/services.files/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const urlTTL = 10 * time.Minute

type FileService struct {
	files_v1.UnimplementedFileServiceServer
	repo    *repository.FileRepository
	storage *storage.Storage
}

func NewFileService(repo *repository.FileRepository, st *storage.Storage) *FileService {
	return &FileService{repo: repo, storage: st}
}

func (s *FileService) CreateUpload(ctx context.Context, req *files_v1.CreateUploadRequest) (*files_v1.CreateUploadResponse, error) {
	rule, err := validateUpload(req)
	if err != nil {
		return nil, err
	}

	id := uuid.NewString()
	key := fmt.Sprintf("%s/%s/%s%s", rule.Dir, req.UserId, id, filepath.Ext(req.FileName))

	err = s.repo.Create(ctx, repository.File{
		ID:          id,
		OwnerID:     req.UserId,
		Purpose:     int16(req.Purpose),
		Status:      repository.StatusPending,
		Name:        req.FileName,
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
		S3Key:       key,
	})
	if err != nil {
		return nil, internalErr("create file", err)
	}

	uploadURL, err := s.storage.PresignUpload(ctx, key, urlTTL)
	if err != nil {
		return nil, internalErr("presign upload", err)
	}

	return &files_v1.CreateUploadResponse{
		FileId:        id,
		UploadUrl:     uploadURL,
		UploadHeaders: map[string]string{"Content-Type": req.ContentType},
		ExpiresAt:     time.Now().Add(urlTTL).Format(time.RFC3339),
	}, nil
}

func (s *FileService) ConfirmUpload(ctx context.Context, req *files_v1.ConfirmUploadRequest) (*files_v1.ConfirmUploadResponse, error) {
	f, err := s.getFile(ctx, req.FileId)
	if err != nil {
		return nil, err
	}

	if f.OwnerID != req.UserId {
		return nil, status.Error(codes.PermissionDenied, "not the owner of the file")
	}
	if f.Status != repository.StatusPending {
		return nil, status.Error(codes.FailedPrecondition, "file is not pending")
	}

	info, err := s.storage.Stat(ctx, f.S3Key)
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, status.Error(codes.FailedPrecondition, "file is not uploaded yet")
		}
		return nil, internalErr("stat object", err)
	}

	rule := rules[files_v1.FilePurpose(f.Purpose)]
	if info.Size != f.SizeBytes || info.Size > rule.MaxSize {
		s.removeObject(ctx, f.S3Key)
		return nil, status.Error(codes.InvalidArgument, "uploaded size does not match")
	}
	if rule.ImageOnly && !strings.HasPrefix(info.ContentType, "image/") {
		s.removeObject(ctx, f.S3Key)
		return nil, status.Error(codes.InvalidArgument, "only images are allowed")
	}

	ready, err := s.repo.MarkReady(ctx, f.ID, info.Size, f.OwnerID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, status.Error(codes.FailedPrecondition, "file is not pending")
	}
	if err != nil {
		return nil, internalErr("mark ready", err)
	}

	return &files_v1.ConfirmUploadResponse{File: toProto(ready, s.storage)}, nil
}

func (s *FileService) GetFile(ctx context.Context, req *files_v1.GetFileRequest) (*files_v1.GetFileResponse, error) {
	f, err := s.getFile(ctx, req.FileId)
	if err != nil {
		return nil, err
	}

	return &files_v1.GetFileResponse{File: toProto(f, s.storage)}, nil
}

func (s *FileService) GetFiles(ctx context.Context, req *files_v1.GetFilesRequest) (*files_v1.GetFilesResponse, error) {
	if len(req.FileIds) == 0 {
		return &files_v1.GetFilesResponse{}, nil
	}

	for _, id := range req.FileIds {
		if _, err := uuid.Parse(id); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid file id: %s", id)
		}
	}

	files, err := s.repo.GetByIds(ctx, req.FileIds)
	if err != nil {
		return nil, internalErr("get files", err)
	}

	out := make([]*files_v1.File, 0, len(files))
	for _, f := range files {
		out = append(out, toProto(f, s.storage))
	}

	return &files_v1.GetFilesResponse{Files: out}, nil
}

func (s *FileService) GetDownloadUrl(ctx context.Context, req *files_v1.GetDownloadUrlRequest) (*files_v1.GetDownloadUrlResponse, error) {
	f, err := s.getFile(ctx, req.FileId)
	if err != nil {
		return nil, err
	}

	if f.Status != repository.StatusReady {
		return nil, status.Error(codes.FailedPrecondition, "file is not ready")
	}

	downloadURL, err := s.storage.PresignDownload(ctx, f.S3Key, urlTTL)
	if err != nil {
		return nil, internalErr("presign download", err)
	}

	return &files_v1.GetDownloadUrlResponse{
		Url:       downloadURL,
		ExpiresAt: time.Now().Add(urlTTL).Format(time.RFC3339),
	}, nil
}

func (s *FileService) DeleteFile(ctx context.Context, req *files_v1.DeleteFileRequest) (
	*files_v1.DeleteFileResponse, error) {
	f, err := s.getFile(ctx, req.FileId)
	if err != nil {
		return nil, err
	}

	if f.OwnerID != req.UserId {
		return nil, status.Error(codes.PermissionDenied, "not the owner of the file")
	}

	err = s.repo.MarkDelete(ctx, f.ID, f.OwnerID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "file not found")
	}
	if err != nil {
		return nil, internalErr("mark delete", err)
	}

	s.removeObject(ctx, f.S3Key)

	return &files_v1.DeleteFileResponse{Success: true}, nil
}

func (s *FileService) removeObject(ctx context.Context, key string) {
	if err := s.storage.Remove(ctx, key); err != nil {
		log.Printf("remove object %s: %v", key, err)
	}
}

func (s *FileService) getFile(ctx context.Context, id string) (repository.File, error) {
	if _, err := uuid.Parse(id); err != nil {
		return repository.File{}, status.Error(codes.InvalidArgument, "file_id must be a uuid")
	}

	f, err := s.repo.GetById(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return repository.File{}, status.Error(codes.NotFound, "file not found")
	}
	if err != nil {
		return repository.File{}, internalErr("get file", err)
	}

	return f, nil
}

func internalErr(op string, err error) error {
	log.Printf("%s: %v", op, err)
	return status.Error(codes.Internal, "internal error")
}
