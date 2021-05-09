package base

type QueryResult struct {
	OK     bool                `json:"ok"`
	Result []*CurrentCfsConfig `json:"result"`
}

type CurrentCfsConfig struct {
	CfsPeriodUS int32 `json:"cfs_period_us"`
	CfsQuotaUS  int32 `json:"cfs_quota_us"`
}
