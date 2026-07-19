package imageupload

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

const (
	MaxUploadBytes int64 = 2 << 20

	maxSourceWidth  = 6000
	maxSourceHeight = 6000
	maxSourcePixels = 20_000_000

	maxOutputWidth  = 1920
	maxOutputHeight = 1920

	jpegQuality = 85
)

var (
	ErrMissingImage = errors.New("no image was uploaded")

	ErrImageTooLarge = errors.New(
		"image exceeds the 2 MB limit",
	)

	ErrUnsupportedImageType = errors.New(
		"only JPEG and PNG images are allowed",
	)

	ErrInvalidImage = errors.New(
		"uploaded file is not a valid image",
	)

	ErrImageDimensions = errors.New(
		"image dimensions are not allowed",
	)

	ErrInvalidImageCategory = errors.New(
		"invalid image category",
	)
)

type SavedImage struct {
	URL      string
	FilePath string
	Width    int
	Height   int
}

// SaveOptional validates, normalizes, re-encodes, and stores an uploaded image.
// The original filename and original bytes are never persisted.
func SaveOptional(
	r *http.Request,
	fieldName string,
	category string,
) (SavedImage, bool, error) {
	file, header, err := r.FormFile(fieldName)
	if errors.Is(err, http.ErrMissingFile) {
		return SavedImage{}, false, nil
	}
	if err != nil {
		return SavedImage{}, false, fmt.Errorf(
			"read uploaded image: %w",
			err,
		)
	}
	defer file.Close()

	savedImage, err := Save(
		file,
		header,
		category,
	)
	if err != nil {
		return SavedImage{}, false, err
	}

	return savedImage, true, nil
}

// Save accepts a JPEG or PNG image, validates its real contents and decoded
// dimensions, applies JPEG orientation, scales it down, composites
// transparency onto white, and writes a new JPEG file.
func Save(
	file multipart.File,
	header *multipart.FileHeader,
	category string,
) (SavedImage, error) {
	directoryName, err := safeCategory(category)
	if err != nil {
		return SavedImage{}, err
	}

	if header != nil &&
		header.Size > MaxUploadBytes {
		return SavedImage{}, ErrImageTooLarge
	}

	imageBytes, err := readLimited(
		file,
		MaxUploadBytes,
	)
	if err != nil {
		return SavedImage{}, err
	}

	if err := validateContentType(imageBytes); err != nil {
		return SavedImage{}, err
	}

	config, format, err := image.DecodeConfig(
		bytes.NewReader(imageBytes),
	)
	if err != nil {
		return SavedImage{}, ErrInvalidImage
	}

	if format != "jpeg" && format != "png" {
		return SavedImage{},
			ErrUnsupportedImageType
	}

	if err := validateDimensions(
		config.Width,
		config.Height,
	); err != nil {
		return SavedImage{}, err
	}

	decodedImage, err := imaging.Decode(
		bytes.NewReader(imageBytes),
		imaging.AutoOrientation(true),
	)
	if err != nil {
		return SavedImage{}, ErrInvalidImage
	}

	normalizedImage := imaging.Fit(
		decodedImage,
		maxOutputWidth,
		maxOutputHeight,
		imaging.Lanczos,
	)

	// JPEG does not support transparency. Composite PNG transparency onto a
	// white background instead of allowing transparent pixels to turn black.
	background := imaging.New(
		normalizedImage.Bounds().Dx(),
		normalizedImage.Bounds().Dy(),
		color.NRGBA{
			R: 255,
			G: 255,
			B: 255,
			A: 255,
		},
	)

	normalizedImage = imaging.Paste(
		background,
		normalizedImage,
		image.Point{},
	)

	fileName, err := randomFileName()
	if err != nil {
		return SavedImage{}, fmt.Errorf(
			"generate image filename: %w",
			err,
		)
	}

	relativeDirectory := filepath.Join(
		"static",
		"uploads",
		directoryName,
	)

	if err := os.MkdirAll(
		relativeDirectory,
		0o750,
	); err != nil {
		return SavedImage{}, fmt.Errorf(
			"create upload directory: %w",
			err,
		)
	}

	finalPath := filepath.Join(
		relativeDirectory,
		fileName,
	)

	tempFile, err := os.CreateTemp(
		relativeDirectory,
		".image-*.tmp",
	)
	if err != nil {
		return SavedImage{}, fmt.Errorf(
			"create temporary image: %w",
			err,
		)
	}

	tempPath := tempFile.Name()
	committed := false

	defer func() {
		_ = tempFile.Close()

		if !committed {
			_ = os.Remove(tempPath)
		}
	}()

	if err := imaging.Encode(
		tempFile,
		normalizedImage,
		imaging.JPEG,
		imaging.JPEGQuality(jpegQuality),
	); err != nil {
		return SavedImage{}, fmt.Errorf(
			"re-encode image: %w",
			err,
		)
	}

	if err := tempFile.Sync(); err != nil {
		return SavedImage{}, fmt.Errorf(
			"sync image: %w",
			err,
		)
	}

	if err := tempFile.Close(); err != nil {
		return SavedImage{}, fmt.Errorf(
			"close image: %w",
			err,
		)
	}

	if err := os.Chmod(
		tempPath,
		0o640,
	); err != nil {
		return SavedImage{}, fmt.Errorf(
			"set image permissions: %w",
			err,
		)
	}

	if err := os.Rename(
		tempPath,
		finalPath,
	); err != nil {
		return SavedImage{}, fmt.Errorf(
			"commit image: %w",
			err,
		)
	}

	committed = true

	return SavedImage{
		URL: "/static/uploads/" +
			directoryName +
			"/" +
			fileName,
		FilePath: finalPath,
		Width:    normalizedImage.Bounds().Dx(),
		Height:   normalizedImage.Bounds().Dy(),
	}, nil
}

