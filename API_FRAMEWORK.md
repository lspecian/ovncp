# OVN Control Platform API Framework

## Overview
The OVN Control Platform API now has a complete REST API framework built with Go and the Gin web framework. This provides the foundation for implementing CRUD operations for OVN resources.

## Architecture

### Directory Structure
```
ovncp/
├── cmd/api/          # Application entry point
├── internal/         # Private application code
│   ├── api/         # API routing and setup
│   │   ├── handlers/  # HTTP request handlers
│   │   └── middleware/ # Middleware components
│   ├── config/      # Configuration management
│   ├── models/      # Data models
│   └── services/    # Business logic layer
├── pkg/ovn/         # OVN client library
└── build/           # Build artifacts
```

### Key Components

1. **Configuration Management** (`internal/config/`)
   - Environment-based configuration
   - Support for API, OVN, Database, Auth, and Logging settings
   - Validation of required settings

2. **OVN Client** (`pkg/ovn/`)
   - Connection management for Northbound and Southbound DBs
   - Retry logic for resilient operations
   - Thread-safe connection handling

3. **Data Models** (`internal/models/`)
   - Complete OVN resource definitions
   - JSON serialization support
   - Timestamp tracking

4. **Service Layer** (`internal/services/`)
   - Business logic separation
   - OVN operation abstraction
   - Ready for actual OVN integration

5. **API Handlers** (`internal/api/handlers/`)
   - RESTful endpoint implementations
   - Request validation
   - Error handling

6. **Middleware** (`internal/api/middleware/`)
   - Request logging
   - Error recovery
   - Request ID tracking

## API Endpoints

### Health Check
- `GET /health` - Service health status

### Logical Switches
- `GET /api/v1/switches` - List all switches
- `POST /api/v1/switches` - Create a switch
- `GET /api/v1/switches/:id` - Get switch details
- `PUT /api/v1/switches/:id` - Update a switch
- `DELETE /api/v1/switches/:id` - Delete a switch

### Logical Routers
- `GET /api/v1/routers` - List all routers
- `POST /api/v1/routers` - Create a router
- `GET /api/v1/routers/:id` - Get router details
- `PUT /api/v1/routers/:id` - Update a router
- `DELETE /api/v1/routers/:id` - Delete a router

### Ports (Placeholder)
- `GET /api/v1/ports` - List all ports
- `POST /api/v1/ports` - Create a port
- `GET /api/v1/ports/:id` - Get port details
- `PUT /api/v1/ports/:id` - Update a port
- `DELETE /api/v1/ports/:id` - Delete a port

### ACLs (Placeholder)
- `GET /api/v1/acls` - List all ACLs
- `POST /api/v1/acls` - Create an ACL
- `GET /api/v1/acls/:id` - Get ACL details
- `PUT /api/v1/acls/:id` - Update an ACL
- `DELETE /api/v1/acls/:id` - Delete an ACL

## Building and Running

### Using Make
```bash
# Build the API
make build

# Run the API
make run

# Run in development mode
make dev

# Run tests
make test

# Clean build artifacts
make clean
```

### Manual Build
```bash
go build -o build/ovncp-api cmd/api/main.go
./build/ovncp-api
```

### Configuration
Copy `.env.example` to `.env` and configure as needed:
```bash
cp .env.example .env
# Edit .env with your configuration
```

## Features

### Implemented
- ✅ RESTful API structure
- ✅ Configuration management
- ✅ Middleware pipeline (logging, error handling, request IDs)
- ✅ Service layer architecture
- ✅ Data models for all OVN resources
- ✅ Graceful shutdown
- ✅ Environment-based configuration

### Ready for Implementation
- 🔲 Actual OVN integration (requires go-ovn or OVSDB client)
- 🔲 Database persistence
- 🔲 Authentication and authorization
- 🔲 API documentation (OpenAPI/Swagger)
- 🔲 Metrics and monitoring
- 🔲 Rate limiting
- 🔲 WebSocket support for real-time updates

## Next Steps
The framework is ready for implementing actual OVN operations. The next tasks involve:
1. Integrating with OVN using go-ovn library or direct OVSDB protocol
2. Implementing CRUD operations for switches, routers, ports, and ACLs
3. Adding authentication and RBAC
4. Setting up database persistence for audit logging
5. Building the web UI