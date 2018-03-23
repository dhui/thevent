// Package thevent provides a typed hierarchical event system
package thevent

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

var (
	ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errType = reflect.TypeOf((*error)(nil)).Elem()
)

// Data is data to be sent with an Event when it's dispatched
type Data interface{}

// Handler is a function that handles/subscribes/listens to an event and should only take 2 parameters
// The first parameter should be a context.Context and the second parameter should have the same data type as
// the Event being handled.
//
// A handler should have the following function signature:
//      func(ctx context.Context, data interface{}) error
type Handler interface{}

// Event is used to represent an event which may be handled and dispatched
type Event struct {
	dataType    reflect.Type
	handlerType reflect.Type

	// Not using sync.Map since we need to protect 2 fields at the same time. Also, by not using sync.Map,
	// we get compile-time type checks
	lock *sync.RWMutex

	// Must use reflect.Value to represent a handler since func(int) != func(interface{})
	// e.g. the empty interface has it's own distinct type. https://golang.org/ref/spec#Type_identity
	handlers map[uintptr]reflect.Value
	children map[*Event]*reflect.StructField
}

// HandlersResults contains the results of handlers handling a dispatched event
type HandlersResults struct {
	NumHandlers uint
	// Errors contains all of the non-nil errors returned by Handlers
	Errors []error
}

// Erred returns true if any Handler for the Event erred
func (r *HandlersResults) Erred() bool {
	return len(r.Errors) > 0
}

// ErrorRate returns the error rate of handlers' for a dispatched event. An error rate of 0.0 means that no errors
// occured and an error rate of 1.0 means that every handler errored
func (r *HandlersResults) ErrorRate() float32 {
	if r.NumHandlers <= 0 {
		return 0.0
	}
	return float32(len(r.Errors)) / float32(r.NumHandlers)
}

// Collect updates the given HandlersResults with the given error channel.
// Designed to be used with Event.DispatchAsyncWithErrors()
func (r *HandlersResults) Collect(ch <-chan error) {
	for err := range ch {
		r.NumHandlers++
		if err != nil {
			r.Errors = append(r.Errors, err)
		}
	}
}

func convertToError(results []reflect.Value) error {
	if len(results) != 1 {
		return TypeError{fmt.Errorf("Expected handler to return a single value, not %d", len(results))}
	}
	res := results[0].Interface()
	if res == nil {
		return nil
	}
	err, ok := res.(error)
	if !ok {
		return TypeError{fmt.Errorf("Expected handler to return an error type, not: %T", res)}
	}
	return err
}

func (r *HandlersResults) addResult(results []reflect.Value) error {
	err := convertToError(results)
	if _, ok := err.(TypeError); ok {
		return err
	}
	r.NumHandlers++
	if err != nil {
		r.Errors = append(r.Errors, err)
	}
	return nil
}

