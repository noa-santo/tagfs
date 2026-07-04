package shared

type InboxEntry struct {
	Name       string `json:"name"`
	IsDir      bool   `json:"is_dir"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
	MimeType   string `json:"mime_type"`
}
