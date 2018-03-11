package engine

import (
	"code.cloudfoundry.org/lager"
	"github.com/concourse/atc"
	"github.com/concourse/atc/db"
	"github.com/concourse/atc/exec"
)

func (build *execBuild) buildAggregateStep(logger lager.Logger, plan atc.Plan) exec.Step {
	logger = logger.Session("aggregate")

	agg := exec.AggregateStep{}

	for _, innerPlan := range *plan.Aggregate {
		innerPlan.Attempts = plan.Attempts
		step := build.buildStep(logger, innerPlan)
		agg = append(agg, step)
	}

	return agg
}

func (build *execBuild) buildDoStep(logger lager.Logger, plan atc.Plan) exec.Step {
	logger = logger.Session("do")

	var step exec.Step

	step = exec.IdentityStep{}

	for i := len(*plan.Do) - 1; i >= 0; i-- {
		innerPlan := (*plan.Do)[i]
		innerPlan.Attempts = plan.Attempts
		previous := build.buildStep(logger, innerPlan)
		step = exec.OnSuccess(previous, step)
	}

	return step
}

func (build *execBuild) buildTimeoutStep(logger lager.Logger, plan atc.Plan) exec.Step {
	innerPlan := plan.Timeout.Step
	innerPlan.Attempts = plan.Attempts
	step := build.buildStep(logger, innerPlan)
	return exec.Timeout(step, plan.Timeout.Duration)
}

func (build *execBuild) buildTryStep(logger lager.Logger, plan atc.Plan) exec.Step {
	innerPlan := plan.Try.Step
	innerPlan.Attempts = plan.Attempts
	step := build.buildStep(logger, innerPlan)
	return exec.Try(step)
}

func (build *execBuild) buildOnAbortStep(logger lager.Logger, plan atc.Plan) exec.Step {
	plan.OnAbort.Step.Attempts = plan.Attempts
	step := build.buildStep(logger, plan.OnAbort.Step)
	plan.OnAbort.Next.Attempts = plan.Attempts
	next := build.buildStep(logger, plan.OnAbort.Next)
	return exec.OnAbort(step, next)
}

func (build *execBuild) buildOnSuccessStep(logger lager.Logger, plan atc.Plan) exec.Step {
	plan.OnSuccess.Step.Attempts = plan.Attempts
	step := build.buildStep(logger, plan.OnSuccess.Step)
	plan.OnSuccess.Next.Attempts = plan.Attempts
	next := build.buildStep(logger, plan.OnSuccess.Next)
	return exec.OnSuccess(step, next)
}

func (build *execBuild) buildOnFailureStep(logger lager.Logger, plan atc.Plan) exec.Step {
	plan.OnFailure.Step.Attempts = plan.Attempts
	step := build.buildStep(logger, plan.OnFailure.Step)
	plan.OnFailure.Next.Attempts = plan.Attempts
	next := build.buildStep(logger, plan.OnFailure.Next)
	return exec.OnFailure(step, next)
}

func (build *execBuild) buildEnsureStep(logger lager.Logger, plan atc.Plan) exec.Step {
	plan.Ensure.Step.Attempts = plan.Attempts
	step := build.buildStep(logger, plan.Ensure.Step)
	plan.Ensure.Next.Attempts = plan.Attempts
	next := build.buildStep(logger, plan.Ensure.Next)
	return exec.Ensure(step, next)
}

// needs rootfs
func (build *execBuild) buildTaskStep(logger lager.Logger, plan atc.Plan) exec.Step {
	logger = logger.Session("task")

	containerMetadata := build.containerMetadata(
		db.ContainerTypeTask,
		plan.Task.Name,
		plan.Attempts,
	)

	return build.factory.Task(
		logger,
		plan,
		build.dbBuild,
		containerMetadata,
		build.delegate.TaskDelegate(plan.ID),
	)
}

// needs rootfs
func (build *execBuild) buildGetStep(logger lager.Logger, plan atc.Plan) exec.Step {
	logger = logger.Session("get", lager.Data{
		"name": plan.Get.Name,
	})

	containerMetadata := build.containerMetadata(
		db.ContainerTypeGet,
		plan.Get.Name,
		plan.Attempts,
	)

	return build.factory.Get(
		logger,
		plan,
		build.dbBuild,
		build.stepMetadata,
		containerMetadata,
		build.delegate.DBActionsBuildEventsDelegate(plan.ID),
		build.delegate.BuildStepDelegate(plan.ID),
	)
}

// needs rootfs
func (build *execBuild) buildPutStep(logger lager.Logger, plan atc.Plan) exec.Step {
	logger = logger.Session("put", lager.Data{
		"name": plan.Put.Name,
	})

	containerMetadata := build.containerMetadata(
		db.ContainerTypePut,
		plan.Put.Name,
		plan.Attempts,
	)

	return build.factory.Put(
		logger,
		plan,
		build.dbBuild,
		build.stepMetadata,
		containerMetadata,
		build.delegate.DBActionsBuildEventsDelegate(plan.ID),
		build.delegate.BuildStepDelegate(plan.ID),
	)
}

func (build *execBuild) buildRetryStep(logger lager.Logger, plan atc.Plan) exec.Step {
	logger = logger.Session("retry")

	steps := []exec.Step{}

	for index, innerPlan := range *plan.Retry {
		innerPlan.Attempts = append(plan.Attempts, index+1)

		step := build.buildStep(logger, innerPlan)
		steps = append(steps, step)
	}

	return exec.Retry(steps...)
}
