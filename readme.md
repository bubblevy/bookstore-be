# Bookstore API

A robust, enterprise-grade RESTful API for an online bookstore e-commerce platform built with **Go 1.27**, **GoFiber v3**, and **PostgreSQL**, following Clean Architecture principles.

The platform includes end-to-end e-commerce capabilities: customer authentication (JWT & Google OAuth 2.0), catalog management, flexible tiered discount promotions, shopping cart, multi-address management, dynamic courier shipping rates with automated order fulfillment via **Biteship**, payment gateway processing with **Midtrans**, and background session maintenance.

---

## Architecture & Project Structure

The codebase is organized according to Clean Architecture and Domain-Driven Design (DDD) principles:

```text
bookstore/
├── cmd/
│   └── main.go                  # Application entry point & graceful shutdown
├── config/                      # Environment configuration loader
├── docs/
│   └── swagger.yml              # Complete OpenAPI 3.0.3 API specification
├── internal/
│   ├── delivery/
│   │   └── http/                # HTTP layer (GoFiber v3)
│   │       ├── dto/             # Data Transfer Objects & validation rules
│   │       ├── handler/         # HTTP request handlers & controllers
│   │       ├── middleware/      # Auth, Admin, Webhook signature, CORS & Rate limiting
│   │       └── router.go        # Route definitions & dependency injection
│   ├── entity/                  # Core domain models, enums & domain errors
│   ├── infrastructure/
│   │   └── pgsql/               # Database repositories & GORM transaction manager
│   ├── repository/              # Repository interfaces & query filter abstractions
│   └── usecase/                 # Business logic implementation & interfaces
├── migrations/                  # Database migration files (Golang Migrate)
├── pkg/                         # Reusable utilities
│   ├── biteship/                # Biteship API HTTP client (Rates, Orders, Tracking)
│   ├── cookie/                  # HTTP-only cookie helpers for JWT tokens
│   ├── email/                   # SMTP email sender for account activation & password reset
│   ├── jwt/                     # JWT token generator & validator
│   ├── logger/                  # Structured zap-based application logger
│   ├── pagination/              # Query pagination helpers & metadata structures
│   ├── payment/                 # Midtrans Core API payment client & signature verifier
│   ├── response/                # Standardized JSON response envelope helpers
│   └── validator/               # Custom request validator & error messages
└── public/                      # Static file assets (e.g. uploaded book covers)
```

---

## Tech Stack

