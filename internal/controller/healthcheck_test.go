package controller

import (
	"testing"
	"time"

	api "github.com/okassov/xds-cp-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestBuildHealthCheck(t *testing.T) {
	reconciler := &XDSControlPlaneReconciler{}

	t.Run("HTTP Health Check", func(t *testing.T) {
		hcSpec := &api.HealthCheckSpec{
			Timeout:            "5s",
			Interval:           "10s",
			IntervalJitter:     "1s",
			UnhealthyThreshold: 3,
			HealthyThreshold:   2,
			ReuseConnection:    true,
			HTTPHealthCheck: &api.HTTPHealthCheckSpec{
				Path: "/health",
				Host: "example.com",
				RequestHeadersToAdd: []api.HeaderValueOptionSpec{
					{
						Header: api.HeaderValueSpec{
							Key:   "X-Health-Check",
							Value: "envoy",
						},
						Append: false,
					},
				},
				ExpectedStatuses: []api.HTTPStatusRangeSpec{
					{Start: 200, End: 299},
				},
			},
		}

		hc, err := reconciler.buildHealthCheck(hcSpec)
		require.NoError(t, err)
		require.NotNil(t, hc)

		// Check basic fields
		assert.Equal(t, durationpb.New(5*time.Second), hc.Timeout)
		assert.Equal(t, durationpb.New(10*time.Second), hc.Interval)
		assert.Equal(t, durationpb.New(1*time.Second), hc.IntervalJitter)
		assert.Equal(t, wrapperspb.UInt32(3), hc.UnhealthyThreshold)
		assert.Equal(t, wrapperspb.UInt32(2), hc.HealthyThreshold)
		assert.Equal(t, wrapperspb.Bool(true), hc.ReuseConnection)

		// Check HTTP health check
		httpHC := hc.GetHttpHealthCheck()
		require.NotNil(t, httpHC)
		assert.Equal(t, "/health", httpHC.Path)
		assert.Equal(t, "example.com", httpHC.Host)
		assert.Len(t, httpHC.RequestHeadersToAdd, 1)
		assert.Equal(t, "X-Health-Check", httpHC.RequestHeadersToAdd[0].Header.Key)
		assert.Equal(t, "envoy", httpHC.RequestHeadersToAdd[0].Header.Value)
		assert.Equal(t, wrapperspb.Bool(false), httpHC.RequestHeadersToAdd[0].Append)
		assert.Len(t, httpHC.ExpectedStatuses, 1)
		assert.Equal(t, int64(200), httpHC.ExpectedStatuses[0].Start)
		assert.Equal(t, int64(299), httpHC.ExpectedStatuses[0].End)
	})

	t.Run("TCP Health Check", func(t *testing.T) {
		hcSpec := &api.HealthCheckSpec{
			Timeout:            "3s",
			Interval:           "5s",
			UnhealthyThreshold: 2,
			HealthyThreshold:   1,
			ReuseConnection:    false,
			TCPHealthCheck: &api.TCPHealthCheckSpec{
				Send:    []byte("PING"),
				Receive: [][]byte{[]byte("PONG")},
			},
		}

		hc, err := reconciler.buildHealthCheck(hcSpec)
		require.NoError(t, err)
		require.NotNil(t, hc)

		// Check basic fields
		assert.Equal(t, durationpb.New(3*time.Second), hc.Timeout)
		assert.Equal(t, durationpb.New(5*time.Second), hc.Interval)
		assert.Equal(t, wrapperspb.UInt32(2), hc.UnhealthyThreshold)
		assert.Equal(t, wrapperspb.UInt32(1), hc.HealthyThreshold)
		assert.Equal(t, wrapperspb.Bool(false), hc.ReuseConnection)

		// Check TCP health check
		tcpHC := hc.GetTcpHealthCheck()
		require.NotNil(t, tcpHC)
		assert.Equal(t, "PING", tcpHC.Send.GetText())
		assert.Len(t, tcpHC.Receive, 1)
		assert.Equal(t, "PONG", tcpHC.Receive[0].GetText())
	})

	t.Run("gRPC Health Check", func(t *testing.T) {
		hcSpec := &api.HealthCheckSpec{
			Timeout:            "2s",
			Interval:           "8s",
			UnhealthyThreshold: 1,
			HealthyThreshold:   1,
			GRPCHealthCheck: &api.GRPCHealthCheckSpec{
				ServiceName: "myapp.v1.HealthService",
				Authority:   "grpc-service.local",
			},
		}

		hc, err := reconciler.buildHealthCheck(hcSpec)
		require.NoError(t, err)
		require.NotNil(t, hc)

		// Check gRPC health check
		grpcHC := hc.GetGrpcHealthCheck()
		require.NotNil(t, grpcHC)
		assert.Equal(t, "myapp.v1.HealthService", grpcHC.ServiceName)
		assert.Equal(t, "grpc-service.local", grpcHC.Authority)
	})

	t.Run("Default TCP Health Check", func(t *testing.T) {
		hcSpec := &api.HealthCheckSpec{
			Timeout:  "1s",
			Interval: "5s",
		}

		hc, err := reconciler.buildHealthCheck(hcSpec)
		require.NoError(t, err)
		require.NotNil(t, hc)

		// Should default to TCP health check
		tcpHC := hc.GetTcpHealthCheck()
		require.NotNil(t, tcpHC)
	})

	t.Run("Default Values", func(t *testing.T) {
		hcSpec := &api.HealthCheckSpec{}

		hc, err := reconciler.buildHealthCheck(hcSpec)
		require.NoError(t, err)
		require.NotNil(t, hc)

		// Check defaults
		assert.Equal(t, durationpb.New(5*time.Second), hc.Timeout)
		assert.Equal(t, durationpb.New(10*time.Second), hc.Interval)
		assert.Equal(t, wrapperspb.UInt32(3), hc.UnhealthyThreshold)
		assert.Equal(t, wrapperspb.UInt32(2), hc.HealthyThreshold)
		assert.Equal(t, wrapperspb.Bool(false), hc.ReuseConnection)
	})

	t.Run("Invalid Timeout", func(t *testing.T) {
		hcSpec := &api.HealthCheckSpec{
			Timeout: "invalid-duration",
		}

		hc, err := reconciler.buildHealthCheck(hcSpec)
		assert.Error(t, err)
		assert.Nil(t, hc)
		assert.Contains(t, err.Error(), "invalid health check timeout")
	})

	t.Run("Invalid Interval", func(t *testing.T) {
		hcSpec := &api.HealthCheckSpec{
			Timeout:  "1s",
			Interval: "invalid-duration",
		}

		hc, err := reconciler.buildHealthCheck(hcSpec)
		assert.Error(t, err)
		assert.Nil(t, hc)
		assert.Contains(t, err.Error(), "invalid health check interval")
	})
}

