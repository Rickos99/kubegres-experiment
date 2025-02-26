package template

import (
	"path"

	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	kubegresv1 "reactive-tech.io/kubegres/api/v1"
	"reactive-tech.io/kubegres/controllers/ctx"
	"reactive-tech.io/kubegres/controllers/spec/template/yaml"
)

type RestoreJobResourcesCreatorTemplate struct {
	kubegresRestoreContext ctx.KubegresRestoreContext
}

func CreateRestoreJobCreator(kubegresRestoreContext ctx.KubegresRestoreContext) RestoreJobResourcesCreatorTemplate {
	return RestoreJobResourcesCreatorTemplate{
		kubegresRestoreContext: kubegresRestoreContext,
	}
}

func (r *RestoreJobResourcesCreatorTemplate) CreateRestoreJob(kubegresSpec kubegresv1.KubegresSpec) (batchv1.Job, error) {
	restoreJobTemplate, err := r.loadRestoreJobFromTemplate()
	if err != nil {
		return restoreJobTemplate, err
	}

	restoreSpec := r.kubegresRestoreContext.KubegresRestore.Spec

	restoreJobTemplate.Name = r.kubegresRestoreContext.GetRestoreJobName()
	restoreJobTemplate.Namespace = r.kubegresRestoreContext.KubegresRestore.Namespace
	restoreJobTemplate.OwnerReferences = r.getOwnerReference()

	restoreJobTemplate.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName = restoreSpec.DataSource.File.PvcName
	if r.kubegresRestoreContext.KubegresRestore.Spec.CustomConfig != "" {
		restoreJobTemplate.Spec.Template.Spec.Volumes[1].ConfigMap.Name = restoreSpec.CustomConfig
	} else {
		restoreJobTemplate.Spec.Template.Spec.Volumes[1].ConfigMap.Name = ctx.BaseConfigMapName
	}

	container := &restoreJobTemplate.Spec.Template.Spec.Containers[0]
	container.VolumeMounts[0].MountPath = restoreSpec.DataSource.File.Mountpath
	container.Env[0].ValueFrom = r.getKubegresEnvVar(ctx.EnvVarNameOfPostgresSuperUserPsw, kubegresSpec).ValueFrom
	container.Env[1].Value = restoreSpec.ClusterName
	container.Env[2].Value = path.Join(restoreSpec.DataSource.File.Mountpath, restoreSpec.DataSource.File.Snapshot)
	container.Env = append(container.Env, r.kubegresRestoreContext.KubegresRestore.Spec.Env...)

	if r.kubegresRestoreContext.AreResourcesSpecifiedForRestoreJob() {
		restoreJobTemplate.Spec.Template.Spec.Containers[0].Resources = restoreSpec.Resources
	} else {
		restoreJobTemplate.Spec.Template.Spec.Containers[0].Resources = core.ResourceRequirements{}
	}

	return restoreJobTemplate, nil
}

func (r *RestoreJobResourcesCreatorTemplate) CreateFileCheckerPod() (core.Pod, error) {
	podTemplate, err := r.loadFileCheckerPodFromTemplate()
	if err != nil {
		return podTemplate, err
	}

	restoreSpec := r.kubegresRestoreContext.KubegresRestore.Spec

	podTemplate.Name = r.kubegresRestoreContext.GetFileCheckerPodName()
	podTemplate.Namespace = r.kubegresRestoreContext.KubegresRestore.Namespace
	podTemplate.OwnerReferences = r.getOwnerReference()

	container := &podTemplate.Spec.Containers[0]
	container.VolumeMounts[0].MountPath = restoreSpec.DataSource.File.Mountpath
	container.Env[0].Value = path.Join(restoreSpec.DataSource.File.Mountpath, restoreSpec.DataSource.File.Snapshot)

	podTemplate.Spec.Volumes[0].PersistentVolumeClaim.ClaimName = restoreSpec.DataSource.File.PvcName

	return podTemplate, nil
}

func (r *RestoreJobResourcesCreatorTemplate) CreateKubegresResource(kubegresSpec kubegresv1.KubegresSpec) kubegresv1.Kubegres {
	var replicas int32 = 1
	kubegres := kubegresv1.Kubegres{}
	kubegres.Spec = kubegresSpec
	kubegres.ObjectMeta.Name = r.kubegresRestoreContext.KubegresRestore.Spec.ClusterName
	kubegres.ObjectMeta.Namespace = r.kubegresRestoreContext.KubegresRestore.Namespace
	kubegres.Spec.Replicas = &replicas

	kubegres.Labels = map[string]string{}
	kubegres.Labels[ctx.ManagedByKubegresRestoreLabel] = r.kubegresRestoreContext.KubegresRestore.Name

	if r.kubegresRestoreContext.AreResourcesSpecifiedForRestoreJob() {
		kubegres.Spec.Resources = r.kubegresRestoreContext.KubegresRestore.Spec.Resources
	} else {
		kubegres.Spec.Resources = r.kubegresRestoreContext.SourceKubegresClusterSpec.Resources
	}

	return kubegres
}

func (r *RestoreJobResourcesCreatorTemplate) getKubegresEnvVar(envName string, kubegresSpec kubegresv1.KubegresSpec) core.EnvVar {
	for _, envVar := range kubegresSpec.Env {
		if envVar.Name == envName {
			return envVar
		}
	}
	return core.EnvVar{}
}

func (r *RestoreJobResourcesCreatorTemplate) getOwnerReference() []metav1.OwnerReference {
	return []metav1.OwnerReference{*metav1.NewControllerRef(r.kubegresRestoreContext.KubegresRestore, kubegresv1.GroupVersion.WithKind(ctx.KindKubegresRestore))}
}

func (r *RestoreJobResourcesCreatorTemplate) loadRestoreJobFromTemplate() (restoreTemplate batchv1.Job, err error) {
	obj, err := r.decodeYaml(yaml.RestoreJob)

	if err != nil {
		r.kubegresRestoreContext.Log.Error(err, "Unable to load Kubegres Restore Job. Given error:")
		return batchv1.Job{}, err
	}
	return *obj.(*batchv1.Job), nil
}

func (r *RestoreJobResourcesCreatorTemplate) loadFileCheckerPodFromTemplate() (core.Pod, error) {
	obj, err := r.decodeYaml(yaml.FileCheckerPodTemplate)

	if err != nil {
		r.kubegresRestoreContext.Log.Error(err, "Unable to load File Checker Pod. Given error:")
		return core.Pod{}, err
	}
	return *obj.(*core.Pod), nil
}

func (r *RestoreJobResourcesCreatorTemplate) decodeYaml(yamlContents string) (runtime.Object, error) {

	decode := scheme.Codecs.UniversalDeserializer().Decode

	obj, _, err := decode([]byte(yamlContents), nil, nil)

	if err != nil {
		r.kubegresRestoreContext.Log.Error(err, "Error in decode: ", "obj", obj)
	}

	return obj, err
}