- **Language**: Go 1.27+
- **HTTP Framework**: [GoFiber v3](https://github.com/gofiber/fiber)
- **Database ORM**: [GORM](https://gorm.io/) with PostgreSQL Driver
- **Database**: PostgreSQL 14+
- **Authentication**: JWT (`golang-jwt/jwt/v5`) with HTTP-only cookies & Google OAuth 2.0
- **Validation**: `go-playground/validator/v10` with custom strong password rules
- **Payment Gateway**: [Midtrans Core API](https://midtrans.com/) (Virtual Account, Mandiri Bill, GoPay, QRIS)
- **Logistics & Shipping**: [Biteship API](https://biteship.com/) (Rates calculation, Area search, Fulfillment dispatch, Tracking)
- **Email Service**: SMTP with HTML templates (Account activation, password reset)
- **Database Migrations**: `golang-migrate`

---

## Key Modules & Features

### 1. Authentication & Account Management
- **Registration**: Email verification workflow with activation tokens.
- **Login / Logout**: Issues Access and Refresh tokens stored in secure HTTP-only cookies (or Bearer Authorization header).
- **Token Refreshing**: Automatic token renewal with single-use refresh token invalidation.
- **Password Management**: Forgot password links, token verification, and authenticated password changes.
- **Google OAuth 2.0**: Seamless single sign-on redirect and callback handling.
- **Security**: Strict rate limiting on authentication routes and background session cleanup.

### 2. Catalog Management (Categories, Publishers, Authors, Books)
- **Categories, Authors, Publishers**: Full CRUD operations (Admin) and public catalog browsing.
- **Books**:
  - Full-text search across titles and ISBNs.
  - Multi-attribute filtering (category, author, publisher, price range).
  - Multi-criteria sorting (newest, price ascending/descending, title).
  - Role-based visibility: customers view active items only; admins can toggle `is_active` filter (`true`, `false`, `all`).
  - Automatic URL slug generation and conflict resolution.

### 3. Promotional Discount & Voucher Engine
- Supports standard **Fixed Amount**, **Percentage**, and multi-level **Tiered ("Up To")** reward models.
- Configurable minimum spend requirements, global maximum discount caps, redemption quotas, and validity dates.
- Item eligibility: universal application (`is_all_books: true`) or restricted to specific book IDs.
- Checkout validation endpoint (`/api/v1/discounts/check`) to verify voucher availability and estimate savings before purchase.

### 4. Shopping Cart
- Persistent cart storage per customer.
- Real-time inventory checks and aggregate weight calculation in grams for accurate shipping quotes.
- Selected cart item evaluation for pre-checkout previews.

### 5. Multi-Address Management
- Customers can store multiple shipping destinations (up to 5 addresses per account).
- Primary address setting with automatic fallback during checkout.
- Stores standardized Biteship Destination Area IDs for precise shipping calculations.

### 6. Shipping & Logistics (Biteship Integration)
- **Area Lookup**: Public location search powered by Biteship Maps API (minimum 3 characters).
- **Couriers**: List all active couriers and service tiers (JNE, SiCepat, J&T, Anteraja, ID Express, etc.).
- **Dynamic Rates**: Real-time rate estimation based on combined book weight and destination address.
- **Fulfillment (Admin)**: Dispatch courier pickup requests, creating official tracking numbers (Waybills / AWBs).
- **Return Handling (Admin)**: Process returns and automatically restore inventory stock.
- **Live Tracking**: Real-time milestone tracking for customers and admins.
- **Webhook Processing**: Secure tracking status updates authenticated with `X-Biteship-Signature`.

### 7. Checkout & Payments (Midtrans Integration)
- **Checkout Preview**: Simulates pricing, discounts, shipping fees, admin fees, and payable totals.
- **Order Placement**: Atomically reserves book stock, applies discounts, records shipping details, and generates payment credentials.
- **Midtrans Webhook**: Listens for asynchronous payment status notifications (Settlement, Expire, Cancel, Deny) with SHA512 signature validation.

### 8. Order Lifecycle
Supported order statuses:
- `waiting_payment` -> Order placed, waiting for customer payment before deadline.
- `paid` -> Payment verified via Midtrans webhook.
- `processing` -> Order prepared for shipping.
- `shipped` -> Courier dispatched and tracking number allocated.
- `completed` -> Order successfully delivered to recipient.
- `cancelled` -> Order cancelled or payment deadline expired (stock restored).
- `returned` -> Order returned to store (stock restored).

---

## API Documentation

The complete API specification is available in OpenAPI 3.0.3 format:
- **Specification File**: [`docs/swagger.yml`](docs/swagger.yml)

### Endpoint Summary

| Module | Method | Endpoint | Access | Description |
| :--- | :--- | :--- | :--- | :--- |
| **Health** | `GET` | `/health` | Public | Service liveness probe |
| **Health** | `GET` | `/health/ready` | Public | Database readiness probe |
| **Static** | `GET` | `/api/v1/static/{filepath}` | Public | Serve static assets (covers, etc.) |
| **Auth** | `POST` | `/api/v1/auth/register` | Public | Register new user account |
| **Auth** | `POST` | `/api/v1/auth/login` | Public | Login with email and password |
| **Auth** | `POST` | `/api/v1/auth/refresh-token` | Public | Refresh JWT access token |
| **Auth** | `DELETE` | `/api/v1/auth/logout` | Public | Invalidate refresh token & clear cookies |
| **Auth** | `POST` | `/api/v1/auth/activate-account` | Public | Activate account using email token |
| **Auth** | `POST` | `/api/v1/auth/resend-activation` | Public | Request new activation email |
| **Auth** | `POST` | `/api/v1/auth/forgot-password` | Public | Request password reset email |
| **Auth** | `POST` | `/api/v1/auth/reset-password` | Public | Reset password using email token |
| **Auth** | `POST` | `/api/v1/auth/change-password` | User | Change password for authenticated user |
| **Auth** | `GET` | `/api/v1/auth/google/login` | Public | Initiate Google OAuth 2.0 login |
| **Auth** | `GET` | `/api/v1/auth/google/callback` | Public | Google OAuth 2.0 callback URL |
| **Category** | `GET` | `/api/v1/categories` | Public | List all categories |
| **Category** | `GET` | `/api/v1/categories/{slug}` | Public | Get category details by slug |
| **Category** | `POST` | `/api/v1/categories` | Admin | Create category |
| **Category** | `PUT` | `/api/v1/categories/{id}` | Admin | Update category |
| **Category** | `DELETE` | `/api/v1/categories/{id}` | Admin | Delete category |
| **Publisher**| `GET` | `/api/v1/publishers` | Public | List all publishers |
| **Publisher**| `GET` | `/api/v1/publishers/{slug}` | Public | Get publisher by slug |
| **Publisher**| `POST` | `/api/v1/publishers` | Admin | Create publisher |
| **Publisher**| `PUT` | `/api/v1/publishers/{id}` | Admin | Update publisher |
| **Publisher**| `DELETE` | `/api/v1/publishers/{id}` | Admin | Delete publisher |
| **Author** | `GET` | `/api/v1/authors` | Public | List authors (paginated) |
| **Author** | `GET` | `/api/v1/authors/{slug}` | Public | Get author by slug |
| **Author** | `POST` | `/api/v1/authors` | Admin | Create author |
| **Author** | `PUT` | `/api/v1/authors/{id}` | Admin | Update author |
| **Author** | `DELETE` | `/api/v1/authors/{id}` | Admin | Delete author |
| **Book** | `GET` | `/api/v1/books` | Public | Search & list books (paginated & filtered) |
| **Book** | `GET` | `/api/v1/books/{slug}` | Public | Get book details by slug |
| **Book** | `POST` | `/api/v1/books` | Admin | Create book |
| **Book** | `PUT` | `/api/v1/books/{id}` | Admin | Update book |
| **Book** | `DELETE` | `/api/v1/books/{id}` | Admin | Delete book |
| **Discount** | `POST` | `/api/v1/discounts/check` | User | Validate voucher against cart items |
| **Discount** | `GET` | `/api/v1/discounts` | Admin | List all discounts (paginated) |
| **Discount** | `GET` | `/api/v1/discounts/{id}` | Admin | Get discount details with tiers |
| **Discount** | `POST` | `/api/v1/discounts` | Admin | Create discount promotion |
| **Discount** | `PUT` | `/api/v1/discounts/{id}` | Admin | Update discount promotion |
| **Discount** | `DELETE` | `/api/v1/discounts/{id}` | Admin | Delete discount promotion |
| **Cart** | `GET` | `/api/v1/cart` | User | Get user shopping cart |
| **Cart** | `DELETE` | `/api/v1/cart` | User | Clear all items from cart |
| **Cart** | `POST` | `/api/v1/cart/items` | User | Add book item to cart |
| **Cart** | `PATCH` | `/api/v1/cart/items/{id}` | User | Update item quantity in cart |
| **Cart** | `DELETE` | `/api/v1/cart/items/{id}` | User | Remove item from cart |
| **Cart** | `POST` | `/api/v1/cart/selected` | User | Calculate totals for selected items |
| **Address** | `GET` | `/api/v1/addresses` | User | List all user saved addresses |
| **Address** | `POST` | `/api/v1/addresses` | User | Create a new address (max 5) |
| **Address** | `GET` | `/api/v1/addresses/primary` | User | Retrieve current primary address |
| **Address** | `GET` | `/api/v1/addresses/{id}` | User | Get address details by ID |
| **Address** | `PUT` | `/api/v1/addresses/{id}` | User | Update address details |
| **Address** | `DELETE` | `/api/v1/addresses/{id}` | User | Soft delete an address |
| **Address** | `PATCH` | `/api/v1/addresses/{id}/primary` | User | Set an address as primary |
| **Shipping** | `GET` | `/api/v1/shipping/areas` | Public | Search destination areas by keyword (Biteship) |
| **Shipping** | `GET` | `/api/v1/shipping/couriers` | Public | List available courier partners |
| **Shipping** | `POST` | `/api/v1/shipping/rates` | User | Calculate dynamic courier shipping rates |
| **Shipping** | `POST` | `/api/v1/shipping/webhook` | Webhook | Handle real-time tracking webhook |
| **Shipping** | `GET` | `/api/v1/orders/{invoice_number}/tracking` | User/Admin | Track order delivery status & history |
| **Shipping** | `POST` | `/api/v1/orders/{invoice_number}/fulfill` | Admin | Dispatch courier pickup via Biteship |
| **Shipping** | `POST` | `/api/v1/orders/{invoice_number}/return` | Admin | Process order return & restore stock |
| **Checkout** | `POST` | `/api/v1/checkout/preview` | User | Preview checkout calculation |
| **Checkout** | `POST` | `/api/v1/checkout/order` | User | Create order and charge via Midtrans |
| **Checkout** | `POST` | `/api/v1/payments/midtrans/webhook` | Webhook | Handle Midtrans payment notification |
| **Order** | `GET` | `/api/v1/orders` | User | List order history (status filter optional) |
| **Order** | `GET` | `/api/v1/orders/{invoice_number}` | User | Get complete order detail by invoice |

---

## Environment Variables Configuration

Copy `.env.example` to `.env` and adjust the configuration parameters:

```bash
cp .env.example .env
```

| Variable | Description | Example / Default |
| :--- | :--- | :--- |
| `APP_ENV` | Application environment (`development`, `production`) | `development` |
| `LOG_PATH` | File path for log output in production mode | `logs/app.log` |
| `PORT` | HTTP server listening port | `8000` |
| `APP_NAME` | Application name used in emails and headers | `BookStore` |
| `BASE_URL` | Base backend URL | `http://localhost:8000` |
| `FRONTEND_URL` | Allowed frontend origin for CORS and redirects | `http://localhost:3000` |
| `DATABASE_HOST` | PostgreSQL database host | `localhost` |
| `DATABASE_PORT` | PostgreSQL database port | `5432` |
| `DATABASE_USER` | PostgreSQL user | `postgres` |
| `DATABASE_PASSWORD` | PostgreSQL password | `secret` |
| `DATABASE_NAME` | PostgreSQL database name | `bookstore` |
| `DB_MAX_OPEN_CONNS` | Max open pool connections | `25` |
| `DB_MAX_IDLE_CONNS` | Max idle pool connections | `5` |
| `DB_CONN_MAX_LIFETIME_MINUTES` | Connection max lifetime | `30` |
| `JWT_SECRET` | Secret key for signing JWT tokens | `your-secure-jwt-secret` |
| `SMTP_HOST` | SMTP server host | `smtp.mailtrap.io` |
| `SMTP_PORT` | SMTP server port | `587` |
| `SMTP_USERNAME` | SMTP account username | `username` |
| `SMTP_PASSWORD` | SMTP account password | `password` |
| `EMAIL_FROM_NAME` | Sender display name in emails | `Bookstore Support` |
| `EMAIL_FROM_ADDRESS` | Sender email address | `no-reply@bookstore.com` |
| `SMTP_USE_TLS` | Enable explicit TLS for SMTP | `false` |
| `GOOGLE_CLIENT_ID` | Google OAuth Client ID | `client-id.apps.googleusercontent.com` |
| `GOOGLE_CLIENT_SECRET`| Google OAuth Client Secret | `secret` |
| `GOOGLE_REDIRECT_URL` | Google OAuth authorized redirect URI | `http://localhost:8000/api/v1/auth/google/callback` |
| `MIDTRANS_SERVER_KEY` | Midtrans server key | `SB-Mid-server-XXXX` |
| `MIDTRANS_CLIENT_KEY` | Midtrans client key | `SB-Mid-client-XXXX` |
| `MIDTRANS_IS_PRODUCTION`| Toggle Midtrans sandbox vs production | `false` |
| `BITESHIP_API_KEY` | Biteship integration API key | `biteship_test.XXXX` |
| `BITESHIP_BASE_URL`| Biteship API base URL | `https://api.biteship.com` |
| `BITESHIP_ORIGIN_AREA_ID`| Store origin area ID from Biteship | `IDNP11IDNC278IDND1234IDZ12190` |
| `BITESHIP_ORIGIN_POSTAL_CODE`| Store origin postal code | `12190` |
| `BITESHIP_WEBHOOK_SECRET`| Webhook verification secret token | `your-webhook-secret` |
| `BITESHIP_AVAILABLE_COURIERS`| Comma-separated couriers list | `jne,sicepat,jnt,anteraja,idexpress` |
| `STORE_SHIPPER_NAME`| Store shipper contact name | `Bookstore Warehouse` |
| `STORE_SHIPPER_PHONE`| Store shipper contact phone | `081234567890` |
| `STORE_SHIPPER_EMAIL`| Store shipper contact email | `shipping@bookstore.com` |
| `STORE_ORIGIN_NAME`| Store origin facility name | `Central Book Depot` |
| `STORE_ORIGIN_PHONE`| Store origin facility phone | `081234567890` |
| `STORE_ORIGIN_ADDRESS`| Store origin pickup physical address | `Jl. Kemang Raya No. 10, Jakarta Selatan` |

---

## Getting Started

### 1. Prerequisites
- [Go](https://golang.org/dl/) (version 1.27 or higher)
- [PostgreSQL](https://www.postgresql.org/) (running instance)
- [Golang Migrate CLI](https://github.com/golang-migrate/migrate) (optional, for manual migrations)

### 2. Database Migration
Run the SQL migration scripts in `migrations/` against your PostgreSQL database:

```bash
migrate -path migrations -database "postgres://postgres:password@localhost:5432/bookstore?sslmode=disable" up
```

### 3. Run the Server
Install dependencies and launch the backend server:

```bash
go mod tidy
go run cmd/main.go
```

The server starts on the configured port (default `:8000`).
- **Health Check**: `http://localhost:8000/health`
- **Readiness Check**: `http://localhost:8000/health/ready`
