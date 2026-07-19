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
	ErrMissingImage = errors.New(
		"no image was uploaded",
	)

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

// Save accepts a JPEG or PNG image, validates its real content and decoded
// dimensions, applies orientation, scales it down, removes transparency, and
// atomically stores a re-encoded JPEG.
func Save(
	file multipart.File,
	header *multipart.FileHeader,
	category string,
) (SavedImage, error) {
	directoryName, err :=
		validateUploadMetadata(
			header,
			category,
		)
	if err != nil {
		return SavedImage{}, err
	}

	imageBytes, err :=
		readAndValidateImage(file)
	if err != nil {
		return SavedImage{}, err
	}

	normalizedImage, err :=
		decodeAndNormalizeImage(imageBytes)
	if err != nil {
		return SavedImage{}, err
	}

	return persistImage(
		normalizedImage,
		directoryName,
	)
}

func validateUploadMetadata(
	header *multipart.FileHeader,
	category string,
) (string, error) {
	directoryName, err := safeCategory(
		category,
	)
	if err != nil {
		return "", err
	}

	if header != nil &&
		header.Size > MaxUploadBytes {
		return "", ErrImageTooLarge
	}

	return directoryName, nil
}

func readAndValidateImage(
	file multipart.File,
) ([]byte, error) {
	imageBytes, err := readLimited(
		file,
		MaxUploadBytes,
	)
	if err != nil {
		return nil, err
	}

	if err := validateContentType(
		imageBytes,
	); err != nil {
		return nil, err
	}

	return imageBytes, nil
}

func decodeAndNormalizeImage(
	imageBytes []byte,
) (image.Image, error) {
	config, format, err := image.DecodeConfig(
		bytes.NewReader(imageBytes),
	)
	if err != nil {
		return nil, ErrInvalidImage
	}

	if err := validateImageFormat(
		format,
	); err != nil {
		return nil, err
	}

	if err := validateDimensions(
		config.Width,
		config.Height,
	); err != nil {
		return nil, err
	}

	decodedImage, err := imaging.Decode(
		bytes.NewReader(imageBytes),
		imaging.AutoOrientation(true),
	)
	if err != nil {
		return nil, ErrInvalidImage
	}

	resizedImage := imaging.Fit(
		decodedImage,
		maxOutputWidth,
		maxOutputHeight,
		imaging.Lanczos,
	)

	return compositeOnWhite(resizedImage), nil
}

func validateImageFormat(
	format string,
) error {
	switch format {
	case "jpeg", "png":
		return nil

	default:
		return ErrUnsupportedImageType
	}
}

func compositeOnWhite(
	sourceImage image.Image,
) image.Image {
	background := imaging.New(
		sourceImage.Bounds().Dx(),
		sourceImage.Bounds().Dy(),
		color.NRGBA{
			R: 255,
			G: 255,
			B: 255,
			A: 255,
		},
	)

	return imaging.Paste(
		background,
		sourceImage,
		image.Point{},
	)
}

func persistImage(
	normalizedImage image.Image,
	directoryName string,
) (SavedImage, error) {
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

	if err := createUploadDirectory(
		relativeDirectory,
	); err != nil {
		return SavedImage{}, err
	}

	finalPath := filepath.Join(
		relativeDirectory,
		fileName,
	)

	if err := writeJPEGAtomically(
		relativeDirectory,
		finalPath,
		normalizedImage,
	); err != nil {
		return SavedImage{}, err
	}

	return SavedImage{
		URL: "/static/uploads/" +
			directoryName +
			"/" +
			fileName,
		FilePath: finalPath,
		Width: normalizedImage.
			Bounds().
			Dx(),
		Height: normalizedImage.
			Bounds().
			Dy(),
	}, nil
}

func createUploadDirectory(
	directory string,
) error {
	if err := os.MkdirAll(
		directory,
		0o750,
	); err != nil {
		return fmt.Errorf(
			"create upload directory: %w",
			err,
		)
	}

	return nil
}

func writeJPEGAtomically(
	directory string,
	finalPath string,
	normalizedImage image.Image,
) error {
	tempFile, err := os.CreateTemp(
		directory,
		".image-*.tmp",
	)
	if err != nil {
		return fmt.Errorf(
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

	if err := encodeJPEG(
		tempFile,
		normalizedImage,
	); err != nil {
		return err
	}

	if err := finalizeTemporaryFile(
		tempFile,
		tempPath,
	); err != nil {
		return err
	}

	if err := os.Rename(
		tempPath,
		finalPath,
	); err != nil {
		return fmt.Errorf(
			"commit image: %w",
			err,
		)
	}

	committed = true

	return nil
}

func encodeJPEG(
	destination io.Writer,
	normalizedImage image.Image,
) error {
	if err := imaging.Encode(
		destination,
		normalizedImage,
		imaging.JPEG,
		imaging.JPEGQuality(jpegQuality),
	); err != nil {
		return fmt.Errorf(
			"re-encode image: %w",
			err,
		)
	}

	return nil
}

func finalizeTemporaryFile(
	tempFile *os.File,
	tempPath string,
) error {
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf(
			"sync image: %w",
			err,
		)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf(
			"close image: %w",
			err,
		)
	}

	if err := os.Chmod(
		tempPath,
		0o640,
	); err != nil {
		return fmt.Errorf(
			"set image permissions: %w",
			err,
		)
	}

	return nil
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

	imageBytes, err := io.ReadAll(
		limitedReader,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"read image bytes: %w",
			err,
		)
	}

	if int64(len(imageBytes)) >
		maximumBytes {
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

	case "flyers":
		return "flyers", nil

	default:
		return "", ErrInvalidImageCategory
	}
}

func randomFileName() (string, error) {
	randomBytes := make([]byte, 16)

	if _, err := rand.Read(
		randomBytes,
	); err != nil {
		return "", err
	}

	return hex.EncodeToString(
		randomBytes,
	) + ".jpg", nil
}
