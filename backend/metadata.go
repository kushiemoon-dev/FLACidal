package backend

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

// FLACMetadata contains parsed metadata from a FLAC file
type FLACMetadata struct {
	Path         string `json:"path"`
	Title        string `json:"title"`
	Artist       string `json:"artist"`
	Album        string `json:"album"`
	TrackNumber  string `json:"trackNumber"`
	Date         string `json:"date"`
	Genre        string `json:"genre"`
	ISRC         string `json:"isrc"`
	Comment      string `json:"comment"`
	Size         int64  `json:"size"`
	Duration     int    `json:"duration"`     // seconds
	SampleRate   int    `json:"sampleRate"`   // Hz
	BitDepth     int    `json:"bitDepth"`     // bits per sample
	Channels     int    `json:"channels"`     // number of channels
	Bitrate      int    `json:"bitrate"`      // kbps (calculated)
	HasCover     bool   `json:"hasCover"`
	CoverMime    string `json:"coverMime,omitempty"`
	CoverSize    int    `json:"coverSize,omitempty"`
	TotalSamples uint64 `json:"totalSamples"`
	// Lyrics fields
	Lyrics       string `json:"lyrics,omitempty"`
	SyncedLyrics string `json:"syncedLyrics,omitempty"`
	HasLyrics    bool   `json:"hasLyrics"`
}

// ReadFLACMetadata reads and parses metadata from a FLAC file
func ReadFLACMetadata(filePath string) (*FLACMetadata, error) {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Verify FLAC signature
	if len(data) < 4 || string(data[:4]) != "fLaC" {
		return nil, fmt.Errorf("not a valid FLAC file")
	}

	meta := &FLACMetadata{
		Path: filePath,
		Size: fileInfo.Size(),
	}

	pos := 4

	// Parse metadata blocks
	for pos < len(data) {
		if pos+4 > len(data) {
			break
		}

		header := data[pos]
		isLast := (header & 0x80) != 0
		blockType := header & 0x7F
		blockSize := int(data[pos+1])<<16 | int(data[pos+2])<<8 | int(data[pos+3])

		if pos+4+blockSize > len(data) {
			break
		}

		blockData := data[pos+4 : pos+4+blockSize]

		switch blockType {
		case 0: // STREAMINFO
			parseStreamInfo(blockData, meta)
		case 4: // VORBIS_COMMENT
			parseVorbisComment(blockData, meta)
		case 6: // PICTURE
			parsePictureBlock(blockData, meta)
		}

		pos += 4 + blockSize

		if isLast {
			break
		}
	}

	// Calculate duration and bitrate
	if meta.SampleRate > 0 && meta.TotalSamples > 0 {
		meta.Duration = int(meta.TotalSamples / uint64(meta.SampleRate))
		if meta.Duration > 0 {
			meta.Bitrate = int((meta.Size * 8) / int64(meta.Duration) / 1000)
		}
	}

	return meta, nil
}

// parseStreamInfo parses the STREAMINFO block
func parseStreamInfo(data []byte, meta *FLACMetadata) {
	if len(data) < 34 {
		return
	}

	// Bytes 10-17 contain: sample rate (20 bits), channels (3 bits), bits per sample (5 bits), total samples (36 bits)
	// Sample rate: bits 80-99 (bytes 10-12, upper 20 bits)
	meta.SampleRate = int(data[10])<<12 | int(data[11])<<4 | int(data[12])>>4

	// Channels: bits 100-102 (lower 4 bits of byte 12, upper 3 bits) + 1
	meta.Channels = int((data[12]&0x0E)>>1) + 1

	// Bits per sample: bits 103-107 (lower 1 bit of byte 12, upper 4 bits of byte 13) + 1
	meta.BitDepth = int((data[12]&0x01)<<4|data[13]>>4) + 1

	// Total samples: bits 108-143 (lower 4 bits of byte 13, bytes 14-17)
	meta.TotalSamples = uint64(data[13]&0x0F)<<32 |
		uint64(data[14])<<24 |
		uint64(data[15])<<16 |
		uint64(data[16])<<8 |
		uint64(data[17])
}