func TestBuildHTTPHealthCheck(t *testing.T) {
	reconciler := &XDSControlPlaneReconciler{}

	t.Run("Full HTTP Config", func(t *testing.T) {
		httpSpec := &api.HTTPHealthCheckSpec{
			Path: "/api/health",
			Host: "api.example.com",
			RequestHeadersToAdd: []api.HeaderValueOptionSpec{
				{
					Header: api.HeaderValueSpec{
						Key:   "Authorization",
						Value: "Bearer token",
					},
					Append: true,
				},
			},
			ExpectedStatuses: []api.HTTPStatusRangeSpec{
				{Start: 200, End: 204},
				{Start: 301, End: 302},
			},
		}

		httpHC, err := reconciler.buildHTTPHealthCheck(httpSpec)
		require.NoError(t, err)
		require.NotNil(t, httpHC)

		assert.Equal(t, "/api/health", httpHC.Path)
		assert.Equal(t, "api.example.com", httpHC.Host)
		assert.Len(t, httpHC.RequestHeadersToAdd, 1)
		assert.Equal(t, "Authorization", httpHC.RequestHeadersToAdd[0].Header.Key)
		assert.Equal(t, "Bearer token", httpHC.RequestHeadersToAdd[0].Header.Value)
		assert.Equal(t, wrapperspb.Bool(true), httpHC.RequestHeadersToAdd[0].Append)
		assert.Len(t, httpHC.ExpectedStatuses, 2)
		assert.Equal(t, int64(200), httpHC.ExpectedStatuses[0].Start)
		assert.Equal(t, int64(204), httpHC.ExpectedStatuses[0].End)
	})

	t.Run("Minimal HTTP Config", func(t *testing.T) {
		httpSpec := &api.HTTPHealthCheckSpec{
			Path: "/health",
		}

		httpHC, err := reconciler.buildHTTPHealthCheck(httpSpec)
		require.NoError(t, err)
		require.NotNil(t, httpHC)

		assert.Equal(t, "/health", httpHC.Path)
		assert.Empty(t, httpHC.Host)
		assert.Empty(t, httpHC.RequestHeadersToAdd)
		// Should have default status range
		assert.Len(t, httpHC.ExpectedStatuses, 1)
		assert.Equal(t, int64(200), httpHC.ExpectedStatuses[0].Start)
		assert.Equal(t, int64(299), httpHC.ExpectedStatuses[0].End)
	})
}
