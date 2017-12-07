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

	"github.com/spf13/viper"
  "github.com/opentracing/opentracing-go"
  flags "github.com/northwesternmutual/kanali/pkg/flags"
  tags "github.com/northwesternmutual/kanali/pkg/tags"
  kanaliError "github.com/northwesternmutual/kanali/pkg/errors"
  utils "github.com/northwesternmutual/kanali/pkg/utils"
  metrics "github.com/northwesternmutual/kanali/pkg/metrics"
  traffic "github.com/northwesternmutual/kanali/pkg/traffic"
  "github.com/northwesternmutual/kanali/pkg/logging"
  "github.com/northwesternmutual/kanali/pkg/apis/kanali.io/v2"
  options "github.com/northwesternmutual/kanali/cmd/kanali/app/options"
  store "github.com/northwesternmutual/kanali/pkg/store/kanali/v2"
)

func init() {
	options.KanaliOptions.Add(
		flagPluginsAPIKeyHeaderKey,
	)
  logging.Init(nil) // change me
  etcdCtlr, _ = traffic.NewTrafficController()
}

var (
	flagPluginsAPIKeyHeaderKey = flags.Flag{
		Long:  "plugins.apiKey.header_key",
		Short: "",
		Value: "apikey",
		Usage: "Name of the HTTP header holding the apikey.",
	}
)

var etcdCtlr *traffic.TrafficController

// APIKeyFactory is factory that implements the Plugin interface
type ApiKeyFactory struct{}

// OnRequest intercepts a request before it get proxied to an upstream service
func (k ApiKeyFactory) OnRequest(ctx context.Context, config map[string]string, m *metrics.Metrics, p v2.ApiProxy, r *http.Request, span opentracing.Span) error {

  // create a contextual logger for this method
  logger := logging.WithContext(ctx)
  currTime := time.Now()

	// do not preform API key validation if a request is made using the OPTIONS http method
	if strings.ToUpper(r.Method) == "OPTIONS" {
		logger.Debug("api key validation will not be preformed on http OPTIONS requests")
		return nil
	}

	// extract the api key header
	apiKey := r.Header.Get(viper.GetString(flagPluginsAPIKeyHeaderKey.GetLong()))
	if len(apiKey) < 1 {
		m.Add(metrics.Metric{tags.KanaliApiKeyName, "unknown", true})
		return &kanaliError.StatusError{http.StatusUnauthorized, errors.New("apikey not found in request")}
	}

	// attempt to find a matching api key
	key := store.ApiKeyStore().Get(apiKey)
	if key == nil {
		m.Add(metrics.Metric{tags.KanaliApiKeyName, "unknown", true})
		return &kanaliError.StatusError{http.StatusUnauthorized, errors.New("apikey not found in k8s cluster")}
	}
	span.SetTag(tags.KanaliApiKeyName, key.ObjectMeta.Name)
	m.Add(metrics.Metric{tags.KanaliApiKeyName, key.ObjectMeta.Name, true})

	rule, rate := store.ApiKeyBindingStore().Get(p.ObjectMeta.Namespace, config["apiKeyBindingName"], key.ObjectMeta.Name, utils.ComputeTargetPath(p.Spec.Source.Path, p.Spec.Target.Path, r.URL.Path))
	if rule == nil {
		return &kanaliError.StatusError{http.StatusUnauthorized, errors.New("no binding found for associated ApiProxy")}
	}
	span.SetTag(tags.KanaliApiKeyBindingName, config["apiKeyBindingName"])
	span.SetTag(tags.KanaliApiKeyBindingNamespace, p.ObjectMeta.Namespace)

	// validate api key
	if !validateAPIKey(*rule, r.Method) {
		return &kanaliError.StatusError{http.StatusUnauthorized, errors.New("api key unauthorized")}
	}

	if store.TrafficStore().IsRateLimitViolated(&p, rate, key.ObjectMeta.Name, currTime) {
		return &kanaliError.StatusError{http.StatusTooManyRequests, errors.New("rate limit exceeded")}
	}

  go etcdCtlr.ReportTraffic(ctx, &store.TrafficPoint{
    Time: currTime.UnixNano(),
    Namespace: p.ObjectMeta.Namespace,
    ProxyName: config["apiKeyBindingName"],
    KeyName: key.ObjectMeta.Name,
  })
	return nil

}

// OnResponse intercepts a request after it has been proxied to an upstream service
// but before the response gets returned to the client
func (k ApiKeyFactory) OnResponse(ctx context.Context, m *metrics.Metrics, p v2.ApiProxy, r *http.Request, resp *http.Response, span opentracing.Span) error {
	return nil
}

// validateAPIKey will return true if the given api key
// is authorized to make the given request.
// Global rule valudation will be given priority over
// granular rule validation
func validateAPIKey(rule v2.Rule, method string) bool {
	return rule.Global || validateGranularRules(method, rule.Granular)
}

// check to see wheather a given HTTP method can be found
// in the list of HTTP methods belonging to a spec.GranularProxy
func validateGranularRules(method string, rule v2.GranularProxy) bool {
	if len(rule.Verbs) < 1 {
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
var Plugin ApiKeyFactory
