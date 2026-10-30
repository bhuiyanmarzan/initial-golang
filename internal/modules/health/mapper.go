package health

func (d ReadinessRequest) toInput() CheckInput { return CheckInput{Dependency: d.Dependency} }
func toResponse(result Result) Response {
	if result.Ready {
		return Response{Status: "ready"}
	}
	return Response{Status: "not_ready"}
}
