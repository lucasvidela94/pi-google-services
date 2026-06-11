package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sombi/pi-google-services/internal/drive"
	"github.com/sombi/pi-google-services/internal/mcp"
)

// DriveService implements the Service interface for Google Drive.
type DriveService struct {
	api *drive.Service
}

// NewDrive creates a DriveService from the Drive API wrapper.
func NewDrive(api *drive.Service) *DriveService {
	return &DriveService{api: api}
}

func (s *DriveService) Name() string { return "drive" }

func (s *DriveService) Scopes() []string {
	return []string{
		"https://www.googleapis.com/auth/drive.file",
		"https://www.googleapis.com/auth/drive.readonly",
	}
}

func (s *DriveService) Tools() []mcp.ToolDefinition {
	return []mcp.ToolDefinition{
		{
			Name:        "list-files",
			Description: "List files in root or a specific folder",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"folderId": {Type: "string", Description: "Folder ID (default: root)"},
					"query":    {Type: "string", Description: "Filter by name"},
					"limit":    {Type: "number", Description: "Max results (default: 50, max: 100)", Default: 50},
				},
			},
		},
		{
			Name:        "search-drive",
			Description: "Search files across Drive by name or content",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"query": {Type: "string", Description: "Search text"},
					"limit": {Type: "number", Description: "Max results (default: 50, max: 100)", Default: 50},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "upload-file",
			Description: "Upload a local file to Drive",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"localPath":      {Type: "string", Description: "Local file path to upload"},
					"parentFolderId": {Type: "string", Description: "Destination folder ID (default: root)"},
				},
				Required: []string{"localPath"},
			},
		},
		{
			Name:        "download-file",
			Description: "Download a file from Drive to local disk",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"fileId":  {Type: "string", Description: "Drive file ID to download"},
					"destDir": {Type: "string", Description: "Local destination directory (default: current dir)"},
				},
				Required: []string{"fileId"},
			},
		},
		{
			Name:        "create-folder",
			Description: "Create a new folder in Drive",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"name":           {Type: "string", Description: "Folder name"},
					"parentFolderId": {Type: "string", Description: "Parent folder ID (default: root)"},
				},
				Required: []string{"name"},
			},
		},
		{
			Name:        "delete-file",
			Description: "Delete/trash a file from Drive",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"fileId": {Type: "string", Description: "File ID to delete"},
				},
				Required: []string{"fileId"},
			},
		},
	}
}

func (s *DriveService) Handle(ctx context.Context, toolName string, params json.RawMessage) (interface{}, *mcp.RPCError) {
	switch toolName {
	case "list-files":
		return s.handleListFiles(ctx, params)
	case "search-drive":
		return s.handleSearchDrive(ctx, params)
	case "upload-file":
		return s.handleUploadFile(ctx, params)
	case "download-file":
		return s.handleDownloadFile(ctx, params)
	case "create-folder":
		return s.handleCreateFolder(ctx, params)
	case "delete-file":
		return s.handleDeleteFile(ctx, params)
	default:
		return nil, &mcp.RPCError{Code: -32601, Message: fmt.Sprintf("Drive tool not found: %s", toolName)}
	}
}

