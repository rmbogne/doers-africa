package imageupload

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveReencodesPNGAsJPEG(t *testing.T) {
	workingDirectory := t.TempDir()

	originalDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(workingDirectory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(
		func() {
			_ = os.Chdir(originalDirectory)
		},
	)

	sourceImage := image.NewNRGBA(
		image.Rect(0, 0, 32, 24),
	)
	sourceImage.Set(
		0,
		0,
		color.NRGBA{
			R: 255,
			A: 120,
		},
	)

	var input bytes.Buffer
	if err := png.Encode(
		&input,
		sourceImage,
	); err != nil {
		t.Fatal(err)
	}

	header := &multipart.FileHeader{
		Filename: "untrusted-name.png",
		Size:     int64(input.Len()),
	}

	savedImage, err := Save(
		nopMultipartFile{
			Reader: bytes.NewReader(input.Bytes()),
		},
		header,
		"events",
	)
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if !strings.HasPrefix(
		savedImage.URL,
		"/static/uploads/events/",
	) {
		t.Fatalf(
			"unexpected URL: %s",
			savedImage.URL,
		)
	}

	if filepath.Ext(savedImage.FilePath) != ".jpg" {
		t.Fatalf(
			"expected JPEG output, got %s",
			savedImage.FilePath,
		)
	}

	outputBytes, err := os.ReadFile(
		savedImage.FilePath,
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := jpeg.Decode(
		bytes.NewReader(outputBytes),
	); err != nil {
		t.Fatalf(
			"saved image is not valid JPEG: %v",
			err,
		)
	}
}

func TestSaveRejectsNonImage(t *testing.T) {
	input := []byte(
		"<script>alert('x')</script>",
	)

	_, err := Save(
		nopMultipartFile{
			Reader: bytes.NewReader(input),
		},
		&multipart.FileHeader{
			Filename: "payload.jpg",
			Size:     int64(len(input)),
		},
		"services",
	)

	if err != ErrUnsupportedImageType {
		t.Fatalf(
			"expected ErrUnsupportedImageType, got %v",
			err,
		)
	}
}

func TestValidateDimensionsRejectsOversizedImage(
	t *testing.T,
) {
	if err := validateDimensions(
		maxSourceWidth+1,
		1,
	); err != ErrImageDimensions {
		t.Fatalf(
			"expected ErrImageDimensions, got %v",
			err,
		)
	}
}

type nopMultipartFile struct {
	*bytes.Reader
}

func (file nopMultipartFile) Close() error {
	return nil
}
