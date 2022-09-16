/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/1284551258/kubebuilder-example/controllers/utils"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ingressv1beta1 "github.com/1284551258/kubebuilder-example/api/v1beta1"
)

// AppReconciler reconciles a App object
type AppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ingress.gientech.com,resources=apps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ingress.gientech.com,resources=apps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ingress.gientech.com,resources=apps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the App object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *AppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("有事件来了" + req.String())
	// TODO(user): your logic here
	app := &ingressv1beta1.App{}
	//从缓存中获取app
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		//判断是不是没找到，没找到返回nil，不是返回err
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//1.deployment的处理
	deployment := utils.NewDeployment(app)
	if err := controllerutil.SetControllerReference(app, deployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	//找到同名的deployment
	d := &v1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, d); err != nil {
		if errors.IsNotFound(err) {
			//如果没找到deployment，则需要创建
			if err := r.Create(ctx, deployment); err != nil {
				logger.Error(err, "create deploy failed!")
				return ctrl.Result{}, err
			}
			logger.Info("create deploy successful!")
		}
	} else {
		//找到deployment，则需要更新
		//if err := r.Update(ctx, deployment); err != nil {
		//	更新deployment
		//return ctrl.Result{}, err
		//}
		logger.Info("update deploy successful!")
	}

	//2.service的处理
	service := utils.NewService(app)
	if err := controllerutil.SetControllerReference(app, service, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	//查找指定service
	s := &corev1.Service{}
	if err := r.Get(ctx, req.NamespacedName, s); err != nil {
		if errors.IsNotFound(err) && app.Spec.EnableService {
			if err := r.Create(ctx, service); err != nil {
				logger.Error(err, "create service failed!")
				return ctrl.Result{}, err
			}
			logger.Info("create service successful!")
		}
	} else {
		//找到service，则更新
		if app.Spec.EnableService {
			//if err := r.Update(ctx, service); err != nil {
			//	return ctrl.Result{}, err
			//}
		} else {
			if err := r.Delete(ctx, s); err != nil {
				return ctrl.Result{}, err
			}
			logger.Info("delete service successful!")
		}
	}
	//3. Ingress的处理,ingress配置可能为空
	//TODO 使用admission校验该值,如果启用了ingress，那么service必须启用
	//TODO 使用admission设置默认值,默认为false
	//Fix: 这里会导致Ingress无法被删除
	//if !app.Spec.EnableService {
	//	return ctrl.Result{}, nil
	//}
	ingress := utils.NewIngress(app)
	if err := controllerutil.SetControllerReference(app, ingress, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	i := &v12.Ingress{}
	if err := r.Get(ctx, req.NamespacedName, i); err != nil {
		if errors.IsNotFound(err) && app.Spec.EnableIngress {
			if err := r.Create(ctx, ingress); err != nil {
				return ctrl.Result{}, err
			}
			logger.Info("create ingress successful!")
		}
	} else {
		if app.Spec.EnableIngress {
			//启用了ingress，进行更新
			//if err := r.Update(ctx, ingress); err != nil {
			//	return ctrl.Result{}, err
			//}
			logger.Info("update ingress successful!")
		} else {
			//没启用ingress，进行删除
			if err := r.Delete(ctx, i); err != nil {
				return ctrl.Result{}, err
			}
			logger.Info("delete ingress successful!")
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ingressv1beta1.App{}).
		Owns(&v1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&v12.Ingress{}).
		Complete(r)
}
