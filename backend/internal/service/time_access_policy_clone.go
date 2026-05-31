package service

func cloneTimeAccessPolicy(policy *TimeAccessPolicy) *TimeAccessPolicy {
	if policy == nil {
		return nil
	}
	out := *policy
	if policy.NotBefore != nil {
		v := *policy.NotBefore
		out.NotBefore = &v
	}
	if policy.NotAfter != nil {
		v := *policy.NotAfter
		out.NotAfter = &v
	}
	if policy.DailyAllowedMinutes != nil {
		v := *policy.DailyAllowedMinutes
		out.DailyAllowedMinutes = &v
	}
	if len(policy.WeeklyWindows) > 0 {
		out.WeeklyWindows = make([]TimeAccessWindow, 0, len(policy.WeeklyWindows))
		for _, window := range policy.WeeklyWindows {
			out.WeeklyWindows = append(out.WeeklyWindows, TimeAccessWindow{
				Days:  append([]int(nil), window.Days...),
				Start: window.Start,
				End:   window.End,
			})
		}
	}
	return &out
}
