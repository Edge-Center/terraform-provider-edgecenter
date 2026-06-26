package cloud

import (
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func CreatedResources(kind string, ids ...string) map[string]interface{} {
	values := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		values = append(values, id)
	}

	return map[string]interface{}{
		kind: values,
	}
}

func TaskWithCreatedResources(kind string, ids ...string) *edgecloudV2.Task {
	return &edgecloudV2.Task{
		CreatedResources: CreatedResources(kind, ids...),
	}
}

func TaskWithState(state edgecloudV2.TaskState) *edgecloudV2.Task {
	return &edgecloudV2.Task{
		State: state,
	}
}

func TaskError() *edgecloudV2.Task {
	return TaskWithState(edgecloudV2.TaskStateError)
}
