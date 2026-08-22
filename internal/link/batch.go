package link

// BatchPlan records validation results before a bulk request starts creating
// aliases. It lets the HTTP layer report the first invalid item deterministically.
type BatchPlan struct {
	Total int `json:"total"`
	Valid int `json:"valid"`
}

func PlanBatch(reqs []CreateReq) (BatchPlan, error) {
	plan := BatchPlan{Total: len(reqs)}
	for _, req := range reqs {
		if err := ValidateCreate(req); err != nil {
			return plan, err
		}
		plan.Valid++
	}
	return plan, nil
}
