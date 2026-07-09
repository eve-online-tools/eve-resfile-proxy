package index

type Entry struct {
	LogicalPath    string `json:"logicalPath"`
	CDNPath        string `json:"cdnPath"`
	Checksum       string `json:"checksum,omitempty"`
	Size           int64  `json:"size,omitempty"`
	CompressedSize int64  `json:"compressedSize,omitempty"`
	Mode           string `json:"mode,omitempty"`
}
