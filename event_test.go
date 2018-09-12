package thevent_test

import (
	"context"
	"errors"
	"path"
	"testing"
)

import (
	"github.com/dhui/thevent"
)

type testStruct struct{ v int }
type testUnexportedEmbeddedStruct struct {
	testStruct
	wrong bool
}
type testUnexportedEmbeddedPtrStruct struct {
	*testStruct
	wrong bool
}
type testUnexportedNamedStruct struct {
	test  testStruct
	wrong bool
}
type testUnexportedNamedPtrStruct struct {
	test  *testStruct
	wrong bool
}
type testExportedNamedUnexportedStruct struct {
	Test  testStruct
	wrong bool
}
type testExportedNamedUnexportedPtrStruct struct {
	Test  *testStruct
	wrong bool
}

type TestStruct struct{ v int }
type testExportedEmbeddedStruct struct {
	TestStruct
	wrong bool
}
type testExportedEmbeddedPtrStruct struct {
	*TestStruct
	wrong bool
}
type testExportedNamedExportedStruct struct {
	Test  TestStruct
	wrong bool
}
type testExportedNamedExportedPtrStruct struct {
	Test  *TestStruct
	wrong bool
}

type unrelatedStruct struct{}

func intHandler(context.Context, int) error                       { return nil }
func testStructHandler(context.Context, testStruct) error         { return nil }
func exportedTestStructHandler(context.Context, TestStruct) error { return nil }
func embeddedTestStructHandler(context.Context, testExportedNamedUnexportedStruct) error {
	return nil
}

func errorMatchesGlob(t *testing.T, err error, glob string) {
	if glob == "" {
		if err != nil {
			t.Error("Got unexpected error:", err)
		}
	} else {
		if err == nil {
			t.Error("Didn't get an error as expected")
		} else {
			matched, e := path.Match(glob, err.Error())
			if e != nil {
				t.Error(e)
			}
			if !matched {
				t.Errorf("Got unexpected error: %q Expected error pattern: %q", err, glob)
			}
		}
	}
}