// Delete removes only application-managed upload URLs. External URLs and
// traversal attempts are ignored.
func Delete(imageURL string) error {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return nil
	}

	const uploadPrefix = "/static/uploads/"

	if !strings.HasPrefix(
		imageURL,
		uploadPrefix,
	) {
		return nil
	}

	relativePath := strings.TrimPrefix(
		imageURL,
		"/",
	)
	cleanPath := filepath.Clean(relativePath)

	uploadRoot := filepath.Clean(
		filepath.Join(
			"static",
			"uploads",
		),
	)

	if cleanPath == uploadRoot ||
		!strings.HasPrefix(
			cleanPath,
			uploadRoot+string(os.PathSeparator),
		) {
		return ErrInvalidImageCategory
	}

	err := os.Remove(cleanPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf(
			"delete uploaded image: %w",
			err,
		)
	}

	return nil
}

func readLimited(
	reader io.Reader,
	maximumBytes int64,
) ([]byte, error) {
	limitedReader := io.LimitReader(
		reader,
		maximumBytes+1,
	)

	imageBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf(
			"read image bytes: %w",
			err,
		)
	}

	if int64(len(imageBytes)) > maximumBytes {
		return nil, ErrImageTooLarge
	}

	if len(imageBytes) == 0 {
		return nil, ErrInvalidImage
	}

	return imageBytes, nil
}

func validateContentType(
	imageBytes []byte,
) error {
	sniffLength := len(imageBytes)
	if sniffLength > 512 {
		sniffLength = 512
	}

	detectedType := http.DetectContentType(
		imageBytes[:sniffLength],
	)

	switch detectedType {
	case "image/jpeg", "image/png":
		return nil

	default:
		return ErrUnsupportedImageType
	}
}

func validateDimensions(
	width int,
	height int,
) error {
	if width <= 0 ||
		height <= 0 ||
		width > maxSourceWidth ||
		height > maxSourceHeight ||
		int64(width)*int64(height) >
			maxSourcePixels {
		return ErrImageDimensions
	}

	return nil
}

func safeCategory(
	category string,
) (string, error) {
	switch strings.TrimSpace(category) {
	case "events":
		return "events", nil

	case "services":
		return "services", nil

	default:
		return "", ErrInvalidImageCategory
	}
}

func randomFileName() (string, error) {
	randomBytes := make([]byte, 16)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes) +
		".jpg", nil
}
