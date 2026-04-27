package workflow

import "github.com/github/gh-aw/pkg/logger"

var maintenanceConditionsLog = logger.New("workflow:maintenance_conditions")

// buildNotForkCondition creates a condition to check the repository is not a fork.
func buildNotForkCondition() ConditionNode {
	return &NotNode{
		Child: BuildPropertyAccess("github.event.repository.fork"),
	}
}

// buildNotDispatchOrCallOrEmptyOperation creates a condition that is true when the event
// is not a workflow_dispatch or workflow_call, or the operation input is empty.
// Uses the `inputs.operation` context which works for both workflow_dispatch and workflow_call.
func buildNotDispatchOrCallOrEmptyOperation() ConditionNode {
	return BuildOr(
		BuildAnd(
			BuildNotEquals(
				BuildPropertyAccess("github.event_name"),
				BuildStringLiteral("workflow_dispatch"),
			),
			BuildNotEquals(
				BuildPropertyAccess("github.event_name"),
				BuildStringLiteral("workflow_call"),
			),
		),
		BuildEquals(
			BuildPropertyAccess("inputs.operation"),
			BuildStringLiteral(""),
		),
	)
}

// buildNotForkAndScheduled creates a condition for jobs that should run on any
// non-dispatch/call event including push, or on workflow_dispatch/workflow_call
// with an empty operation, and never on forks. Unlike buildNotForkAndScheduleOnly,
// this function does NOT exclude push events.
// Condition: !fork && ((event_name != 'workflow_dispatch' && event_name != 'workflow_call') || operation == ”)
func buildNotForkAndScheduled() ConditionNode {
	return BuildAnd(
		buildNotForkCondition(),
		buildNotDispatchOrCallOrEmptyOperation(),
	)
}

// buildNotForkAndScheduleOnly creates a condition for jobs that should run on schedule
// (or empty dispatch/call) but NOT on push events, and never on forks.
func buildNotForkAndScheduleOnly() ConditionNode {
	return BuildAnd(
		buildNotForkCondition(),
		BuildAnd(
			BuildNotEquals(
				BuildPropertyAccess("github.event_name"),
				BuildStringLiteral("push"),
			),
			buildNotDispatchOrCallOrEmptyOperation(),
		),
	)
}

// buildNotForkAndScheduleOnlyOrOperation creates a condition for jobs that run on
// schedule (or empty dispatch/call) or when a specific operation is selected,
// but NOT on push events, and never on forks.
func buildNotForkAndScheduleOnlyOrOperation(operation string) ConditionNode {
	maintenanceConditionsLog.Printf("Building not-fork-and-schedule-only-or-operation condition: %s", operation)
	return BuildAnd(
		buildNotForkCondition(),
		BuildAnd(
			BuildNotEquals(
				BuildPropertyAccess("github.event_name"),
				BuildStringLiteral("push"),
			),
			BuildOr(
				buildNotDispatchOrCallOrEmptyOperation(),
				BuildEquals(
					BuildPropertyAccess("inputs.operation"),
					BuildStringLiteral(operation),
				),
			),
		),
	)
}

// buildDispatchOperationCondition creates a condition for jobs that should run
// only when a specific workflow_dispatch or workflow_call operation is selected and not a fork.
// Condition: (dispatch || call) && operation == op && !fork
func buildDispatchOperationCondition(operation string) ConditionNode {
	return BuildAnd(
		BuildAnd(
			BuildOr(
				BuildEventTypeEquals("workflow_dispatch"),
				BuildEventTypeEquals("workflow_call"),
			),
			BuildEquals(
				BuildPropertyAccess("inputs.operation"),
				BuildStringLiteral(operation),
			),
		),
		buildNotForkCondition(),
	)
}

// buildRunOperationCondition creates the condition for the unified run_operation
// job that handles all dispatch/call operations except the ones with dedicated jobs.
// Condition: (dispatch || call) && operation != ” && operation != each excluded && !fork.
func buildRunOperationCondition(excludedOperations ...string) ConditionNode {
	maintenanceConditionsLog.Printf("Building run operation condition, excluding %d operation(s): %v", len(excludedOperations), excludedOperations)
	// Start with: event is workflow_dispatch or workflow_call AND operation is not empty
	condition := BuildAnd(
		BuildOr(
			BuildEventTypeEquals("workflow_dispatch"),
			BuildEventTypeEquals("workflow_call"),
		),
		BuildNotEquals(
			BuildPropertyAccess("inputs.operation"),
			BuildStringLiteral(""),
		),
	)

	// Exclude each dedicated operation
	for _, op := range excludedOperations {
		condition = BuildAnd(
			condition,
			BuildNotEquals(
				BuildPropertyAccess("inputs.operation"),
				BuildStringLiteral(op),
			),
		)
	}

	// AND not a fork
	return BuildAnd(condition, buildNotForkCondition())
}
