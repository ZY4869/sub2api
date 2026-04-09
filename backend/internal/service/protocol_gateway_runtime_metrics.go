package service

import "github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"

type ProtocolGatewayRuntimeMetricsSnapshot = protocolruntime.MetricsSnapshot

func SnapshotProtocolGatewayRuntimeMetrics() ProtocolGatewayRuntimeMetricsSnapshot {
	return protocolruntime.Snapshot()
}