func TestNew(t *testing.T) {
	testCases := []struct {
		name      string
		data      thevent.Data
		handlers  []thevent.Handler
		errorGlob string
	}{
		// int event data
		{name: "int data - no handlers", data: 5},
		{name: "int data - non-function handler", data: 5, handlers: []thevent.Handler{5},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
		{name: "int data - valid handler", data: 5, handlers: []thevent.Handler{intHandler}},
		{name: "int data - mismatched handler", data: 5, handlers: []thevent.Handler{testStructHandler},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
		{name: "int data - valid and mismatched handler", data: 5,
			handlers:  []thevent.Handler{intHandler, testStructHandler},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
		// struct event data
		{name: "struct data - no handlers", data: testStruct{}},
		{name: "struct data - non-function handler", data: testStruct{},
			handlers:  []thevent.Handler{testStruct{}},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
		{name: "struct data - valid handler", data: testStruct{},
			handlers: []thevent.Handler{testStructHandler}},
		{name: "struct data - mismatched handler", data: testStruct{},
			handlers:  []thevent.Handler{intHandler},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
		{name: "struct data - valid and mismatched handler", data: testStruct{},
			handlers:  []thevent.Handler{testStructHandler, intHandler},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := thevent.New(tc.data, tc.handlers...)
			errorMatchesGlob(t, err, tc.errorGlob)
		})
	}
}

func TestAddHandlers(t *testing.T) {
	testCases := []struct {
		name      string
		handlers  []thevent.Handler
		errorGlob string
	}{
		{name: "no handlers"},
		{name: "non-function handler", handlers: []thevent.Handler{5},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
		{name: "valid handler", handlers: []thevent.Handler{testStructHandler}},
		{name: "duplicate valid handler", handlers: []thevent.Handler{testStructHandler,
			testStructHandler}, errorGlob: "Unable to add duplicate handler"},
		{name: "mismatched handler", handlers: []thevent.Handler{intHandler},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
		{name: "mismatched handler - diff struct",
			handlers:  []thevent.Handler{exportedTestStructHandler},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
		{name: "mismatched handler - embedded struct",
			handlers:  []thevent.Handler{embeddedTestStructHandler},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
		{name: "valid and mismatched handler",
			handlers:  []thevent.Handler{testStructHandler, intHandler},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e, err := thevent.New(testStruct{})
			if err != nil {
				t.Fatal("Unable to create event:", err)
			}

			err = e.AddHandlers(tc.handlers...)
			errorMatchesGlob(t, err, tc.errorGlob)
		})
	}

	// Test duplicate handler in subsequent AddHandlers() call
	e, err := thevent.New(5)
	if err != nil {
		t.Fatal("Unable to create event:", err)
	}

	if err := e.AddHandlers(intHandler); err != nil {
		t.Error("Unable to add valid handler")
	}
	err = e.AddHandlers(intHandler)
	errorMatchesGlob(t, err, "Unable to add duplicate handler")
}

func TestDispatch(t *testing.T) {
	e, err := thevent.New(5)
	if err != nil {
		t.Fatal("Unable to create event:", err)
	}

	asyncEvent, err := thevent.New(5)
	if err != nil {
		t.Fatal("Unable to create event:", err)
	}

	calledWith := -1
	calledHandler := func(ctx context.Context, i int) error { // nolint: unparam
		calledWith = i
		return nil
	}
	if err := e.AddHandlers(calledHandler); err != nil {
		t.Fatal("Unable to add handler to test event:", err)
	}
	handlerError := func(ctx context.Context, i int) error {
		return errors.New("handler always errors")
	}
	if err := e.AddHandlers(handlerError); err != nil {
		t.Fatal("Unable to add handler to test event:", err)
	}

	calledWithAsync := make(chan int)
	calledHandlerAsync := func(ctx context.Context, i int) error { // nolint: unparam
		calledWithAsync <- i
		return nil
	}
	if err := asyncEvent.AddHandlers(calledHandlerAsync); err != nil {
		t.Fatal("Unable to add handler to test async event:", err)
	}
	if err := asyncEvent.AddHandlers(handlerError); err != nil {
		t.Fatal("Unable to add handler to test async event:", err)
	}

	testCases := []struct {
		name         string
		data         thevent.Data
		errorGlob    string
		expectedData int
	}{
		{name: "wrong data type - float", data: 1.0,
			errorGlob: "Dispatch called with incorrect event data type. Expected: int Got: float*"},
		{name: "wrong data type - string", data: "",
			errorGlob: "Dispatch called with incorrect event data type. Expected: int Got: string"},
		{name: "wrong data type - struct", data: testStruct{},
			errorGlob: "Dispatch called with incorrect event data type. Expected: int Got: *"},
		{name: "valid data", data: 1, expectedData: 1},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("Dispatch", func(t *testing.T) {
				err := e.Dispatch(ctx, tc.data)
				errorMatchesGlob(t, err, tc.errorGlob)
				if tc.errorGlob == "" && calledWith != tc.expectedData {
					t.Error("Handler wasn't correctly notified by Dispatch.", calledWith, "!=", tc.expectedData)
				}
			})

			t.Run("DispatchWithResults", func(t *testing.T) {
				res, err := e.DispatchWithResults(ctx, tc.data)
				errorMatchesGlob(t, err, tc.errorGlob)
				if tc.errorGlob == "" && calledWith != tc.expectedData {
					t.Error("Handler wasn't correctly notified by Dispatch.", calledWith, "!=", tc.expectedData)
				}
				if err == nil {
					if res.NumHandlers != 2 {
						t.Error("2 handlers should have been dispatched, not", res.NumHandlers)
					}
					if len(res.Errors) != 1 {
						t.Error("Expected 1 handler error, instead have errors:", res.Errors)
					}
				}
			})

			t.Run("DispatchAsync", func(t *testing.T) {
				err := asyncEvent.DispatchAsync(ctx, tc.data)
				errorMatchesGlob(t, err, tc.errorGlob)
				if tc.errorGlob == "" {
					v := <-calledWithAsync
					if v != tc.expectedData {
						t.Error("Handler wasn't correctly notified by Dispatch.", v, "!=", tc.expectedData)
					}
				}
			})

			t.Run("DispatchAsyncWithResults", func(t *testing.T) {
				ch, err := asyncEvent.DispatchAsyncWithResults(ctx, tc.data)
				res := thevent.HandlersResults{}
				errorMatchesGlob(t, err, tc.errorGlob)
				if tc.errorGlob == "" {
					v := <-calledWithAsync
					if v != tc.expectedData {
						t.Error("Handler wasn't correctly notified by Dispatch.", v, "!=", tc.expectedData)
					}
				}
				if err == nil {
					res.Collect(ch)
					if res.NumHandlers != 2 {
						t.Error("2 handlers should have been dispatched, not", res.NumHandlers)
					}
					if len(res.Errors) != 1 {
						t.Error("Expected 1 handler error, instead have errors:", res.Errors)
					}
				}
			})
		})
	}
}

func TestNewSubEvent(t *testing.T) {
	nonStructDataEvent, err := thevent.New(5)
	if err != nil {
		t.Fatal("Unable to create event:", err)
	}

	unexportedStructDataEvent, err := thevent.New(testStruct{})
	if err != nil {
		t.Fatal("Unable to crate event:", err)
	}
	exportedStructDataEvent, err := thevent.New(TestStruct{})
	if err != nil {
		t.Fatal("Unable to crate event:", err)
	}

	type testCase struct {
		name      string
		data      thevent.Data
		fieldName string
		handlers  []thevent.Handler
		errorGlob string
	}
	unexportedTestCases := []testCase{
		// int event data
		{name: "int data", data: 5, errorGlob: "data type must be a struct, not int"},
		// unrelated struct event data
		{name: "unrelated struct data", data: unrelatedStruct{},
			errorGlob: `sub-Event's data type (thevent_test.unrelatedStruct) doesn't match parent's (thevent_test.testStruct)`},
		// unexported embedded struct data
		{name: "unexported embedded struct data - no field name", data: testUnexportedEmbeddedStruct{},
			errorGlob: "sub-Event's data type (thevent_test.testUnexportedEmbeddedStruct) doesn't match parent's (thevent_test.testStruct)"},
		{name: "unexported embedded struct data - non existing field name", data: testUnexportedEmbeddedStruct{},
			fieldName: "doesnotexist", errorGlob: "No such field with name: doesnotexist in data"},
		{name: "unexported embedded struct data - with incorrect field name",
			data: testUnexportedEmbeddedStruct{}, fieldName: "wrong",
			errorGlob: "Field with name: wrong has wrong type: bool. Should be: thevent_test.testStruct"},
		{name: "unexported embedded struct data - with correct field name", data: testUnexportedEmbeddedStruct{},
			fieldName: "testStruct",
			errorGlob: "Field with name: testStruct has correct data type but must be exported"},
		// unexported embedded ptr struct data
		{name: "unexported embedded ptr struct data - no field name", data: testUnexportedEmbeddedPtrStruct{},
			errorGlob: "sub-Event's data type (thevent_test.testUnexportedEmbeddedPtrStruct) doesn't match parent's (thevent_test.testStruct)"},
		{name: "unexported embedded ptr struct data - non existing field name",
			data: testUnexportedEmbeddedPtrStruct{}, fieldName: "doesnotexist",
			errorGlob: "No such field with name: doesnotexist in data"},
		{name: "unexported embedded ptr struct data - with incorrect field name",
			data: testUnexportedEmbeddedPtrStruct{}, fieldName: "wrong",
			errorGlob: "Field with name: wrong has wrong type: bool. Should be: thevent_test.testStruct"},
		{name: "unexported embedded ptr struct data - with correct field name",
			data: testUnexportedEmbeddedPtrStruct{}, fieldName: "testStruct",
			errorGlob: "Field with name: testStruct has correct data type but must be exported"},
		// unexported named struct data
		{name: "unexported named struct data - no field name", data: testUnexportedNamedStruct{},
			errorGlob: "sub-Event's data type (thevent_test.testUnexportedNamedStruct) doesn't match parent's (thevent_test.testStruct)"},
		{name: "unexported named struct data - non existing field name", data: testUnexportedNamedStruct{},
			fieldName: "doesnotexist", errorGlob: "No such field with name: doesnotexist in data"},
		{name: "unexported named struct data - with incorrect field name", data: testUnexportedNamedStruct{},
			fieldName: "wrong",
			errorGlob: "Field with name: wrong has wrong type: bool. Should be: thevent_test.testStruct"},
		{name: "unexported named struct data - with correct field name", data: testUnexportedNamedStruct{},
			fieldName: "test", errorGlob: "Field with name: test has correct data type but must be exported"},
		// unexported named ptr struct data
		{name: "unexported named ptr struct data - no field name", data: testUnexportedNamedPtrStruct{},
			errorGlob: "sub-Event's data type (thevent_test.testUnexportedNamedPtrStruct) doesn't match parent's (thevent_test.testStruct)"},
		{name: "unexported named ptr struct data - non existing field name", data: testUnexportedNamedPtrStruct{},
			fieldName: "doesnotexist", errorGlob: "No such field with name: doesnotexist in data"},
		{name: "unexported named ptr struct data - with incorrect field name",
			data: testUnexportedNamedPtrStruct{}, fieldName: "wrong",
			errorGlob: "Field with name: wrong has wrong type: bool. Should be: thevent_test.testStruct"},
		{name: "unexported named ptr struct data - with correct field name", data: testUnexportedNamedPtrStruct{},
			fieldName: "test", errorGlob: "Field with name: test has correct data type but must be exported"},
		// exported named unexported struct data
		{name: "exported named unexported struct data - no field name", data: testExportedNamedUnexportedStruct{},
			errorGlob: "sub-Event's data type (thevent_test.testExportedNamedUnexportedStruct) doesn't match parent's (thevent_test.testStruct)"},
		{name: "exported named unexported struct data - non existing field name",
			data: testExportedNamedUnexportedStruct{}, fieldName: "doesnotexist",
			errorGlob: "No such field with name: doesnotexist in data"},
		{name: "exported named unexported struct data - with incorrect field name",
			data: testExportedNamedUnexportedStruct{}, fieldName: "wrong",
			errorGlob: "Field with name: wrong has wrong type: bool. Should be: thevent_test.testStruct"},
		{name: "exported named unexported struct data - with correct field name",
			data: testExportedNamedUnexportedStruct{}, fieldName: "Test"},
		// exported named unexported ptr struct data
		{name: "exported named unexported ptr struct data - no field name",
			data:      testExportedNamedUnexportedPtrStruct{},
			errorGlob: "sub-Event's data type (thevent_test.testExportedNamedUnexportedPtrStruct) doesn't match parent's (thevent_test.testStruct)"},
		{name: "exported named unexported ptr struct data - non existing field name",
			data: testExportedNamedUnexportedPtrStruct{}, fieldName: "doesnotexist",
			errorGlob: "No such field with name: doesnotexist in data"},
		{name: "exported named unexported ptr struct data - with incorrect field name",
			data: testExportedNamedUnexportedPtrStruct{}, fieldName: "wrong",
			errorGlob: "Field with name: wrong has wrong type: bool. Should be: thevent_test.testStruct"},
		{name: "exported named unexported ptr struct data - with correct field name",
			data: testExportedNamedUnexportedPtrStruct{}, fieldName: "Test"},
		// same struct event data
		{name: "same struct data - no handlers", data: testStruct{}},
		{name: "same struct data - non-function handler", data: testStruct{},
			handlers:  []thevent.Handler{testStruct{}},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
		{name: "same struct data - valid handler", data: testStruct{},
			handlers: []thevent.Handler{testStructHandler}},
		{name: "same struct data - mismatched handler", data: testStruct{},
			handlers:  []thevent.Handler{intHandler},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
		{name: "same struct data - valid and mismatched handler", data: testStruct{},
			handlers:  []thevent.Handler{testStructHandler, intHandler},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
	}

	for _, tc := range unexportedTestCases {
		t.Run(tc.name, func(t *testing.T) {
			if e, err := nonStructDataEvent.New(tc.data, tc.fieldName, tc.handlers...); err == nil {
				t.Error("Created sub-Event with non struct data parent Event. Sub-Event:", e)
			}
			_, err := unexportedStructDataEvent.New(tc.data, tc.fieldName, tc.handlers...)
			errorMatchesGlob(t, err, tc.errorGlob)
		})
	}

	exportedTestCases := []testCase{
		// exported embedded struct data
		{name: "exported embedded struct data - no field name", data: testExportedEmbeddedStruct{},
			errorGlob: "sub-Event's data type (thevent_test.testExportedEmbeddedStruct) doesn't match parent's (thevent_test.TestStruct)"},
		{name: "exported embedded struct data - non existing field name", data: testExportedEmbeddedStruct{},
			fieldName: "doesnotexist", errorGlob: "No such field with name: doesnotexist in data"},
		{name: "exported embedded struct data - with incorrect field name", data: testExportedEmbeddedStruct{},
			fieldName: "wrong",
			errorGlob: "Field with name: wrong has wrong type: bool. Should be: thevent_test.TestStruct"},
		{name: "exported embedded struct data - with correct field name", data: testExportedEmbeddedStruct{},
			fieldName: "TestStruct"},
		// exported embedded ptr struct data
		{name: "exported embedded ptr struct data - no field name", data: testExportedEmbeddedPtrStruct{},
			errorGlob: "sub-Event's data type (thevent_test.testExportedEmbeddedPtrStruct) doesn't match parent's (thevent_test.TestStruct)"},
		{name: "exported embedded ptr struct data - non existing field name",
			data: testExportedEmbeddedPtrStruct{}, fieldName: "doesnotexist",
			errorGlob: "No such field with name: doesnotexist in data"},
		{name: "exported embedded ptr struct data - with incorrect field name",
			data: testExportedEmbeddedPtrStruct{}, fieldName: "wrong",
			errorGlob: "Field with name: wrong has wrong type: bool. Should be: thevent_test.TestStruct"},
		{name: "exported embedded ptr struct data - with correct field name",
			data: testExportedEmbeddedPtrStruct{}, fieldName: "TestStruct"},
		// exported named exported struct data
		{name: "exported named exported struct data - no field name", data: testExportedNamedExportedStruct{},
			errorGlob: "sub-Event's data type (thevent_test.testExportedNamedExportedStruct) doesn't match parent's (thevent_test.TestStruct)"},
		{name: "exported named exported struct data - non existing field name",
			data: testExportedNamedExportedStruct{}, fieldName: "doesnotexist",
			errorGlob: "No such field with name: doesnotexist in data"},
		{name: "exported named exported struct data - with incorrect field name",
			data: testExportedNamedExportedStruct{}, fieldName: "wrong",
			errorGlob: "Field with name: wrong has wrong type: bool. Should be: thevent_test.TestStruct"},
		{name: "exported named exported struct data - with correct field name",
			data: testExportedNamedExportedStruct{}, fieldName: "Test"},
		// exported named exported ptr struct data
		{name: "exported named exported ptr struct data - no field name",
			data:      testExportedNamedExportedPtrStruct{},
			errorGlob: "sub-Event's data type (thevent_test.testExportedNamedExportedPtrStruct) doesn't match parent's (thevent_test.TestStruct)"},
		{name: "exported named exported ptr struct data - non existing field name",
			data: testExportedNamedExportedPtrStruct{}, fieldName: "doesnotexist",
			errorGlob: "No such field with name: doesnotexist in data"},
		{name: "exported named exported ptr struct data - with incorrect field name",
			data: testExportedNamedExportedPtrStruct{}, fieldName: "wrong",
			errorGlob: "Field with name: wrong has wrong type: bool. Should be: thevent_test.TestStruct"},
		{name: "exported named exported ptr struct data - with correct field name",
			data: testExportedNamedExportedPtrStruct{}, fieldName: "Test"},
		// same struct event data
		{name: "same struct data - no handlers", data: TestStruct{}},
		{name: "same struct data - non-function handler", data: TestStruct{},
			handlers:  []thevent.Handler{TestStruct{}},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
		{name: "same struct data - valid handler", data: TestStruct{},
			handlers: []thevent.Handler{exportedTestStructHandler}},
		{name: "same struct data - mismatched handler", data: TestStruct{},
			handlers:  []thevent.Handler{intHandler},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
		{name: "same struct data - valid and mismatched handler", data: TestStruct{},
			handlers:  []thevent.Handler{exportedTestStructHandler, intHandler},
			errorGlob: "Handler uses incorrect data type. Expected: * Got: *"},
	}
	for _, tc := range exportedTestCases {
		t.Run(tc.name, func(t *testing.T) {
			if e, err := nonStructDataEvent.New(tc.data, tc.fieldName, tc.handlers...); err == nil {
				t.Error("Created sub-Event with non struct data parent Event. Sub-Event:", e)
			}
			_, err := exportedStructDataEvent.New(tc.data, tc.fieldName, tc.handlers...)
			errorMatchesGlob(t, err, tc.errorGlob)
		})
	}

}

func TestDispatchSubEvent(t *testing.T) {
	unexportedStructDataEvent, err := thevent.New(testStruct{})
	if err != nil {
		t.Fatal("Unable to crate event:", err)
	}
	exportedStructDataEvent, err := thevent.New(TestStruct{})
	if err != nil {
		t.Fatal("Unable to crate event:", err)
	}

	type newEventParams struct {
		data      thevent.Data
		fieldName string
	}

	unexportedCalled := 0
	unexportedTestStructHandler := func(ctx context.Context, e testStruct) error { // nolint: unparam
		unexportedCalled += e.v
		return nil
	}
	// nolint: unparam
	exportedNamedUnexportedStructHandler := func(ctx context.Context, e testExportedNamedUnexportedStruct) error {
		unexportedCalled += e.Test.v
		return nil
	}
	// nolint: unparam
	exportedNamedUnexportedPtrStructHandler := func(ctx context.Context,
		e testExportedNamedUnexportedPtrStruct) error {
		unexportedCalled += e.Test.v
		return nil
	}
	exportedCalled := 0
	exportedTestStructHandler := func(ctx context.Context, e TestStruct) error { // nolint: unparam
		exportedCalled += e.v
		return nil
	}
	// nolint: unparam
	exportedEmbeddedStructHandler := func(ctx context.Context, e testExportedEmbeddedStruct) error {
		exportedCalled += e.TestStruct.v
		return nil
	}
	// nolint: unparam
	exportedEmbeddedPtrStructHandler := func(ctx context.Context, e testExportedEmbeddedPtrStruct) error {
		exportedCalled += e.TestStruct.v
		return nil
	}
	// nolint: unparam
	exportedNamedExportedStructHandler := func(ctx context.Context, e testExportedNamedExportedStruct) error {
		exportedCalled += e.Test.v
		return nil
	}
	// nolint: unparam
	exportedNamedExportedPtrStructHandler := func(ctx context.Context, e testExportedNamedExportedPtrStruct) error {
		exportedCalled += e.Test.v
		return nil
	}

	unexportedChildEventA, err := unexportedStructDataEvent.New(testStruct{}, "", unexportedTestStructHandler)
	if err != nil {
		t.Fatal("Unable to create child Event:", err)
	}
	unexportedChildEventB, err := unexportedStructDataEvent.New(testExportedNamedUnexportedStruct{}, "Test",
		exportedNamedUnexportedStructHandler)
	if err != nil {
		t.Fatal("Unable to create child Event:", err)
	}
	unexportedChildEventC, err := unexportedStructDataEvent.New(testExportedNamedUnexportedPtrStruct{}, "Test",
		exportedNamedUnexportedPtrStructHandler)
	if err != nil {
		t.Fatal("Unable to create child Event:", err)
	}

	exportedChildEventA, err := exportedStructDataEvent.New(TestStruct{}, "", exportedTestStructHandler)
	if err != nil {
		t.Fatal("Unable to create child Event:", err)
	}
	exportedChildEventB, err := exportedStructDataEvent.New(testExportedEmbeddedStruct{}, "TestStruct",
		exportedEmbeddedStructHandler)
	if err != nil {
		t.Fatal("Unable to create child Event:", err)
	}
	exportedChildEventC, err := exportedStructDataEvent.New(testExportedEmbeddedPtrStruct{}, "TestStruct",
		exportedEmbeddedPtrStructHandler)
	if err != nil {
		t.Fatal("Unable to create child Event:", err)
	}
	exportedChildEventD, err := exportedStructDataEvent.New(testExportedNamedExportedStruct{}, "Test",
		exportedNamedExportedStructHandler)
	if err != nil {
		t.Fatal("Unable to create child Event:", err)
	}
	exportedChildEventE, err := exportedStructDataEvent.New(testExportedNamedExportedPtrStruct{}, "Test",
		exportedNamedExportedPtrStructHandler)
	if err != nil {
		t.Fatal("Unable to create child Event:", err)
	}

	// Test direct dispatch on child event
	ctx := context.Background()
	testCases := []struct {
		name          string
		e             *thevent.Event
		data          thevent.Data
		calledCounter *int
		incr          int
	}{
		// test unexported children bad data
		{name: "unexported child A bad data", e: unexportedChildEventA, data: 5, calledCounter: &unexportedCalled,
			incr: 0},
		{name: "unexported child B bad data", e: unexportedChildEventB, data: 5, calledCounter: &unexportedCalled,
			incr: 0},
		{name: "unexported child C bad data", e: unexportedChildEventC, data: 5, calledCounter: &unexportedCalled,
			incr: 0},
		// test unexported children good data
		{name: "unexported child A good data", e: unexportedChildEventA, data: testStruct{1},
			calledCounter: &unexportedCalled, incr: 1},
		{name: "unexported child B good data", e: unexportedChildEventB,
			data: testExportedNamedUnexportedStruct{Test: testStruct{1}}, calledCounter: &unexportedCalled,
			incr: 1},
		{name: "unexported child C good data", e: unexportedChildEventC,
			data: testExportedNamedUnexportedPtrStruct{Test: &testStruct{1}}, calledCounter: &unexportedCalled,
			incr: 1},
		// test exported children bad data
		{name: "exported child A bad data", e: exportedChildEventA, data: 5, calledCounter: &exportedCalled,
			incr: 0},
		{name: "exported child B bad data", e: exportedChildEventB, data: 5, calledCounter: &exportedCalled,
			incr: 0},
		{name: "exported child C bad data", e: exportedChildEventC, data: 5, calledCounter: &exportedCalled,
			incr: 0},
		{name: "exported child D bad data", e: exportedChildEventD, data: 5, calledCounter: &exportedCalled,
			incr: 0},
		{name: "exported child E bad data", e: exportedChildEventE, data: 5, calledCounter: &exportedCalled,
			incr: 0},
		// test exported children good data
		{name: "exported child A good data", e: exportedChildEventA, data: TestStruct{1},
			calledCounter: &exportedCalled, incr: 1},
		{name: "exported child B good data", e: exportedChildEventB,
			data: testExportedEmbeddedStruct{TestStruct: TestStruct{1}}, calledCounter: &exportedCalled, incr: 1},
		{name: "exported child C good data", e: exportedChildEventC,
			data: testExportedEmbeddedPtrStruct{TestStruct: &TestStruct{1}}, calledCounter: &exportedCalled,
			incr: 1},
		{name: "exported child D good data", e: exportedChildEventD,
			data: testExportedNamedExportedStruct{Test: TestStruct{1}}, calledCounter: &exportedCalled, incr: 1},
		{name: "exported child E good data", e: exportedChildEventE,
			data: testExportedNamedExportedPtrStruct{Test: &TestStruct{1}}, calledCounter: &exportedCalled,
			incr: 1},
		// test unexported parent bad data
		{name: "unexported parent bad data", e: unexportedStructDataEvent, data: 5,
			calledCounter: &unexportedCalled, incr: 0},
		// test unexported parent good data
		{name: "unexported parent good data", e: unexportedStructDataEvent, data: testStruct{1},
			calledCounter: &unexportedCalled, incr: 3},
		// test exported parent bad data
		{name: "exported parent bad data", e: exportedStructDataEvent, data: 5, calledCounter: &exportedCalled,
			incr: 0},
		// test exported parent good data
		{name: "exported parent good data", e: exportedStructDataEvent, data: TestStruct{1},
			calledCounter: &exportedCalled, incr: 5},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var prev int
			if tc.incr != 0 {
				prev = *tc.calledCounter
			}
			err := tc.e.Dispatch(ctx, tc.data)
			if err != nil {
				if tc.incr != 0 {
					t.Error("Unexpected error dispatching child event:", err)
				}
			} else {
				if tc.incr == 0 {
					t.Error("Expected an error triggring child event, but didn't get one")
				} else {
					if tc.calledCounter == nil {
						t.Fatal("Misconfigured test, unable to test dispatching")
					}
					if prev+tc.incr != *tc.calledCounter {
						t.Error("Handler to child event not dispatched")
					}
				}
			}
		})
	}
}

func TestMust(t *testing.T) {
	testCases := []struct {
		name        string
		data        thevent.Data
		handlers    []thevent.Handler
		expectPanic bool
	}{
		{name: "no handlers", data: 5},
		{name: "non-function handler", data: 5, handlers: []thevent.Handler{5}, expectPanic: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var e *thevent.Event
			defer func() {
				r := recover()
				if tc.expectPanic {
					if r == nil {
						t.Error("Expected a panic but didn't get one")
					}
					if e != nil {
						t.Error("Expected panic but a event was successfully created")
					}
				} else {
					if r != nil {
						t.Error("Recovered from unexpected panic:", r)
					}
					if e == nil {
						t.Error("Did not expect a panic but failed to create a new event")
					}
				}
			}()
			e = thevent.Must(thevent.New(tc.data, tc.handlers...))
		})
	}
}

func TestHandlersResultsErred(t *testing.T) {
	testCases := []struct {
		hr    thevent.HandlersResults
		erred bool
	}{
		{hr: thevent.HandlersResults{}, erred: false},
		{hr: thevent.HandlersResults{NumHandlers: 50}, erred: false},
		{hr: thevent.HandlersResults{NumHandlers: 50, Errors: []error{errors.New("error")}}, erred: true},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			if erred := tc.hr.Erred(); erred != tc.erred {
				t.Error("HandlersResults.Erred() returned:", erred, "expected:", tc.erred)
			}
		})
	}
}

func TestHandlersResultsErrorRate(t *testing.T) {
	testCases := []struct {
		hr        thevent.HandlersResults
		errorRate float32
	}{
		{hr: thevent.HandlersResults{}, errorRate: 0.0},
		{hr: thevent.HandlersResults{NumHandlers: 50}, errorRate: 0.0},
		{hr: thevent.HandlersResults{NumHandlers: 50, Errors: []error{errors.New("error")}}, errorRate: 0.02},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			if errorRate := tc.hr.ErrorRate(); errorRate != tc.errorRate {
				t.Error("HandlersResults.Erred() returned:", errorRate, "expected:", tc.errorRate)
			}
		})
	}
}
