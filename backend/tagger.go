package backend

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FLACTagger handles metadata tagging for FLAC files
type FLACTagger struct {
	client *http.Client
}

// TrackMetadata contains metadata to embed in FLAC file
type TrackMetadata struct {
	Title       string
	Artist      string
	Album       string
	TrackNumber int
	TotalTracks int
	Year        string
	Genre       string
	ISRC        string
	CoverURL    string
	// Lyrics fields
	Lyrics       string // Plain text lyrics (LYRICS tag)
	SyncedLyrics string // LRC format synced lyrics (SYNCEDLYRICS tag)
}

// NewFLACTagger creates a new FLAC tagger
func NewFLACTagger() *FLACTagger {
	return &FLACTagger{
		client: &http.Client{},
	}
}

// TagFile applies metadata to a FLAC file
func (t *FLACTagger) TagFile(filePath string, meta TrackMetadata) error {
	// Read the original file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Verify FLAC signature
	if len(data) < 4 || string(data[:4]) != "fLaC" {
		return fmt.Errorf("not a valid FLAC file")
	}

	// Parse and rebuild with new metadata
	newData, err := t.rebuildWithMetadata(data, meta)
	if err != nil {
		return fmt.Errorf("failed to rebuild FLAC: %w", err)
	}

	// Write back
	if err := os.WriteFile(filePath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// rebuildWithMetadata rebuilds FLAC file with new Vorbis comments and picture
func (t *FLACTagger) rebuildWithMetadata(data []byte, meta TrackMetadata) ([]byte, error) {
	var result bytes.Buffer

	// Write FLAC signature
	result.Write(data[:4])

	pos := 4
	var streamInfoBlock []byte
	var audioData []byte

	// Parse existing metadata blocks
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

		// Keep STREAMINFO (type 0), skip old VORBIS_COMMENT (4) and PICTURE (6)
		if blockType == 0 {
			streamInfoBlock = blockData
		}

		pos += 4 + blockSize

		if isLast {
			audioData = data[pos:]
			break
		}
	}

	if streamInfoBlock == nil {
		return nil, fmt.Errorf("STREAMINFO block not found")
	}

	// Write STREAMINFO block (not last)
	result.WriteByte(0x00) // Type 0, not last
	writeBlockSize(&result, len(streamInfoBlock))
	result.Write(streamInfoBlock)

	// Create and write Vorbis comment block
	vorbisComment := t.createVorbisComment(meta)
	result.WriteByte(0x04) // Type 4 (VORBIS_COMMENT), not last
	writeBlockSize(&result, len(vorbisComment))
	result.Write(vorbisComment)

	// Download and write picture block if cover URL provided
	if meta.CoverURL != "" {
		pictureBlock, err := t.createPictureBlock(meta.CoverURL)
		if err == nil && len(pictureBlock) > 0 {
			result.WriteByte(0x86) // Type 6 (PICTURE), last block
			writeBlockSize(&result, len(pictureBlock))
			result.Write(pictureBlock)
		} else {
			// No picture, mark vorbis comment as last
			// Need to go back and fix the header - simpler to just add padding as last
			result.WriteByte(0x81) // Type 1 (PADDING), last block
			writeBlockSize(&result, 0)
		}
	} else {
		// No picture, add padding as last block
		result.WriteByte(0x81) // Type 1 (PADDING), last block
		writeBlockSize(&result, 0)
	}

	// Write audio data
	result.Write(audioData)

	return result.Bytes(), nil
}

// createVorbisComment creates a Vorbis comment block
func (t *FLACTagger) createVorbisComment(meta TrackMetadata) []byte {
	var buf bytes.Buffer

	// Vendor string
	vendor := "TidalFLACDownloader"
	binary.Write(&buf, binary.LittleEndian, uint32(len(vendor)))
	buf.WriteString(vendor)

	// Build comments
	var comments []string
	if meta.Title != "" {
		comments = append(comments, fmt.Sprintf("TITLE=%s", meta.Title))
	}
	if meta.Artist != "" {
		comments = append(comments, fmt.Sprintf("ARTIST=%s", meta.Artist))
	}
	if meta.Album != "" {
		comments = append(comments, fmt.Sprintf("ALBUM=%s", meta.Album))
	}
	if meta.TrackNumber > 0 {
		if meta.TotalTracks > 0 {
			comments = append(comments, fmt.Sprintf("TRACKNUMBER=%d/%d", meta.TrackNumber, meta.TotalTracks))
		} else {
			comments = append(comments, fmt.Sprintf("TRACKNUMBER=%d", meta.TrackNumber))
		}
	}
	if meta.Year != "" {
		comments = append(comments, fmt.Sprintf("DATE=%s", meta.Year))
	}
	if meta.Genre != "" {
		comments = append(comments, fmt.Sprintf("GENRE=%s", meta.Genre))
	}
	if meta.ISRC != "" {
		comments = append(comments, fmt.Sprintf("ISRC=%s", meta.ISRC))
	}
	// Add lyrics tags
	if meta.Lyrics != "" {
		comments = append(comments, fmt.Sprintf("LYRICS=%s", meta.Lyrics))
	}
	if meta.SyncedLyrics != "" {
		comments = append(comments, fmt.Sprintf("SYNCEDLYRICS=%s", meta.SyncedLyrics))
	}

	// Write comment count
	binary.Write(&buf, binary.LittleEndian, uint32(len(comments)))

	// Write each comment
	for _, comment := range comments {
		binary.Write(&buf, binary.LittleEndian, uint32(len(comment)))
		buf.WriteString(comment)
	}

	return buf.Bytes()
}

// createPictureBlock creates a PICTURE metadata block
func (t *FLACTagger) createPictureBlock(coverURL string) ([]byte, error) {
	// Download cover image
	resp, err := t.client.Get(coverURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to download cover: %d", resp.StatusCode)
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Detect MIME type
	mimeType := "image/jpeg"
	if len(imageData) > 8 {
		// PNG signature
		if imageData[0] == 0x89 && imageData[1] == 'P' && imageData[2] == 'N' && imageData[3] == 'G' {
			mimeType = "image/png"
		}
	}

	var buf bytes.Buffer

	// Picture type: 3 = Front cover
	binary.Write(&buf, binary.BigEndian, uint32(3))

	// MIME type
	binary.Write(&buf, binary.BigEndian, uint32(len(mimeType)))
	buf.WriteString(mimeType)

	// Description (empty)
	binary.Write(&buf, binary.BigEndian, uint32(0))

	// Width, height, depth, colors (0 = unknown)
	binary.Write(&buf, binary.BigEndian, uint32(0)) // width
	binary.Write(&buf, binary.BigEndian, uint32(0)) // height
	binary.Write(&buf, binary.BigEndian, uint32(0)) // depth
	binary.Write(&buf, binary.BigEndian, uint32(0)) // colors

	// Picture data
	binary.Write(&buf, binary.BigEndian, uint32(len(imageData)))
	buf.Write(imageData)

	return buf.Bytes(), nil
}

// writeBlockSize writes a 3-byte big-endian size
func writeBlockSize(w *bytes.Buffer, size int) {
	w.WriteByte(byte((size >> 16) & 0xFF))
	w.WriteByte(byte((size >> 8) & 0xFF))
	w.WriteByte(byte(size & 0xFF))
}

// EmbedLyrics embeds lyrics into an existing FLAC file
func (t *FLACTagger) EmbedLyrics(filePath string, lyrics, syncedLyrics string) error {
	// Read existing metadata
	meta, err := ReadFLACMetadata(filePath)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	// Create TrackMetadata with existing info + lyrics
	trackMeta := TrackMetadata{
		Title:        meta.Title,
		Artist:       meta.Artist,
		Album:        meta.Album,
		Year:         meta.Date,
		Genre:        meta.Genre,
		ISRC:         meta.ISRC,
		Lyrics:       lyrics,
		SyncedLyrics: syncedLyrics,
	}

	// Parse track number
	if meta.TrackNumber != "" {
		fmt.Sscanf(meta.TrackNumber, "%d", &trackMeta.TrackNumber)
	}

	// Read the original file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Verify FLAC signature
	if len(data) < 4 || string(data[:4]) != "fLaC" {
		return fmt.Errorf("not a valid FLAC file")
	}

	// Rebuild with lyrics (preserving cover art)
	newData, err := t.rebuildWithLyrics(data, trackMeta)
	if err != nil {
		return fmt.Errorf("failed to rebuild FLAC: %w", err)
	}

	// Write back
	if err := os.WriteFile(filePath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// rebuildWithLyrics rebuilds FLAC file preserving existing metadata but adding/updating lyrics
func (t *FLACTagger) rebuildWithLyrics(data []byte, meta TrackMetadata) ([]byte, error) {
	var result bytes.Buffer

	// Write FLAC signature
	result.Write(data[:4])

	pos := 4
	var streamInfoBlock []byte
	var existingPicture []byte
	var audioData []byte

	// Parse existing metadata blocks
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

		// Keep STREAMINFO (type 0) and PICTURE (type 6)
		if blockType == 0 {
			streamInfoBlock = blockData
		} else if blockType == 6 {
			existingPicture = blockData
		}

		pos += 4 + blockSize

		if isLast {
			audioData = data[pos:]
			break
		}
	}

	if streamInfoBlock == nil {
		return nil, fmt.Errorf("STREAMINFO block not found")
	}

	// Write STREAMINFO block (not last)
	result.WriteByte(0x00) // Type 0, not last
	writeBlockSize(&result, len(streamInfoBlock))
	result.Write(streamInfoBlock)

	// Create and write Vorbis comment block
	vorbisComment := t.createVorbisComment(meta)
	result.WriteByte(0x04) // Type 4 (VORBIS_COMMENT), not last
	writeBlockSize(&result, len(vorbisComment))
	result.Write(vorbisComment)

	// Write picture block if exists
	if len(existingPicture) > 0 {
		result.WriteByte(0x86) // Type 6 (PICTURE), last block
		writeBlockSize(&result, len(existingPicture))
		result.Write(existingPicture)
	} else {
		// No picture, add padding as last block
		result.WriteByte(0x81) // Type 1 (PADDING), last block
		writeBlockSize(&result, 0)
	}

	// Write audio data
	result.Write(audioData)

	return result.Bytes(), nil
}

// DownloadCover downloads cover art to a file
func (t *FLACTagger) DownloadCover(coverURL, outputPath string) error {
	resp, err := t.client.Get(coverURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download cover: %d", resp.StatusCode)
	}

	// Determine extension from URL or content type
	ext := ".jpg"
	if strings.Contains(coverURL, ".png") || strings.Contains(resp.Header.Get("Content-Type"), "png") {
		ext = ".png"
	}

	// Ensure directory exists
	os.MkdirAll(filepath.Dir(outputPath), 0755)

	// Add extension if not present
	if !strings.HasSuffix(outputPath, ext) && !strings.HasSuffix(outputPath, ".jpg") && !strings.HasSuffix(outputPath, ".png") {
		outputPath += ext
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}
