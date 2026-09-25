Struktur Folder:
/cmd                // Entry Point
/internal           // Internal Workflow
  /entity           // Business Domain Model
  /usecase          // Interface & Main Business Logic
  /repository       // Interface (Abstraction) of data layer
  /infrastructure   // Implementation of data layer Interface
  /delivery         // Handler
/pkg                // Helper & Utility
/migrations         // Database Migration


Tech:
Go 1.27
GoFiber V3
Gorm
Postgres
Golang Migrate
Validator
JWT
Godotenv