// parseVorbisComment parses the VORBIS_COMMENT block
func parseVorbisComment(data []byte, meta *FLACMetadata) {
	if len(data) < 8 {
		return
	}

	pos := 0

	// Vendor string length (little-endian)
	vendorLen := int(binary.LittleEndian.Uint32(data[pos:]))
	pos += 4

	if pos+vendorLen > len(data) {
		return
	}
	pos += vendorLen // Skip vendor string

	if pos+4 > len(data) {
		return
	}

	// Number of comments
	commentCount := int(binary.LittleEndian.Uint32(data[pos:]))
	pos += 4

	// Parse each comment
	for i := 0; i < commentCount && pos+4 <= len(data); i++ {
		commentLen := int(binary.LittleEndian.Uint32(data[pos:]))
		pos += 4

		if pos+commentLen > len(data) {
			break
		}

		comment := string(data[pos : pos+commentLen])
		pos += commentLen

		// Parse key=value
		parts := strings.SplitN(comment, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToUpper(parts[0])
		value := parts[1]

		switch key {
		case "TITLE":
			meta.Title = value
		case "ARTIST":
			meta.Artist = value
		case "ALBUM":
			meta.Album = value
		case "TRACKNUMBER":
			meta.TrackNumber = value
		case "DATE":
			meta.Date = value
		case "GENRE":
			meta.Genre = value
		case "ISRC":
			meta.ISRC = value
		case "COMMENT", "DESCRIPTION":
			meta.Comment = value
		case "LYRICS", "UNSYNCEDLYRICS":
			meta.Lyrics = value
			meta.HasLyrics = true
		case "SYNCEDLYRICS":
			meta.SyncedLyrics = value
			meta.HasLyrics = true
		}
	}
}

// parsePictureBlock parses the PICTURE block to get cover info
func parsePictureBlock(data []byte, meta *FLACMetadata) {
	if len(data) < 32 {
		return
	}

	pos := 0

	// Picture type (4 bytes, big-endian)
	pos += 4

	// MIME type length
	mimeLen := int(binary.BigEndian.Uint32(data[pos:]))
	pos += 4

	if pos+mimeLen > len(data) {
		return
	}

	meta.CoverMime = string(data[pos : pos+mimeLen])
	pos += mimeLen

	// Description length
	if pos+4 > len(data) {
		return
	}
	descLen := int(binary.BigEndian.Uint32(data[pos:]))
	pos += 4 + descLen

	// Width, height, depth, colors (4 bytes each)
	if pos+16 > len(data) {
		return
	}
	pos += 16

	// Picture data length
	if pos+4 > len(data) {
		return
	}
	pictureDataLen := int(binary.BigEndian.Uint32(data[pos:]))

	meta.HasCover = true
	meta.CoverSize = pictureDataLen
}

// GetCoverArt extracts cover art from a FLAC file
func GetCoverArt(filePath string) ([]byte, string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	if len(data) < 4 || string(data[:4]) != "fLaC" {
		return nil, "", fmt.Errorf("not a valid FLAC file")
	}

	pos := 4

	for pos < len(data) {
		if pos+4 > len(data) {
			break
		}

		header := data[pos]
		isLast := (header & 0x80) != 0
		blockType := header & 0x7F
		blockSize := int(data[pos+1])<<16 | int(data[pos+2])<<8 | int(data[pos+3])

		if pos+4+blockSize > len(data) {
			break
		}

		blockData := data[pos+4 : pos+4+blockSize]

		if blockType == 6 { // PICTURE
			return extractPictureData(blockData)
		}

		pos += 4 + blockSize

		if isLast {
			break
		}
	}

	return nil, "", fmt.Errorf("no cover art found")
}

// extractPictureData extracts the actual image data from a PICTURE block
func extractPictureData(data []byte) ([]byte, string, error) {
	if len(data) < 32 {
		return nil, "", fmt.Errorf("invalid picture block")
	}

	pos := 4 // Skip picture type

	// MIME type
	mimeLen := int(binary.BigEndian.Uint32(data[pos:]))
	pos += 4

	if pos+mimeLen > len(data) {
		return nil, "", fmt.Errorf("invalid MIME length")
	}

	mimeType := string(data[pos : pos+mimeLen])
	pos += mimeLen

	// Description
	descLen := int(binary.BigEndian.Uint32(data[pos:]))
	pos += 4 + descLen

	// Skip width, height, depth, colors
	pos += 16

	// Picture data
	pictureLen := int(binary.BigEndian.Uint32(data[pos:]))
	pos += 4

	if pos+pictureLen > len(data) {
		return nil, "", fmt.Errorf("invalid picture data length")
	}

	return data[pos : pos+pictureLen], mimeType, nil
}

// FormatMetadataForDisplay returns formatted metadata as key-value pairs
func FormatMetadataForDisplay(meta *FLACMetadata) map[string]string {
	result := make(map[string]string)

	if meta.Title != "" {
		result["Title"] = meta.Title
	}
	if meta.Artist != "" {
		result["Artist"] = meta.Artist
	}
	if meta.Album != "" {
		result["Album"] = meta.Album
	}
	if meta.TrackNumber != "" {
		result["Track"] = meta.TrackNumber
	}
	if meta.Date != "" {
		result["Date"] = meta.Date
	}
	if meta.Genre != "" {
		result["Genre"] = meta.Genre
	}
	if meta.ISRC != "" {
		result["ISRC"] = meta.ISRC
	}

	// Audio info
	result["Sample Rate"] = fmt.Sprintf("%d Hz", meta.SampleRate)
	result["Bit Depth"] = fmt.Sprintf("%d bit", meta.BitDepth)
	result["Channels"] = fmt.Sprintf("%d", meta.Channels)
	if meta.Bitrate > 0 {
		result["Bitrate"] = fmt.Sprintf("%d kbps", meta.Bitrate)
	}
	if meta.Duration > 0 {
		mins := meta.Duration / 60
		secs := meta.Duration % 60
		result["Duration"] = fmt.Sprintf("%d:%02d", mins, secs)
	}

	// File info
	result["Size"] = formatBytes(meta.Size)

	if meta.HasCover {
		result["Cover Art"] = fmt.Sprintf("Yes (%s, %s)", meta.CoverMime, formatBytes(int64(meta.CoverSize)))
	} else {
		result["Cover Art"] = "No"
	}

	return result
}

// formatBytes formats bytes to human-readable string
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// GetCoverArtBase64 returns cover art as base64 encoded string
func GetCoverArtBase64(filePath string) (string, string, error) {
	imageData, mimeType, err := GetCoverArt(filePath)
	if err != nil {
		return "", "", err
	}

	// Base64 encode
	encoded := bytes.Buffer{}
	encoder := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	for i := 0; i < len(imageData); i += 3 {
		var b uint32
		remaining := len(imageData) - i

		if remaining >= 3 {
			b = uint32(imageData[i])<<16 | uint32(imageData[i+1])<<8 | uint32(imageData[i+2])
			encoded.WriteByte(encoder[b>>18&0x3F])
			encoded.WriteByte(encoder[b>>12&0x3F])
			encoded.WriteByte(encoder[b>>6&0x3F])
			encoded.WriteByte(encoder[b&0x3F])
		} else if remaining == 2 {
			b = uint32(imageData[i])<<16 | uint32(imageData[i+1])<<8
			encoded.WriteByte(encoder[b>>18&0x3F])
			encoded.WriteByte(encoder[b>>12&0x3F])
			encoded.WriteByte(encoder[b>>6&0x3F])
			encoded.WriteByte('=')
		} else {
			b = uint32(imageData[i]) << 16
			encoded.WriteByte(encoder[b>>18&0x3F])
			encoded.WriteByte(encoder[b>>12&0x3F])
			encoded.WriteByte('=')
			encoded.WriteByte('=')
		}
	}

	return encoded.String(), mimeType, nil
}
