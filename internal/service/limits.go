package service

import (
	"strings"

	"github.com/google/uuid"
	files_v1 "github.com/voice-chat-team/contracts/gen/go/files/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type purposeRule struct {
	Dir       string
	MaxSize   int64
	ImageOnly bool
	Public    bool
}

var rules = map[files_v1.FilePurpose]purposeRule{
	files_v1.FilePurpose_FILE_PURPOSE_MESSAGE_ATTACHMENT: {
		Dir:     "attachments",
		MaxSize: 25 << 20,
	},
	files_v1.FilePurpose_FILE_PURPOSE_GUILD_ICON: {
		Dir:       "guild-icons",
		MaxSize:   5 << 20,
		ImageOnly: true,
		Public:    true,
	},
	files_v1.FilePurpose_FILE_PURPOSE_USER_AVATAR: {
		Dir:       "avatars",
		MaxSize:   5 << 20,
		ImageOnly: true,
		Public:    true,
	},
}

func validateUpload(req *files_v1.CreateUploadRequest) (purposeRule, error) {
	if _, err := uuid.Parse(req.UserId); err != nil {
		return purposeRule{}, status.Error(codes.InvalidArgument, "user_id must be a uuid")
	}

	rule, ok := rules[req.Purpose]
	if !ok {
		return purposeRule{}, status.Error(codes.InvalidArgument, "unknown purpose")
	}
	if req.FileName == "" {
		return purposeRule{}, status.Error(codes.InvalidArgument, "file_name is required")
	}
	if req.SizeBytes <= 0 || req.SizeBytes > rule.MaxSize {
		return purposeRule{}, status.Errorf(codes.InvalidArgument, "size must be 1..%d bytes", rule.MaxSize)
	}
	if rule.ImageOnly && !strings.HasPrefix(req.ContentType, "image/") {
		return purposeRule{}, status.Error(codes.InvalidArgument, "only images are allowed")
	}

	return rule, nil
}
