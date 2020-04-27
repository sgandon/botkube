// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package filters

import (
	"fmt"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	log "github.com/infracloudio/botkube/pkg/logging"
	networkV1beta1 "k8s.io/api/networking/v1beta1"
)

// IngressValidator checks if service and tls secret used in ingress specs is already present
// and adds recommendations to event struct accordingly
type IngressValidator struct {
	Description string
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(IngressValidator{
		Description: "Checks if services and tls secrets used in ingress specs are available.",
	})
}

// Run filers and modifies event struct
func (iv IngressValidator) Run(object interface{}, event *events.Event) {
	if event.Kind != "Ingress" || event.Type != config.CreateEvent {
		return
	}
	ingressObj, ok := object.(*networkV1beta1.Ingress)
	if !ok {
		return
	}

	ingNs := ingressObj.ObjectMeta.Namespace

	// Check if service names are valid in the configuration
	for _, rule := range ingressObj.Spec.Rules {
		for _, path := range rule.IngressRuleValue.HTTP.Paths {
			serviceName := path.Backend.ServiceName
			servicePort := path.Backend.ServicePort.IntValue()
			ns := FindNamespaceFromService(serviceName)
			if ns == "default" {
				ns = ingNs
			}
			_, err := ValidServicePort(serviceName, ns, int32(servicePort))
			if err != nil {
				event.Warnings = append(event.Warnings, fmt.Sprintf("Service '%s' used in ingress '%s' config does not exist or port '%v' not exposed", serviceName, ingressObj.Name, servicePort))
			}
		}
	}

	// Check if tls secret exists
	for _, tls := range ingressObj.Spec.TLS {
		_, err := ValidSecret(tls.SecretName, ingNs)
		if err != nil {
			event.Recommendations = append(event.Recommendations, fmt.Sprintf("TLS secret %s does not exist", tls.SecretName))
		}
	}
	log.Logger.Debug("Ingress Validator filter successful!")
}

// Describe filter
func (iv IngressValidator) Describe() string {
	return iv.Description
}
