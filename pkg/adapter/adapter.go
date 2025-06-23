/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2024 HPC-Gridware GmbH
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
*
************************************************************************/
/*___INFO__MARK_END__*/

package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

const name = "go.hpc-gridware.com/example/qconf"

var (
	tracer    = otel.Tracer(name)
	meter     = otel.Meter(name)
	logger    = otelslog.NewLogger(name)
	startTime = time.Now()
)

func init() {

}

// Usage
// router := mux.NewRouter()
// router.Handle("/api/v0/command", adapter.NewAdapter(qconf)).Methods("POST")

// A JSON request body is expected with the following structure:
// {
// 	"method": "<method name>",
// 	"args": [
// 		"arg1",
// 		"arg2",
// 		...
// 	]
// }

type CommandRequest struct {
	MethodName string            `json:"method"`
	Args       []json.RawMessage `json:"args"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Uptime    string `json:"uptime"`
}

type MethodInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parameters  []string `json:"parameters"`
	ReturnType  string   `json:"return_type"`
}

type MethodsResponse struct {
	Methods []MethodInfo `json:"methods"`
	Count   int          `json:"count"`
}

// NewAdapter creates an http.Handler for any Go interface.
// The method name and arguments are expected in the JSON request body.
// The response is the return value of the method also in JSON format.
// The arguments and the return values must have a JSON serializable format.
// Only 1 or 2 return values are supported. In case of an error of the
// executed function an http status code 500 is returned.
//
// The adapter uses OpenTelemetry to trace the method calls and log the errors.
func NewAdapter(instance interface{}) http.Handler {
	loggerProvider, err := newLoggerProvider()
	if err != nil {
		panic(err)
	}
	global.SetLoggerProvider(loggerProvider)

	tracerProvider, err := newTraceProvider()
	if err != nil {
		panic(err)
	}
	otel.SetTracerProvider(tracerProvider)

	meterProvider, err := newMeterProvider()
	if err != nil {
		panic(err)
	}
	otel.SetMeterProvider(meterProvider)

	return &adapter{
		instance: instance,
	}
}

type adapter struct {
	instance interface{}
}

func (a *adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "request")
	defer span.End()

	// Handle GET endpoints without timeout
	if r.Method == "GET" {
		a.handleGetEndpoints(w, r)
		return
	}

	// Only POST method requires timeout and JSON parsing
	if r.Method != "POST" {
		a.fail(ctx, w, r, http.StatusMethodNotAllowed, "Method not allowed. Use POST for commands or GET for health/methods", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logErr := fmt.Errorf("invalid request payload: %w", err)
		a.fail(ctx, w, r, http.StatusBadRequest, logErr.Error(), err)
		return
	}

	logger.InfoContext(ctx, "request", "method", req.MethodName)

	method := reflect.ValueOf(a.instance).MethodByName(req.MethodName)
	if !method.IsValid() {
		logErr := fmt.Errorf("method not found: %s", req.MethodName)
		a.fail(ctx, w, r, http.StatusNotFound, logErr.Error(), nil)
		return
	}

	methodType := method.Type()
	if methodType.NumIn() != len(req.Args) {
		logErr := fmt.Errorf("invalid number of arguments for method %s: %d but should be %d",
			req.MethodName, len(req.Args), methodType.NumIn())
		a.fail(ctx, w, r, http.StatusBadRequest,
			logErr.Error(), nil)
		return
	}

	args := make([]reflect.Value, len(req.Args))
	for i, arg := range req.Args {
		argType := methodType.In(i)
		argValue := reflect.New(argType).Interface()
		if err := json.Unmarshal(arg, argValue); err != nil {
			a.fail(ctx, w, r, http.StatusBadRequest,
				fmt.Sprintf("Invalid argument %d", i), err)
			return
		}
		args[i] = reflect.Indirect(reflect.ValueOf(argValue))
	}

	qconfCallValueAttr := attribute.String("qconf.command", req.MethodName)
	span.SetAttributes(qconfCallValueAttr)
	//requestCounter.Add(ctx, 1, metric.WithAttributes(qconfCallValueAttr))

	results := method.Call(args)
	if len(results) > 1 {
		if err, ok := results[1].Interface().(error); ok && err != nil {
			logErr := fmt.Errorf("method call %s failed: %w", req.MethodName,
				results[1].Interface().(error))
			a.fail(ctx, w, r, http.StatusInternalServerError, logErr.Error(), err)
			return
		}
	}

	if len(results) > 0 {
		// check if the result is an error
		if err, ok := results[0].Interface().(error); ok && err != nil {
			logErr := fmt.Errorf("method call %s failed: %w", req.MethodName,
				results[0].Interface().(error))
			a.fail(ctx, w, r, http.StatusInternalServerError, logErr.Error(),
				results[0].Interface().(error))
			return
		}

		// if there is no error, encode the result
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(results[0].Interface()); err != nil {
			logErr := fmt.Errorf("failed to encode response for method %s: %w",
				req.MethodName, err)
			a.fail(ctx, w, r, http.StatusInternalServerError,
				logErr.Error(), err)
			return
		}
	}
	logger.InfoContext(ctx, "request successfully processed", "method", req.MethodName)
}

func (a *adapter) handleGetEndpoints(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	switch r.URL.Path {
	case "/health":
		a.handleHealth(w, r)
	case "/methods":
		a.handleMethods(w, r)
	default:
		response := map[string]string{"error": "Not found. Available GET endpoints: /health, /methods"}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
	}
}

func (a *adapter) handleHealth(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(startTime)
	health := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    uptime.String(),
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

func (a *adapter) handleMethods(w http.ResponseWriter, r *http.Request) {
	instanceValue := reflect.ValueOf(a.instance)
	instanceType := instanceValue.Type()
	
	var methods []MethodInfo
	for i := 0; i < instanceType.NumMethod(); i++ {
		method := instanceType.Method(i)
		methodType := method.Type
		
		// Get parameter types
		var params []string
		for j := 1; j < methodType.NumIn(); j++ { // Skip receiver (index 0)
			params = append(params, methodType.In(j).String())
		}
		
		// Get return type
		var returnType string
		if methodType.NumOut() > 0 {
			returnTypes := make([]string, methodType.NumOut())
			for j := 0; j < methodType.NumOut(); j++ {
				returnTypes[j] = methodType.Out(j).String()
			}
			returnType = strings.Join(returnTypes, ", ")
		} else {
			returnType = "void"
		}
		
		// Generate description based on method name
		description := a.generateMethodDescription(method.Name)
		
		methods = append(methods, MethodInfo{
			Name:        method.Name,
			Description: description,
			Parameters:  params,
			ReturnType:  returnType,
		})
	}
	
	response := MethodsResponse{
		Methods: methods,
		Count:   len(methods),
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (a *adapter) generateMethodDescription(methodName string) string {
	// Generate basic descriptions based on method name patterns
	switch {
	case strings.HasPrefix(methodName, "Show"):
		return "Retrieve information about " + strings.ToLower(strings.TrimPrefix(methodName, "Show"))
	case strings.HasPrefix(methodName, "Get"):
		return "Get " + strings.ToLower(strings.TrimPrefix(methodName, "Get"))
	case strings.HasPrefix(methodName, "Add"):
		return "Add a new " + strings.ToLower(strings.TrimPrefix(methodName, "Add"))
	case strings.HasPrefix(methodName, "Delete"):
		return "Delete " + strings.ToLower(strings.TrimPrefix(methodName, "Delete"))
	case strings.HasPrefix(methodName, "Modify"):
		return "Modify " + strings.ToLower(strings.TrimPrefix(methodName, "Modify"))
	case strings.HasPrefix(methodName, "Apply"):
		return "Apply " + strings.ToLower(strings.TrimPrefix(methodName, "Apply"))
	default:
		return "Execute " + methodName + " operation"
	}
}

func (a *adapter) fail(ctx context.Context, w http.ResponseWriter, r *http.Request, status int, message string, err error) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{"error": message}
	logger.InfoContext(ctx, message, "URL", r.URL.Path)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.ErrorContext(ctx, "Failed to encode error response", "error", err)
	}
	// Removed duplicate error writing - JSON response already contains the error
}

func newLoggerProvider() (*log.LoggerProvider, error) {
	logExporter, err := stdoutlog.New()
	if err != nil {
		return nil, err
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
	)
	return loggerProvider, nil
}

func newTraceProvider() (*trace.TracerProvider, error) {
	traceExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			trace.WithBatchTimeout(time.Second)),
	)
	return traceProvider, nil
}

func newMeterProvider() (*metric.MeterProvider, error) {
	metricExporter, err := stdoutmetric.New()
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			// Default is 1m. Set to 10s for demonstrative purposes.
			metric.WithInterval(10*time.Second))),
	)
	return meterProvider, nil
}
