(function () {
    "use strict";

    const defaultMaximumBytes = 2 * 1024 * 1024;

    function popupFor(form) {
        const popupID = form.dataset.uploadPopup;

        if (!popupID) {
            return null;
        }

        return document.getElementById(popupID);
    }

    function showPopup(form, message) {
        const popup = popupFor(form);

        if (!popup) {
            return;
        }

        const messageElement = popup.querySelector(
            "[data-upload-message]"
        );

        if (messageElement) {
            messageElement.textContent = message;
        }

        popup.hidden = false;
        popup.focus();
    }

    function hidePopup(popup) {
        if (popup) {
            popup.hidden = true;
        }
    }

    function maximumBytes(form) {
        const configuredMaximum = Number(
            form.dataset.maxUploadBytes
        );

        if (
            Number.isFinite(configuredMaximum) &&
            configuredMaximum > 0
        ) {
            return configuredMaximum;
        }

        return defaultMaximumBytes;
    }

    function validateSelectedFile(form) {
        const input = form.querySelector(
            'input[type="file"][name="image"]'
        );

        if (
            !input ||
            !input.files ||
            input.files.length === 0
        ) {
            return true;
        }

        const selectedFile = input.files[0];
        const maximum = maximumBytes(form);

        if (selectedFile.size <= maximum) {
            return true;
        }

        showPopup(
            form,
            "The selected image is too large. " +
                "Choose a JPEG or PNG smaller than 2 MB."
        );

        input.value = "";

        return false;
    }

    function removeUploadErrorFromURL() {
        const currentURL = new URL(
            document.location.href
        );

        if (
            !currentURL.searchParams.has(
                "upload_error"
            )
        ) {
            return;
        }

        currentURL.searchParams.delete(
            "upload_error"
        );

        const query = currentURL.searchParams.toString();

        history.replaceState(
            {},
            document.title,
            currentURL.pathname +
                (query ? "?" + query : "")
        );
    }

    document.addEventListener(
        "DOMContentLoaded",
        function () {
            const forms = document.querySelectorAll(
                "form[data-image-upload-form]"
            );

            forms.forEach(function (form) {
                const input = form.querySelector(
                    'input[type="file"][name="image"]'
                );

                if (input) {
                    input.addEventListener(
                        "change",
                        function () {
                            validateSelectedFile(form);
                        }
                    );
                }

                form.addEventListener(
                    "submit",
                    function (event) {
                        if (!validateSelectedFile(form)) {
                            event.preventDefault();
                        }
                    }
                );
            });

            document.querySelectorAll(
                "[data-upload-dismiss]"
            ).forEach(function (button) {
                button.addEventListener(
                    "click",
                    function () {
                        hidePopup(
                            button.closest(
                                '[role="alert"]'
                            )
                        );
                    }
                );
            });

            removeUploadErrorFromURL();
        }
    );
})();
