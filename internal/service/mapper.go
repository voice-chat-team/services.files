package service

import (
	"time"

	files_v1 "github.com/voice-chat-team/contracts/gen/go/files/v1"
	"github.com/voice-chat-team/services.files/internal/repository"
	"github.com/voice-chat-team/services.files/internal/storage"
)

func toProto(f repository.File, st *storage.Storage) *files_v1.File {
	pf := &files_v1.File{
		Id:          f.ID,
		OwnerId:     f.OwnerID,
		Purpose:     files_v1.FilePurpose(f.Purpose),
		Status:      files_v1.FileStatus(f.Status),
		FileName:    f.Name,
		ContentType: f.ContentType,
		SizeBytes:   f.SizeBytes,
		CreatedAt:   f.CreatedAt.Format(time.RFC3339),
	}
	if rule, ok := rules[pf.Purpose]; ok && rule.Public && f.Status == repository.StatusReady {
		pf.PublicUrl = st.PublicURL(f.S3Key)
	}
	return pf
}
