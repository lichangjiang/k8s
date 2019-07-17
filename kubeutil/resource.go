package kubeutil

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SetOwnerRef(obj *metav1.ObjectMeta, ownerRef *metav1.OwnerReference) {
	if ownerRef == nil {
		return
	}

	SetOwnerRefs(obj, []metav1.OwnerReference{*ownerRef})
}

func SetOwnerRefs(obj *metav1.ObjectMeta, ownerRefs []metav1.OwnerReference) {
	obj.OwnerReferences = ownerRefs
}
