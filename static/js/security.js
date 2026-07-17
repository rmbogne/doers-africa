(function () {
    "use strict";

    function csrfToken() {
        const meta = document.querySelector(
            'meta[name="csrf-token"]'
        );

        if (!meta) {
            return "";
        }

        return meta.getAttribute("content") || "";
    }

    function isStateChangingForm(form) {
        const method = (
            form.getAttribute("method") || "GET"
        ).toUpperCase();

        return method === "POST" ||
            method === "PUT" ||
            method === "PATCH" ||
            method === "DELETE";
    }

    function attachCSRFToken(form, token) {
        if (!token || !isStateChangingForm(form)) {
            return;
        }

        let input = form.querySelector(
            'input[name="csrf_token"]'
        );

        if (!input) {
            input = document.createElement("input");
            input.type = "hidden";
            input.name = "csrf_token";
            form.appendChild(input);
        }

        input.value = token;
    }

    function initializeSecurityFields() {
        const token = csrfToken();

        document.querySelectorAll("form").forEach(
            function (form) {
                attachCSRFToken(form, token);
            }
        );
    }

    // Safe DOM utility for any future dynamic messages.
    // Use textContent instead of innerHTML for untrusted values.
    window.aosSetText = function (element, value) {
        if (!element) {
            return;
        }

        element.textContent = String(value ?? "");
    };

    document.addEventListener(
        "DOMContentLoaded",
        initializeSecurityFields
    );
})();
