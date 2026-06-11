// Package drive wraps the Google Drive API v3.
package drive

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Service wraps the Google Drive API client.
type Service struct {
	svc *drive.Service
}

// FileSummary is a lightweight file representation.
type FileSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MimeType    string `json:"mime_type"`
	Size        int64  `json:"size,omitempty"`
	Created     string `json:"created,omitempty"`
	Modified    string `json:"modified,omitempty"`
	Parents     string `json:"parents,omitempty"`
	WebViewLink string `json:"web_view_link,omitempty"`
}

// New creates a Service from an OAuth2 token source.
func New(ctx context.Context, ts oauth2.TokenSource) (*Service, error) {
	svc, err := drive.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}
	return &Service{svc: svc}, nil
}

// ListFiles lists files in the root or a specific folder.
func (s *Service) ListFiles(ctx context.Context, folderID, query string, pageSize int64) ([]*FileSummary, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50
	}

	q := fmt.Sprintf("trashed=false")
	if folderID != "" {
		q += fmt.Sprintf(" and '%s' in parents", folderID)
	} else {
		q += " and 'root' in parents"
	}
	if query != "" {
		q += fmt.Sprintf(" and name contains '%s'", query)
	}

	files, err := s.svc.Files.List().
		Q(q).
		PageSize(pageSize).
		Fields("files(id,name,mimeType,size,createdTime,modifiedTime,parents,webViewLink)").
		OrderBy("modifiedTime desc").
		Do()
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	summaries := make([]*FileSummary, 0, len(files.Files))
	for _, f := range files.Files {
		s := &FileSummary{
			ID:          f.Id,
			Name:        f.Name,
			MimeType:    f.MimeType,
			Size:        f.Size,
			Created:     fmtTime(f.CreatedTime),
			Modified:    fmtTime(f.ModifiedTime),
			WebViewLink: f.WebViewLink,
		}
		if len(f.Parents) > 0 {
			s.Parents = f.Parents[0]
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// SearchDrive searches files across the entire Drive by query.
func (s *Service) SearchDrive(ctx context.Context, query string, pageSize int64) ([]*FileSummary, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50
	}

	q := fmt.Sprintf("trashed=false and (name contains '%s' or fullText contains '%s')", query, query)
	files, err := s.svc.Files.List().
		Q(q).
		PageSize(pageSize).
		Fields("files(id,name,mimeType,size,createdTime,modifiedTime,webViewLink)").
		OrderBy("modifiedTime desc").
		Do()
	if err != nil {
		return nil, fmt.Errorf("search drive: %w", err)
	}

	summaries := make([]*FileSummary, 0, len(files.Files))
	for _, f := range files.Files {
		summaries = append(summaries, &FileSummary{
			ID:          f.Id,
			Name:        f.Name,
			MimeType:    f.MimeType,
			Size:        f.Size,
			Created:     fmtTime(f.CreatedTime),
			Modified:    fmtTime(f.ModifiedTime),
			WebViewLink: f.WebViewLink,
		})
	}
	return summaries, nil
}

// GetFile retrieves metadata for a single file.
func (s *Service) GetFile(ctx context.Context, fileID string) (*FileSummary, error) {
	f, err := s.svc.Files.Get(fileID).
		Fields("id,name,mimeType,size,createdTime,modifiedTime,parents,webViewLink,description").
		Do()
	if err != nil {
		return nil, fmt.Errorf("get file: %w", err)
	}

	return &FileSummary{
		ID:          f.Id,
		Name:        f.Name,
		MimeType:    f.MimeType,
		Size:        f.Size,
		Created:     fmtTime(f.CreatedTime),
		Modified:    fmtTime(f.ModifiedTime),
		WebViewLink: f.WebViewLink,
		Parents:     fmtParents(f.Parents),
	}, nil
}

// UploadFile uploads a local file to Drive. Returns the created file metadata.
func (s *Service) UploadFile(ctx context.Context, localPath, parentFolderID, mimeType string) (*FileSummary, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	name := filepath.Base(localPath)

	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(name))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}

	driveFile := &drive.File{Name: name}
	if parentFolderID != "" {
		driveFile.Parents = []string{parentFolderID}
	}

	created, err := s.svc.Files.Create(driveFile).
		Media(f).
		Fields("id,name,mimeType,size,createdTime,webViewLink").
		Do()
	if err != nil {
		return nil, fmt.Errorf("upload file: %w", err)
	}

	return &FileSummary{
		ID:          created.Id,
		Name:        created.Name,
		MimeType:    created.MimeType,
		Size:        created.Size,
		Created:     fmtTime(created.CreatedTime),
		WebViewLink: created.WebViewLink,
	}, nil
}

// DownloadFile downloads a file from Drive and saves it locally.
func (s *Service) DownloadFile(ctx context.Context, fileID, destPath string) error {
	resp, err := s.svc.Files.Get(fileID).Download()
	if err != nil {
		return fmt.Errorf("download get: %w", err)
	}
	defer resp.Body.Close()

	// Create parent dirs
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	if written == 0 {
		return fmt.Errorf("downloaded 0 bytes")
	}
	return nil
}

// CreateFolder creates a new folder in Drive.
func (s *Service) CreateFolder(ctx context.Context, name, parentFolderID string) (*FileSummary, error) {
	folder := &drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
	}
	if parentFolderID != "" {
		folder.Parents = []string{parentFolderID}
	}

	created, err := s.svc.Files.Create(folder).
		Fields("id,name,mimeType,createdTime,webViewLink").
		Do()
	if err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}

	return &FileSummary{
		ID:          created.Id,
		Name:        created.Name,
		MimeType:    created.MimeType,
		Created:     fmtTime(created.CreatedTime),
		WebViewLink: created.WebViewLink,
	}, nil
}

// DeleteFile moves a file to trash.
func (s *Service) DeleteFile(ctx context.Context, fileID string) error {
	return s.svc.Files.Delete(fileID).Do()
}

func fmtTime(t string) string {
	if t == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return t
	}
	return parsed.Format("Jan 2 15:04")
}

func fmtParents(parents []string) string {
	if len(parents) == 0 {
		return ""
	}
	return parents[0]
}
