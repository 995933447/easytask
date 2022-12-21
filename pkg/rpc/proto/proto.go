package proto

type SchedMode int

const (
	SchedModeNil SchedMode = iota
	SchedModeTimeCron
	SchedModeTimeSpec
	SchedModeTimeInterval
)