func (e *Event) dispatch(ctx context.Context, async bool, trackResults bool,
	data interface{}) (*HandlersResults, <-chan error, error) {
	dataValue := reflect.ValueOf(data)
	dataType := dataValue.Type()
	if dataType != e.dataType {
		return nil, nil, TypeError{fmt.Errorf("Dispatch called with incorrect event data type. Expected: %s Got: %s",
			e.dataType.String(), dataType.String())}
	}
	args := []reflect.Value{reflect.ValueOf(ctx), dataValue}

	var results HandlersResults
	wg := sync.WaitGroup{}
	var errorsCh chan error
	if async && trackResults {
		errorsCh = make(chan error)
		defer func() {
			go func() {
				wg.Wait()
				close(errorsCh)
			}()
		}()
	}
	var errs MultiTypeError

	e.lock.RLock()
	defer e.lock.RUnlock()
	// Fine to hold onto read lock while handlers and all sub-Event handlers run
	for _, h := range e.handlers {
		if async {
			wg.Add(1)
			go func(_h reflect.Value) {
				defer wg.Done()
				res := _h.Call(args)
				if trackResults {
					err := convertToError(res)
					errorsCh <- err
				}
			}(h)
		} else {
			res := h.Call(args)
			if trackResults {
				if err := results.addResult(res); err != nil {
					e, ok := err.(TypeError)
					if ok {
						errs = append(errs, e)
					} else {
						errs = append(errs,
							TypeError{fmt.Errorf("Got unexpected error running handler: %v", err)})
					}
				}
			}
		}
	}
	// Dispatch children after the parents
	for subEvent, field := range e.children {
		dataForChild := data // default to same event data as parent
		if field != nil {
			// Use reflection to populate the child struct w/ the parent event data
			subDataPtr := reflect.New(subEvent.dataType)
			subDataStruct := subDataPtr.Elem()
			f := subDataStruct.FieldByIndex(field.Index)
			if !f.IsValid() {
				return nil, nil, TypeError{
					fmt.Errorf("Sub-Event: %s data type changed. Unable to get field with name: %s",
						subEvent.dataType.String(), field.Name)}
			}
			if !f.CanSet() {
				return nil, nil, TypeError{fmt.Errorf("Unable to set field %s for sub-Event: %s", field.Name,
					subEvent.dataType.String())}
			}
			if f.Kind() == reflect.Ptr {
				if dataValue.CanAddr() {
					f.Set(dataValue.Addr())
				} else {
					// copy parent event struct data over
					c := reflect.New(dataType)
					c.Elem().Set(dataValue)
					f.Set(c)
				}
			} else {
				// copy parent event struct data over
				f.Set(dataValue)
			}
			dataForChild = subDataStruct.Interface()
		}
		// RWMutexes aren't re-entrant but we don't have this problem since each sub-Event has its own RWMutex
		res, ch, err := subEvent.dispatch(ctx, async, trackResults, dataForChild)
		if err != nil {
			e, ok := err.(TypeError)
			if ok {
				errs = append(errs, e)
			} else {
				errs = append(errs,
					TypeError{fmt.Errorf("Got unexpected error running handler: %v", err)})
			}
		}
		if trackResults {
			// propagate sub-Event results
			if async {
				for e := range ch {
					errorsCh <- e
				}
			} else {
				results.NumHandlers += res.NumHandlers
				results.Errors = append(results.Errors, res.Errors...)
			}
		}
	}
	if async && trackResults {
		return nil, errorsCh, nil
	}
	if len(errs) > 0 {
		return nil, errorsCh, TypeError{errs}
	}
	return &results, nil, nil
}

// Dispatch will notify all handlers of the Event and sub-Events using depth-first pre-order traversal.
// Dispatch will not return until all Event and sub-Event handlers have finished running. Any errors encountered
// which dispatching a
func (e *Event) Dispatch(ctx context.Context, data interface{}) error {
	_, _, err := e.dispatch(ctx, false, false, data)
	return err
}

// DispatchWithResults is the same as Dispatch but collects the results
func (e *Event) DispatchWithResults(ctx context.Context, data interface{}) (*HandlersResults, error) {
	res, _, err := e.dispatch(ctx, false, true, data)
	return res, err
}

// DispatchAsync will asynchronously notify all handlers of the Event and sub-Events. All handlers may not be
// finished running when DispatchAsync returns.
func (e *Event) DispatchAsync(ctx context.Context, data interface{}) error {
	_, _, err := e.dispatch(ctx, true, false, data)
	return err
}

// DispatchAsyncWithErrors is the same as DispatchAsync but additionally provides a channel that streams the
// returned error from every handler for the event. It's the caller's responsibility to range over the channel as
// the channel will be closed when all handlers are finished running. Not ranging over the returned channel will
// leave dangling handlers. To "join" all of the errors use, HandlersResults.Collect().
func (e *Event) DispatchAsyncWithErrors(ctx context.Context, data interface{}) (<-chan error, error) {
	_, ch, err := e.dispatch(ctx, true, true, data)
	return ch, err
}

