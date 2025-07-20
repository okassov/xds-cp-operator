# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-07-20

### 🎉 Added
- **Health Check Support**: Complete implementation of HTTP, TCP, and gRPC health checks for Envoy clusters
  - HTTP health checks with custom paths, headers, and status code validation
  - TCP health checks with Base64 encoded binary payload support
  - gRPC health checks with service name and authority configuration
  - Configurable timeouts, intervals, thresholds, and connection reuse settings

### 🔧 Technical Improvements
- Added comprehensive health check API types in `api/v1alpha1/xdscontrolplane_types.go`
- Implemented health check builders in controller for all supported types
- Added proper Base64 encoding/decoding for TCP health check binary data
- Fixed critical bug in TCP health check payload handling (binary vs text)

### 🧪 Testing
- Added 26 comprehensive unit tests for health check functionality
- Implemented integration testing framework with real Envoy proxy v1.28
- Created Docker Compose environment for end-to-end testing
- Added backend services (nginx HTTP server, Python TCP server) for health check validation

### 📚 Documentation
- **NEW**: `docs/healthcheck.md` - Complete health check configuration guide
- **NEW**: `docs/integration-testing.md` - Integration testing with real Envoy proxies
- Updated README.md with health check examples and modern documentation
- Added sample configurations in `config/samples/xds_v1alpha1_xdscontrolplane_healthcheck.yaml`

### 🐛 Bug Fixes
- Fixed TCP health check payload type from `Payload_Text` to `Payload_Binary`
- Resolved Base64 encoding issues in TCP health check implementation
- Fixed Envoy rejection of TCP health checks with proper binary payload handling

### 🏗️ Infrastructure
- Enhanced operator with health check processing capabilities
- Improved error handling and logging for health check operations
- Added validation for health check parameters and configurations

### ✅ Validation
- **Production Ready**: Successfully tested with real Envoy proxy v1.28
- **Protocol Compliance**: Full xDS v3 API compatibility verified
- **Real-world Integration**: Multi-service Docker environment validation completed
- **Binary Data Handling**: Proper Base64 and binary payload processing confirmed

## [0.1.0] - 2024-XX-XX

### Added
- Initial XDS Control Plane Operator implementation
- Basic cluster and listener configuration support
- Multiple Envoy node ID support
- Universal Envoy type support through fallback mechanism
- Transport socket support (proxy protocol, TLS, raw buffer)
- Comprehensive status tracking with phases and conditions
- Lifecycle management for xDS servers

### Technical Details
- Kubernetes operator using controller-runtime
- Support for major Envoy filter types
- JSON to protobuf conversion for configuration
- Real-time configuration updates via xDS protocol

---

## Legend

- 🎉 **Added**: New features
- 🔧 **Changed**: Changes in existing functionality  
- 🐛 **Fixed**: Bug fixes
- 🗑️ **Removed**: Removed features
- 🔒 **Security**: Security improvements
- 📚 **Documentation**: Documentation changes
- 🧪 **Testing**: Testing improvements
- 🏗️ **Infrastructure**: Infrastructure and tooling changes 