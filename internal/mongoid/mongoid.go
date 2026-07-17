package mongoid

import (
	"errors"
	"net/url"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrInvalidObjectID = errors.New(
	"invalid MongoDB object ID",
)

// Normalize converts supported object-ID representations into the canonical
// 24-character lowercase hexadecimal form.
//
// Supported inputs:
//   - 507f1f77bcf86cd799439011
//   - ObjectID("507f1f77bcf86cd799439011")
//   - ObjectID('507f1f77bcf86cd799439011')
//
// The wrapper forms are accepted temporarily so legacy URLs do not break.
// New templates must always render IDs with {{.ID.Hex}}.
func Normalize(rawID string) (string, error) {
	decodedID, err := url.PathUnescape(
		strings.TrimSpace(rawID),
	)
	if err != nil {
		return "", ErrInvalidObjectID
	}

	decodedID = strings.Trim(
		decodedID,
		"/ \t\r\n",
	)

	decodedID = unwrapObjectID(decodedID)

	objectID, err := primitive.ObjectIDFromHex(
		decodedID,
	)
	if err != nil {
		return "", ErrInvalidObjectID
	}

	return objectID.Hex(), nil
}

func unwrapObjectID(value string) string {
	const prefix = "ObjectID("

	if !strings.HasPrefix(value, prefix) ||
		!strings.HasSuffix(value, ")") {
		return value
	}

	value = strings.TrimSpace(
		strings.TrimSuffix(
			strings.TrimPrefix(value, prefix),
			")",
		),
	)

	if len(value) >= 2 {
		first := value[0]
		last := value[len(value)-1]

		if (first == '"' && last == '"') ||
			(first == '\'' && last == '\'') {
			value = value[1 : len(value)-1]
		}
	}

	return strings.TrimSpace(value)
}
