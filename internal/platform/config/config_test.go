package config

import "testing"

func TestDefaultUsesLocalAddressesWithoutEnvironmentOverrides(t *testing.T) {
	for _, name := range []string{"HTTP_ADDR", "STATE_GRPC_ADDR", "REDIS_ADDR", "RCENTER_GRPC_ADDR", "TCP_ADDR"} {
		t.Setenv(name, "")
	}

	cfg := Default()
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8080")
	}
	if cfg.StateGRPCAddr != "127.0.0.1:9001" {
		t.Fatalf("StateGRPCAddr = %q, want %q", cfg.StateGRPCAddr, "127.0.0.1:9001")
	}
	if cfg.Redis.Addr != "127.0.0.1:6379" {
		t.Fatalf("Redis.Addr = %q, want %q", cfg.Redis.Addr, "127.0.0.1:6379")
	}
	if cfg.RCenterGRPCAddr != "127.0.0.1:9002" {
		t.Fatalf("RCenterGRPCAddr = %q, want %q", cfg.RCenterGRPCAddr, "127.0.0.1:9002")
	}
	if cfg.TCPAddr != ":8081" {
		t.Fatalf("TCPAddr = %q, want %q", cfg.TCPAddr, ":8081")
	}
}

func TestDefaultUsesAddressEnvironmentOverrides(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":8088")
	t.Setenv("STATE_GRPC_ADDR", "state:9001")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("RCENTER_GRPC_ADDR", "rcenter:9002")
	t.Setenv("TCP_ADDR", ":8089")

	cfg := Default()
	if cfg.HTTPAddr != ":8088" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8088")
	}
	if cfg.StateGRPCAddr != "state:9001" {
		t.Fatalf("StateGRPCAddr = %q, want %q", cfg.StateGRPCAddr, "state:9001")
	}
	if cfg.Redis.Addr != "redis:6379" {
		t.Fatalf("Redis.Addr = %q, want %q", cfg.Redis.Addr, "redis:6379")
	}
	if cfg.RCenterGRPCAddr != "rcenter:9002" {
		t.Fatalf("RCenterGRPCAddr = %q, want %q", cfg.RCenterGRPCAddr, "rcenter:9002")
	}
	if cfg.TCPAddr != ":8089" {
		t.Fatalf("TCPAddr = %q, want %q", cfg.TCPAddr, ":8089")
	}
}
