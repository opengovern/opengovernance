package statemanager

import (
	"context"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	contourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

func (s *Service) syncHTTPProxy(workspaces []*db.Workspace) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var httpIncludes []contourv1.Include
	var grpcIncludes []contourv1.Include
	for _, w := range workspaces {
		if w.Status != api.StatusProvisioned && w.Status != api.StatusBootstrapping {
			continue
		}
		httpIncludes = append(httpIncludes, contourv1.Include{
			Name:      "http-proxy-route",
			Namespace: w.ID,
			Conditions: []contourv1.MatchCondition{
				{
					Prefix: "/" + w.Name,
				},
			},
		})
		grpcIncludes = append(grpcIncludes, contourv1.Include{
			Name:      "grpc-proxy-route",
			Namespace: w.ID,
			Conditions: []contourv1.MatchCondition{
				{
					Header: &contourv1.HeaderMatchCondition{
						Name:  "workspace-name",
						Exact: w.Name,
					},
				},
			},
		})
	}

	httpKey := types.NamespacedName{
		Name:      "http-proxy-route",
		Namespace: s.cfg.KaytuOctopusNamespace,
	}
	var httpProxy contourv1.HTTPProxy

	grpcKey := types.NamespacedName{
		Name:      "grpc-proxy-route",
		Namespace: s.cfg.KaytuOctopusNamespace,
	}
	var grpcProxy contourv1.HTTPProxy

	httpExists := true
	if err := s.kubeClient.Get(ctx, httpKey, &httpProxy); err != nil {
		if apierrors.IsNotFound(err) {
			httpExists = false
		} else {
			s.logger.Error("failed to get http proxy", zap.Error(err))
			return err
		}
	}

	grpcExists := true
	if err := s.kubeClient.Get(ctx, grpcKey, &grpcProxy); err != nil {
		if apierrors.IsNotFound(err) {
			grpcExists = false
		} else {
			s.logger.Error("failed to get grpc proxy", zap.Error(err))
			return err
		}
	}

	httpResourceVersion := httpProxy.GetResourceVersion()
	httpProxy = contourv1.HTTPProxy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPProxy",
			APIVersion: "projectcontour.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "http-proxy-route",
			Namespace: s.cfg.KaytuOctopusNamespace,
		},
		Spec: contourv1.HTTPProxySpec{
			Includes: httpIncludes,
		},
		Status: contourv1.HTTPProxyStatus{},
	}

	grpcResourceVersion := grpcProxy.GetResourceVersion()
	grpcProxy = contourv1.HTTPProxy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPProxy",
			APIVersion: "projectcontour.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grpc-proxy-route",
			Namespace: s.cfg.KaytuOctopusNamespace,
		},
		Spec: contourv1.HTTPProxySpec{
			Includes: grpcIncludes,
		},
		Status: contourv1.HTTPProxyStatus{},
	}

	if httpExists {
		httpProxy.SetResourceVersion(httpResourceVersion)
		err := s.kubeClient.Update(ctx, &httpProxy)
		if err != nil {
			s.logger.Error("failed to update http proxy", zap.Error(err), zap.Any("httpProxy", httpProxy))
			return err
		}
	} else {
		err := s.kubeClient.Create(ctx, &httpProxy)
		if err != nil {
			s.logger.Error("failed to create http proxy", zap.Error(err), zap.Any("httpProxy", httpProxy))
			return err
		}
	}

	if grpcExists {
		grpcProxy.SetResourceVersion(grpcResourceVersion)
		err := s.kubeClient.Update(ctx, &grpcProxy)
		if err != nil {
			s.logger.Error("failed to update grpc proxy", zap.Error(err), zap.Any("grpcProxy", grpcProxy))
			return err
		}
	} else {
		err := s.kubeClient.Create(ctx, &grpcProxy)
		if err != nil {
			s.logger.Error("failed to create grpc proxy", zap.Error(err), zap.Any("grpcProxy", grpcProxy))
			return err
		}
	}
	return nil
}
