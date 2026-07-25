package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	passwordutil "github.com/mbogne/african-doers/internal/password"
	passwordreset "github.com/mbogne/african-doers/internal/passwordreset"
	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
)

const resetPasswordPage = "reset_password.html"

type passwordResetInput struct {
	RawToken        string
	NewPassword     string
	ConfirmPassword string
}

// ResetPasswordHandler displays the password-reset form and processes
// password-reset submissions.
func ResetPasswordHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	switch r.Method {
	case http.MethodGet:
		showResetPasswordForm(w, r)

	case http.MethodPost:
		processPasswordReset(w, r)

	default:
		writeMethodNotAllowed(
			w,
			http.MethodGet,
			http.MethodPost,
		)
	}
}

// showResetPasswordForm validates the raw token before rendering the form.
func showResetPasswordForm(
	w http.ResponseWriter,
	r *http.Request,
) {
	rawToken := strings.TrimSpace(
		r.URL.Query().Get("token"),
	)

	if err := passwordreset.ValidateToken(
		rawToken,
	); err != nil {
		renderInvalidResetLink(w, r)
		return
	}

	resetToken, _, err :=
		loadValidPasswordResetToken(
			r,
			rawToken,
		)
	if err != nil {
		log.Printf(
			"validate password reset token: %v",
			err,
		)

		handlePasswordResetTokenError(
			w,
			r,
			err,
		)
		return
	}

	log.Printf(
		"valid password reset token found: user_id=%d role=%s",
		resetToken.UserID,
		resetToken.Role,
	)

	render(
		w,
		r,
		resetPasswordPage,
		PageData{
			ResetToken: rawToken,
		},
	)
}

// processPasswordReset coordinates the reset operation while delegating
// validation and persistence to focused helpers.
func processPasswordReset(
	w http.ResponseWriter,
	r *http.Request,
) {
	input, ok := parsePasswordResetInput(w, r)
	if !ok {
		return
	}

	if message := validatePasswordResetInput(input); message != "" {
		renderPasswordResetError(
			w,
			r,
			input.RawToken,
			message,
		)
		return
	}

	tokenHash, err := passwordreset.HashToken(
		input.RawToken,
	)
	if err != nil {
		renderInvalidResetLink(
			w,
			r,
		)
		return
	}

	passwordHash, err :=
		passwordutil.Hash(
			input.NewPassword,
		)
	if err != nil {
		log.Printf(
			"hash new password: %v",
			err,
		)

		http.Error(
			w,
			"Unable to process password reset",
			http.StatusInternalServerError,
		)
		return
	}

	resetRole, err := completePasswordReset(
		r,
		tokenHash,
		passwordHash,
	)
	if err != nil {
		if errors.Is(
			err,
			sql.ErrNoRows,
		) {
			renderInvalidResetLink(
				w,
				r,
			)
			return
		}

		log.Printf(
			"complete password reset: %v",
			err,
		)

		http.Error(
			w,
			"Unable to complete password reset",
			http.StatusInternalServerError,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/login?role="+
			url.QueryEscape(resetRole)+
			"&password_reset=1",
		http.StatusSeeOther,
	)
}

func parsePasswordResetInput(
	w http.ResponseWriter,
	r *http.Request,
) (passwordResetInput, bool) {
	if err := r.ParseForm(); err != nil {
		http.Error(
			w,
			"Invalid password reset request",
			http.StatusBadRequest,
		)

		return passwordResetInput{}, false
	}

	return passwordResetInput{
		RawToken: strings.TrimSpace(
			r.FormValue("token"),
		),
		NewPassword:     r.FormValue("password"),
		ConfirmPassword: r.FormValue("confirm_password"),
	}, true
}

func validatePasswordResetInput(
	input passwordResetInput,
) string {
	if input.RawToken == "" {
		return "The password reset token is missing."
	}

	if input.NewPassword !=
		input.ConfirmPassword {
		return "The passwords do not match."
	}

	if err := passwordreset.ValidateToken(
		input.RawToken,
	); err != nil {
		return "The password reset link is invalid or expired."
	}

	if err := passwordutil.Validate(
		input.NewPassword,
	); err != nil {
		return passwordValidationMessage(err)
	}

	return ""
}

func passwordValidationMessage(
	err error,
) string {
	switch {
	case errors.Is(
		err,
		passwordutil.ErrTooShort,
	):
		return fmt.Sprintf(
			"Your password must contain at least %d characters.",
			passwordutil.MinimumLength,
		)

	case errors.Is(
		err,
		passwordutil.ErrTooLong,
	):
		return fmt.Sprintf(
			"Your password must not exceed %d bytes.",
			passwordutil.MaximumBytes,
		)

	default:
		return "The password does not meet the required policy."
	}
}

func renderPasswordResetError(
	w http.ResponseWriter,
	r *http.Request,
	rawToken string,
	message string,
) {
	render(
		w,
		r,
		resetPasswordPage,
		PageData{
			ResetToken:         rawToken,
			PasswordResetError: message,
		},
	)
}

func renderInvalidResetLink(
	w http.ResponseWriter,
	r *http.Request,
) {
	render(
		w,
		r,
		resetPasswordPage,
		PageData{
			PasswordResetError: "The password reset link is invalid or expired.",
		},
	)
}

func loadValidPasswordResetToken(
	r *http.Request,
	rawToken string,
) (
	models.PasswordResetToken,
	string,
	error,
) {
	tokenHash, err :=
		passwordreset.HashToken(
			rawToken,
		)
	if err != nil {
		return models.PasswordResetToken{},
			"",
			err
	}

	resetToken, err :=
		store.DB.GetValidPasswordResetToken(
			r.Context(),
			tokenHash,
		)
	if err != nil {
		return models.PasswordResetToken{},
			"",
			err
	}

	return resetToken, tokenHash, nil
}

func handlePasswordResetTokenError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		renderInvalidResetLink(w, r)
		return
	}

	log.Printf(
		"load valid password reset token: %v",
		err,
	)

	http.Error(
		w,
		"Unable to validate password reset link",
		http.StatusInternalServerError,
	)
}

func completePasswordReset(
	r *http.Request,
	tokenHash string,
	passwordHash string,
) (string, error) {
	role, err :=
		store.DB.CompletePasswordReset(
			r.Context(),
			tokenHash,
			passwordHash,
		)
	if err != nil {
		return "", fmt.Errorf(
			"complete transactional password reset: %w",
			err,
		)
	}

	return role, nil
}
