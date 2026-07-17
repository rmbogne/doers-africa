package mongoid

import "testing"

func TestNormalize(t *testing.T) {
	const expected = "507f1f77bcf86cd799439011"

	testCases := []string{
		expected,
		"  " + expected + "  ",
		`ObjectID("507f1f77bcf86cd799439011")`,
		`ObjectID('507f1f77bcf86cd799439011')`,
	}

	for _, testCase := range testCases {
		actual, err := Normalize(testCase)
		if err != nil {
			t.Fatalf(
				"Normalize(%q) returned error: %v",
				testCase,
				err,
			)
		}

		if actual != expected {
			t.Fatalf(
				"Normalize(%q) = %q, expected %q",
				testCase,
				actual,
				expected,
			)
		}
	}
}

func TestNormalizeRejectsInvalidValues(t *testing.T) {
	testCases := []string{
		"",
		"123",
		"not-an-object-id",
		`ObjectID("invalid")`,
	}

	for _, testCase := range testCases {
		if _, err := Normalize(testCase); err == nil {
			t.Fatalf(
				"expected Normalize(%q) to fail",
				testCase,
			)
		}
	}
}
