package dto

// UploadFileResponse represents the standard response payload when a file is uploaded.
type UploadFileResponse struct {
	FileKey      string `json:"file_key"`
	FileURL      string `json:"file_url"`
	FileName     string `json:"file_name"`
	OriginalName string `json:"original_name"`
	FileSize     int64  `json:"file_size"`
	ContentType  string `json:"content_type"`
}
