package statemanager

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/internal/helm"
	contourv1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

func (s *Service) syncSupersetHttpProxy(ctx context.Context, workspace *db.Workspace) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	httpKey := types.NamespacedName{
		Name:      fmt.Sprintf("proxy-ss-%s", workspace.ID),
		Namespace: s.cfg.KaytuOctopusNamespace,
	}
	var httpProxy contourv1.HTTPProxy

	httpExists := true
	if err := s.kubeClient.Get(ctx, httpKey, &httpProxy); err != nil {
		if apierrors.IsNotFound(err) {
			httpExists = false
		} else {
			s.logger.Error("failed to get http proxy", zap.Error(err))
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
			Name:      fmt.Sprintf("proxy-ss-%s", workspace.ID),
			Namespace: s.cfg.KaytuOctopusNamespace,
		},
		Spec: contourv1.HTTPProxySpec{
			Includes: []contourv1.Include{
				{
					Name:      "http-proxy-route-ss",
					Namespace: workspace.ID,
					Conditions: []contourv1.MatchCondition{
						{
							Prefix: "/",
							Header: nil,
						},
					},
				},
			},
			VirtualHost: &contourv1.VirtualHost{
				Fqdn: fmt.Sprintf("ss-%s.kaytu.io", workspace.ID),
				TLS: &contourv1.TLS{
					SecretName: "web-tls",
				},
				Authorization: nil,
				CORSPolicy: &contourv1.CORSPolicy{
					AllowCredentials: true,
					AllowOrigin:      []string{"*"},
					AllowMethods:     []contourv1.CORSHeaderValue{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
					AllowHeaders:     []contourv1.CORSHeaderValue{"authorization", "cache-control", "content-type", "data", "baggage"},
					ExposeHeaders:    []contourv1.CORSHeaderValue{"Content-Length", "Content-Range"},
					MaxAge:           "10m",
				},
				RateLimitPolicy: nil,
			},
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

	return nil
}

func (s *Service) syncHTTPProxy(ctx context.Context, workspaces []*db.Workspace) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var httpIncludes []contourv1.Include
	var grpcIncludes []contourv1.Include
	for _, w := range workspaces {
		if !(w.Status == api.StateID_Provisioned ||
			((w.Status == api.StateID_WaitingForCredential || w.Status == api.StateID_Provisioning) && w.IsCreated)) {
			continue
		}

		err := s.syncSupersetHttpProxy(ctx, w)
		if err != nil {
			return err
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

func (s *Service) ensureSettingsSynced(ctx context.Context, workspace db.Workspace) error {
	needsUpdate, settings, err := helm.GetUpToDateWorkspaceHelmValues(ctx, s.cfg, s.kubeClient, s.db, s.vault, workspace)
	if err != nil {
		return fmt.Errorf("get up to date workspace helm values: %w", err)
	}

	if !needsUpdate {
		s.logger.Debug("no need to update helm release", zap.String("workspace", workspace.ID))
		return nil
	}

	s.logger.Info("updating helm release", zap.String("workspace", workspace.ID))
	valuesJson, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	err = helm.UpdateHelmRelease(ctx, s.cfg, s.kubeClient, workspace, valuesJson)
	if err != nil {
		return fmt.Errorf("update helm release: %w", err)
	}

	return nil
}

func (s *Service) syncHelmValues(ctx context.Context, workspaces []*db.Workspace) error {
	for _, w := range workspaces {
		if !(w.Status == api.StateID_Provisioned ||
			((w.Status == api.StateID_WaitingForCredential || w.Status == api.StateID_Provisioning) && w.IsCreated)) {
			continue
		}

		err := s.ensureSettingsSynced(ctx, *w)
		if err != nil {
			return err
		}
	}
	return nil
}
