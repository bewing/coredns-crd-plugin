package k8scrd

/*
Copyright 2022 The k8gb Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Generated by GoLic, for more details see: https://github.com/AbsaOSS/golic
*/

import (
	"context"
	"fmt"

	"github.com/AbsaOSS/k8s_crd/common/k8sctrl"

	"github.com/AbsaOSS/k8s_crd/service"
	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
)

type K8sCRD struct {
	Next       plugin.Handler
	Filter     string
	controller *k8sctrl.KubeController
	container  service.PluginContainer
}

func NewK8sCRD(ct configType, filter string) (*K8sCRD, error) {
	cfg, rct, err := configFactory(ct)
	if err != nil {
		return nil, err
	}
	log.Infof("Running '%s' kube controller", rct)
	ctrl, err := RunKubeController(context.Background(), cfg, filter)
	if err != nil {
		return nil, err
	}
	return &K8sCRD{
		container:  service.NewCommonContainer(),
		controller: ctrl,
	}, nil
}

// ServeDNS implements the plugin.Handle interface.
func (p *K8sCRD) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	if !p.controller.HasSynced() {
		// TODO maybe there's a better way to do this? e.g. return an error back to the client?
		return dns.RcodeServerFailure, plugin.Error(thisPlugin, fmt.Errorf("could not sync required resources"))
	}

	err := p.container.Execute(ctx, w, r)
	if err != nil {
		log.Errorf("%s", err)
		return dns.RcodeServerFailure, err
	}
	return dns.RcodeSuccess, nil
}

func (p *K8sCRD) Name() string {
	return thisPlugin
}