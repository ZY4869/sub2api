package service

const (
	channelMonitorMinIntervalSeconds = 15
	channelMonitorMaxIntervalSeconds = 3600
	channelMonitorMaxJitterSeconds   = channelMonitorMaxIntervalSeconds - channelMonitorMinIntervalSeconds
	channelMonitorDegradedThreshold  = 6000
	channelMonitorHistoryKeepDays    = 30
)
