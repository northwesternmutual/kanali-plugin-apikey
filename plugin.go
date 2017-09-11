// Copyright (c) 2017 Northwestern Mutual.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// package
package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/northwesternmutual/kanali/controller"
  "github.com/northwesternmutual/kanali/config"
	"github.com/northwesternmutual/kanali/metrics"
	"github.com/northwesternmutual/kanali/server"
	"github.com/northwesternmutual/kanali/spec"
	"github.com/northwesternmutual/kanali/utils"
	"github.com/opentracing/opentracing-go"
	"github.com/spf13/viper"
)

func init() {
	config.Flags.Add(
		flagPluginsAPIKeyHeaderKey,
	)
}

var (
	flagPluginsAPIKeyHeaderKey = config.Flag{
		Long:  "plugin.apiKey.header_key",
		Short: "",
		Value: "apikey",
		Usage: "Name of the HTTP header holding the apikey.",
	}
)

// APIKeyFactory is factory that implements the Plugin interface
type APIKeyFactory struct{}

// OnRequest intercepts a request before it get proxied to an upstream service
func (k APIKeyFactory) OnRequest(ctx context.Context, m *metrics.Metrics, p spec.APIProxy, c controller.Controller, r *http.Request, span opentracing.Span) error {

	// do not preform API key validation if a request is made using the OPTIONS http method
	if strings.ToUpper(r.Method) == "OPTIONS" {
		logrus.Debug("API key validation will not be preformed on HTTP OPTIONS requests")
		return nil
	}

	// extract the api key header
	apiKey := r.Header.Get(viper.GetString(flagPluginsAPIKeyHeaderKey.GetLong()))
	if apiKey == "" {
		m.Add(metrics.Metric{"api_key_name", "unknown", true})
		m.Add(metrics.Metric{"api_key_namespace", "unknown", true})
		return &utils.StatusError{http.StatusUnauthorized, errors.New("apikey not found in request")}
	}

	// attempt to find a matching api key
	keyStore := spec.KeyStore
	untypedKey, err := keyStore.Get(apiKey)
	if err != nil || untypedKey == nil {
		m.Add(metrics.Metric{"api_key_name", "unknown", true})
		m.Add(metrics.Metric{"api_key_namespace", "unknown", true})
		return &utils.StatusError{http.StatusUnauthorized, errors.New("apikey not found in k8s cluster")}
	}

	key, ok := untypedKey.(spec.APIKey)
	if !ok {
		m.Add(metrics.Metric{"api_key_name", "unknown", true})
		m.Add(metrics.Metric{"api_key_namespace", "unknown", true})
		return &utils.StatusError{http.StatusUnauthorized, errors.New("apikey not found in k8s cluster")}
	}

	span.SetTag("kanali.api_key_name", key.ObjectMeta.Name)
	span.SetTag("kanali.api_key_namespace", key.ObjectMeta.Namespace)

	m.Add(metrics.Metric{"api_key_name", key.ObjectMeta.Name, true})
	m.Add(metrics.Metric{"api_key_namespace", key.ObjectMeta.Namespace, true})

	bindingsStore := spec.BindingStore
	untypedBinding, err := bindingsStore.Get(p.ObjectMeta.Name, p.ObjectMeta.Namespace)
	if err != nil || untypedBinding == nil {
		return &utils.StatusError{http.StatusUnauthorized, errors.New("no binding found for associated APIProxy")}
	}
	binding, ok := untypedBinding.(spec.APIKeyBinding)
	if !ok {
		return &utils.StatusError{http.StatusUnauthorized, errors.New("no binding found for associated APIProxy")}
	}

	span.SetTag("kanali.api_binding_name", binding.ObjectMeta.Name)
	span.SetTag("kanali.api_binding_namespace", binding.ObjectMeta.Namespace)

	keyObj := binding.GetAPIKey(key.ObjectMeta.Name)
	if keyObj == nil {
		return &utils.StatusError{http.StatusUnauthorized, errors.New("api key not authorized for this proxy")}
	}

	rule := keyObj.GetRule(utils.ComputeTargetPath(p.Spec.Path, p.Spec.Target, r.URL.Path))

	// validate api key
	if !validateAPIKey(rule, r.Method) {
		return &utils.StatusError{http.StatusUnauthorized, errors.New("api key unauthorized")}
	}

	if spec.TrafficStore.IsQuotaViolated(binding, key.ObjectMeta.Name) {
		return &utils.StatusError{http.StatusTooManyRequests, errors.New("quota limit reached. please contact your administrator")}
	}

	if spec.TrafficStore.IsRateLimitViolated(binding, key.ObjectMeta.Name, time.Now()) {
		time.Sleep(2 * time.Second)
	}

	go server.Emit(binding, key.ObjectMeta.Name, time.Now())
	return nil

}

// OnResponse intercepts a request after it has been proxied to an upstream service
// but before the response gets returned to the client
func (k APIKeyFactory) OnResponse(ctx context.Context, m *metrics.Metrics, p spec.APIProxy, c controller.Controller, r *http.Request, resp *http.Response, span opentracing.Span) error {

	return nil

}

// validateAPIKey will return true if the given api key
// is authorized to make the given request.
// Global rule valudation will be given priority over
// granular rule validation
func validateAPIKey(rule spec.Rule, method string) bool {

	return rule.Global || validateGranularRules(method, rule.Granular)

}

// check to see wheather a given HTTP method can be found
// in the list of HTTP methods belonging to a spec.GranularProxy
func validateGranularRules(method string, rule *spec.GranularProxy) bool {
	if rule == nil {
		return false
	}
	for _, verb := range rule.Verbs {
		if strings.ToUpper(verb) == strings.ToUpper(method) {
			return true
		}
	}
	return false
}

// Plugin can be discovered by golang plugin package
var Plugin APIKeyFactory
