# African Doers

A modern platform connecting African event organizers and service providers (Doers) with prospects and customers.

## Overview
This full-stack web application is built with standard Go libraries on the backend and pure HTML, CSS, and JavaScript on the frontend. The project embodies a premium and responsive glassmorphism aesthetic, aiming to connect service providers with eager customers across Africa.

### Stack
- **Backend:** Go (`net/http`)
- **Frontend:** HTML5, CSS3 (Vanilla + Custom Properties), Vanilla JS
- **Data Store:** In-memory store with seeded dummy data

## Features
- **Home Carousel:** Auto-advancing sliding window of the top 5 upcoming events.
- **Role-based Authentication:** Separate registrations and dashboards for 'Doers' and 'Customers'.
- **Doer Dashboard:** Manage your events and services (creation/archiving operations).
- **Customer Dashboard:** Browse your upcoming RSVP'd events.
- **Event Detail & RSVP:** View event details and easily RSVP.
- **Prospect Browsing:** Unauthenticated index of all Doers and public events.
- **Observability:** Custom logging middleware for audits and monitoring.

## Getting Started

### Prerequisites
- [Go 1.20+](https://golang.org/dl/) installed on your machine.

### Installation
1. Clone the repository and `cd` into the project root:
   ```bash
   cd african-doers
   ```
2. Download dependencies (none currently needed outside the standard library):
   ```bash
   go mod tidy
   ```

### Running the Application
To start the application locally:
```bash
go run main.go
```
The server will start on `http://localhost:8080`.

### Testing with Dummy Data
The database is pre-seeded. Feel free to use existing accounts to test features:
- **Doer:** `kwame@events.com` | Password: `password123`
- **Customer:** `alice@test.com` | Password: `password123`

You can also register a brand new Doer or Customer from the `/register` page.

## Making Further Changes
- **Templates:** Modify `.html` files in the `templates/` directory. The layout is defined in `base.html` and injected via `{{define "content"}}`.
- **Styling:** CSS is purely vanilla. Adjust tokens in `static/css/style.css` under the `:root` pseudo-class for immediate theme changes.
- **Backend:** Application logic is housed in the `handlers/` directory. New roles or permissions can be injected within `middleware/auth.go`.
- **Data Model:** Extending entities is straightforward by modifying structs in `models/models.go` and modifying the in-memory map logic present within `store/store.go`. To move. to a persistent database such as PostgreSQL or SQLite, implement methods matching the `Store` signature.
