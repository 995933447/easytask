package task

const (
	// val type: int64
	QueryOptKeyCheckedHealthLt = iota
	// val type: nil
	QueryOptKeyEnabledHeathCheck
	// val type: []string
	QueryOptKeyInIds
	// val type: string
	QueryOptKeyEqName
	// val type: int64
	QueryOptKeyCreatedExceed
	// val type: nil
	QueryOptKeyTaskFinished
)
