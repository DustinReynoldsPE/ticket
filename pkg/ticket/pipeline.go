package ticket

import "fmt"

// Pipelines defines the stage sequence for each ticket type.
var Pipelines = map[TicketType][]Stage{
	TypeFeature: {StageTriage, StageSpec, StageDesign, StageImplement, StageTest, StageVerify, StageDone},
	TypeBug:     {StageTriage, StageImplement, StageTest, StageVerify, StageDone},
	TypeChore:   {StageTriage, StageImplement, StageDone},
	TypeEpic:    {StageTriage, StageSpec, StageDesign, StageDone},
	TypeTask:    {StageTriage, StageImplement, StageTest, StageVerify, StageDone},
}

// PipelineFor returns the stage sequence for the given ticket type.
func PipelineFor(t TicketType) ([]Stage, error) {
	p, ok := Pipelines[t]
	if !ok {
		return nil, fmt.Errorf("no pipeline defined for type %q", t)
	}
	return p, nil
}

// HasStage reports whether the pipeline for the given type includes the stage.
func HasStage(t TicketType, s Stage) bool {
	p, ok := Pipelines[t]
	if !ok {
		return false
	}
	for _, ps := range p {
		if ps == s {
			return true
		}
	}
	return false
}

// StageIndex returns the zero-based position of a stage in a type's pipeline,
// or -1 if the stage is not part of that pipeline.
func StageIndex(t TicketType, s Stage) int {
	p, ok := Pipelines[t]
	if !ok {
		return -1
	}
	for i, ps := range p {
		if ps == s {
			return i
		}
	}
	return -1
}

// NextStage returns the stage after s in the pipeline for type t.
// Returns empty string and false if s is the final stage or not in the pipeline.
func NextStage(t TicketType, s Stage) (Stage, bool) {
	idx := StageIndex(t, s)
	p := Pipelines[t]
	if idx < 0 || idx >= len(p)-1 {
		return "", false
	}
	return p[idx+1], true
}

// PrevStage returns the stage before s in the pipeline for type t.
// Returns empty string and false if s is the first stage or not in the pipeline.
func PrevStage(t TicketType, s Stage) (Stage, bool) {
	idx := StageIndex(t, s)
	if idx <= 0 {
		return "", false
	}
	return Pipelines[t][idx-1], true
}

// IsFinalStage reports whether s is the last stage in the pipeline for type t.
func IsFinalStage(t TicketType, s Stage) bool {
	p, ok := Pipelines[t]
	if !ok {
		return false
	}
	return p[len(p)-1] == s
}
