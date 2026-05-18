package struc

import (
	"slices"
	"strings"
)

type AttachmentType string

const (
	Image AttachmentType = "IMAGE"
	Video AttachmentType = "VIDEO"
	File  AttachmentType = "FILE"
	Link  AttachmentType = "LINK"
)

func getAttachmentTypes() []AttachmentType {
	return []AttachmentType{Image, Video, File, Link}
}

// Values provides list valid values for Enum.
func (AttachmentType) Values() (kinds []string) {
	for _, value := range getAttachmentTypes() {
		kinds = append(kinds, string(value))
	}
	return
}

// Value return the value of the Enum.
func (t AttachmentType) Value() string {
	return string(t)
}

func (t AttachmentType) IsImage() bool {
	return t == Image
}

func (t AttachmentType) IsVideo() bool {
	return t == Video
}

func (t AttachmentType) IsFile() bool {
	return t == File
}

func (t AttachmentType) IsLink() bool {
	return t == Link
}

func (t AttachmentType) IsValid() bool {
	return slices.Contains(getAttachmentTypes(), t)
}

func ExtensionToType(extension string) AttachmentType {
	switch strings.ToLower(extension) {
	case "jpg", "jpeg", "png", "gif", "webp", "bmp", "wbmp":
		return Image
	case "mp4", "avi", "mov", "webm", "3gp", "3g2", "mkv":
		return Video
	default:
		return File
	}
}
