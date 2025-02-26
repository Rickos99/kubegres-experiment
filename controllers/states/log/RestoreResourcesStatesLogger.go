package log

import (
	"reactive-tech.io/kubegres/controllers/ctx"
	"reactive-tech.io/kubegres/controllers/states"
)

type RestoreResourcesStatesLogger struct {
	kubegresRestoreContext ctx.KubegresRestoreContext
	restoreResourcesStates states.RestoreResourceStates
}

func CreateRestoreResourcesStatesLogger(kubegresRestoreContext ctx.KubegresRestoreContext, restoreResourcesStates states.RestoreResourceStates) RestoreResourcesStatesLogger {
	return RestoreResourcesStatesLogger{
		kubegresRestoreContext: kubegresRestoreContext,
		restoreResourcesStates: restoreResourcesStates,
	}
}

func (r *RestoreResourcesStatesLogger) Log() {
	r.logKubegresStates()
	r.logRestoreJobStates()
	r.logFileCheckerPodStates()
}

func (r *RestoreResourcesStatesLogger) logKubegresStates() {
	r.kubegresRestoreContext.Log.Info("Kubegres states.",
		"IsDeployed", r.restoreResourcesStates.Cluster.IsDeployed,
		"IsReady", r.restoreResourcesStates.Cluster.IsReady)
}

func (r *RestoreResourcesStatesLogger) logRestoreJobStates() {
	r.kubegresRestoreContext.Log.Info("RestoreJob states.",
		"IsJobDeployed", r.restoreResourcesStates.Job.IsJobDeployed,
		"IsPvcDeployed", r.restoreResourcesStates.Job.IsPvcDeployed,
		"JobPhase", r.restoreResourcesStates.Job.JobPhase)
}

func (r *RestoreResourcesStatesLogger) logFileCheckerPodStates() {
	r.kubegresRestoreContext.Log.Info("FileCheckerPod states.",
		"IsPodDeployed", r.restoreResourcesStates.FileChecker.IsPodDeployed,
		"IsPodTerminated", r.restoreResourcesStates.FileChecker.IsPodTerminated,
		"ExitStatus", r.restoreResourcesStates.FileChecker.ExitStatus,
	)
}