// AddHandlers adds the Handlers to the Event
func (e *Event) AddHandlers(handlers ...Handler) error {
	convertedHandlers := make(map[uintptr]reflect.Value, len(handlers))
	for _, h := range handlers {
		hV := reflect.ValueOf(h)
		hT := hV.Type()
		if hT != e.handlerType {
			return TypeError{fmt.Errorf("Handler uses incorrect data type. Expected: %s Got: %s",
				e.handlerType.String(), hT.String())}
		}
		if _, ok := convertedHandlers[hV.Pointer()]; ok {
			return TypeError{errors.New("Unable to add duplicate handler")}
		}
		convertedHandlers[hV.Pointer()] = hV
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	for _, cH := range convertedHandlers {
		if _, ok := e.handlers[cH.Pointer()]; ok {
			return TypeError{errors.New("Unable to add duplicate handler")}
		}
	}
	for _, cH := range convertedHandlers {
		e.handlers[cH.Pointer()] = cH
	}
	return nil
}

// New creates a new sub-Event that's also dispatched whenever the "parent" Event is dispatched.
//
// data must be a struct which either:
//   - is the same as the parent Event's data (fieldName should be an empty string)
//   - has a field with the parent Event's data specified by the fieldName
func (e *Event) New(data interface{}, fieldName string, handlers ...Handler) (*Event, error) {
	if e.dataType.Kind() != reflect.Struct {
		return nil, TypeError{fmt.Errorf("New() can only be used on Events with event type: %s, not %s",
			reflect.Struct.String(), e.dataType.Kind().String())}
	}
	dataType := reflect.TypeOf(data)
	if dataType.Kind() != reflect.Struct {
		return nil, TypeError{fmt.Errorf("data type must be a %s, not %s",
			reflect.Struct.String(), dataType.Kind().String())}
	}
	var matchedField *reflect.StructField

	if fieldName != "" {
		f, ok := dataType.FieldByName(fieldName)
		if !ok {
			return nil, TypeError{fmt.Errorf("No such field with name: %s in data", fieldName)}
		}
		if f.Type != e.dataType && f.Type != reflect.PtrTo(e.dataType) {
			return nil, TypeError{fmt.Errorf("Field with name: %s has wrong type: %s. Should be: %s",
				fieldName, f.Type.String(), e.dataType.String())}
		}
		if f.PkgPath != "" {
			return nil, TypeError{fmt.Errorf("Field with name: %s has correct data type but must be exported",
				fieldName)}
		}
		matchedField = &f
	} else if dataType != e.dataType { // && dataType != reflect.PtrTo(e.dataType) {
		return nil, TypeError{fmt.Errorf("sub-Event's data type (%s) doesn't match parent's (%s)", dataType.String(),
			e.dataType.String())}
	}

	subEvent, err := New(data, handlers...)
	if err != nil {
		return nil, err
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	e.children[subEvent] = matchedField
	return subEvent, nil
}

// New creates a new Event
//
// data is a sample of the event Data that handlers will receive. The empty/zero value of the event Data
// should be used.
func New(data interface{}, handlers ...Handler) (*Event, error) {
	dataType := reflect.TypeOf(data)
	handlerType := reflect.FuncOf([]reflect.Type{ctxType, dataType}, []reflect.Type{errType}, false)
	event := &Event{dataType: dataType, handlerType: handlerType, lock: &sync.RWMutex{},
		handlers: make(map[uintptr]reflect.Value, len(handlers)),
		children: map[*Event]*reflect.StructField{}}
	if err := event.AddHandlers(handlers...); err != nil {
		return nil, err
	}
	return event, nil
}

// Must is a helper to be used with New() and Event.New() that converts the error to a panic.
//
// Example:
//     type eventData struct{}
//     type childEventData struct{event}
//     parentEvent := Must(New(eventData{}))
//     childEvent := Must(parentEvent.New(childEventData{}, "eventData"))
func Must(e *Event, err error) *Event {
	if err != nil {
		panic(err)
	}
	return e
}