func (s *DriveService) handleListFiles(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		FolderID string `json:"folderId"`
		Query    string `json:"query"`
		Limit    int64  `json:"limit"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}

	files, err := s.api.ListFiles(ctx, args.FolderID, args.Query, args.Limit)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to list files", Data: err.Error()}
	}

	var b strings.Builder
	if len(files) == 0 {
		b.WriteString("No files found.")
	} else {
		for i, f := range files {
			icon := fileIcon(f.MimeType)
			b.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, icon, f.Name))
			b.WriteString(fmt.Sprintf("   📎 %s  🕐 %s", f.ID, f.Modified))
			if f.MimeType != "application/vnd.google-apps.folder" && f.Size > 0 {
				b.WriteString(fmt.Sprintf("  📦 %s", fmtSize(f.Size)))
			}
			b.WriteString("\n")
		}
	}
	return contentResponse(b.String()), nil
}

func (s *DriveService) handleSearchDrive(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		Query string `json:"query"`
		Limit int64  `json:"limit"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.Query == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "query required"}
	}

	files, err := s.api.SearchDrive(ctx, args.Query, args.Limit)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to search", Data: err.Error()}
	}

	var b strings.Builder
	if len(files) == 0 {
		b.WriteString("No results.")
	} else {
		for i, f := range files {
			icon := fileIcon(f.MimeType)
			b.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, icon, f.Name))
			b.WriteString(fmt.Sprintf("   📎 %s  🕐 %s", f.ID, f.Modified))
			if f.Size > 0 {
				b.WriteString(fmt.Sprintf("  📦 %s", fmtSize(f.Size)))
			}
			if f.WebViewLink != "" {
				b.WriteString(fmt.Sprintf("\n   🔗 %s", f.WebViewLink))
			}
			b.WriteString("\n")
		}
	}
	return contentResponse(b.String()), nil
}

func (s *DriveService) handleUploadFile(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		LocalPath      string `json:"localPath"`
		ParentFolderID string `json:"parentFolderId"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.LocalPath == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "localPath required"}
	}

	created, err := s.api.UploadFile(ctx, args.LocalPath, args.ParentFolderID, "")
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to upload", Data: err.Error()}
	}

	return contentResponse(fmt.Sprintf("✅ Uploaded: %s\n   ID: %s\n   📦 %s\n   🔗 %s",
		created.Name, created.ID, fmtSize(created.Size), created.WebViewLink)), nil
}

func (s *DriveService) handleDownloadFile(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		FileID  string `json:"fileId"`
		DestDir string `json:"destDir"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.FileID == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "fileId required"}
	}
	if args.DestDir == "" {
		args.DestDir = "."
	}

	// Get file name first
	info, err := s.api.GetFile(ctx, args.FileID)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to get file info", Data: err.Error()}
	}

	destPath := args.DestDir + "/" + info.Name
	if err := s.api.DownloadFile(ctx, args.FileID, destPath); err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to download", Data: err.Error()}
	}

	return contentResponse(fmt.Sprintf("✅ Downloaded: %s → %s\n   📦 %s", info.Name, destPath, fmtSize(info.Size))), nil
}

func (s *DriveService) handleCreateFolder(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		Name           string `json:"name"`
		ParentFolderID string `json:"parentFolderId"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.Name == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "name required"}
	}

	folder, err := s.api.CreateFolder(ctx, args.Name, args.ParentFolderID)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to create folder", Data: err.Error()}
	}

	return contentResponse(fmt.Sprintf("✅ Folder created: %s\n   ID: %s", folder.Name, folder.ID)), nil
}

func (s *DriveService) handleDeleteFile(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		FileID string `json:"fileId"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.FileID == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "fileId required"}
	}

	if err := s.api.DeleteFile(ctx, args.FileID); err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to delete", Data: err.Error()}
	}

	return contentResponse(fmt.Sprintf("✅ File deleted (ID: %s)", args.FileID)), nil
}

// helpers

func fileIcon(mimeType string) string {
	switch {
	case mimeType == "application/vnd.google-apps.folder":
		return "📁"
	case strings.HasPrefix(mimeType, "image/"):
		return "🖼️"
	case strings.HasPrefix(mimeType, "video/"):
		return "🎬"
	case strings.HasPrefix(mimeType, "audio/"):
		return "🎵"
	case strings.Contains(mimeType, "pdf"):
		return "📄"
	case strings.Contains(mimeType, "spreadsheet") || strings.Contains(mimeType, "sheet"):
		return "📊"
	case strings.Contains(mimeType, "document") || strings.Contains(mimeType, "text"):
		return "📝"
	case strings.Contains(mimeType, "presentation") || strings.Contains(mimeType, "slide"):
		return "📽️"
	default:
		return "📎"
	}
}

func fmtSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}
