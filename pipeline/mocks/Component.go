// Code generated by mockery v1.0.0
package mocks

import app "github.com/ksonnet/ksonnet/metadata/app"
import component "github.com/bryanl/woowoo/component"
import mock "github.com/stretchr/testify/mock"

// Component is an autogenerated mock type for the Component type
type Component struct {
	mock.Mock
}

// Components provides a mock function with given fields: ns
func (_m *Component) Components(ns component.Namespace) ([]component.Component, error) {
	ret := _m.Called(ns)

	var r0 []component.Component
	if rf, ok := ret.Get(0).(func(component.Namespace) []component.Component); ok {
		r0 = rf(ns)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]component.Component)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(component.Namespace) error); ok {
		r1 = rf(ns)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// EnvParams provides a mock function with given fields: ksApp, envName
func (_m *Component) EnvParams(ksApp app.App, envName string) (string, error) {
	ret := _m.Called(ksApp, envName)

	var r0 string
	if rf, ok := ret.Get(0).(func(app.App, string) string); ok {
		r0 = rf(ksApp, envName)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(app.App, string) error); ok {
		r1 = rf(ksApp, envName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NSResolveParams provides a mock function with given fields: ns
func (_m *Component) NSResolveParams(ns component.Namespace) (string, error) {
	ret := _m.Called(ns)

	var r0 string
	if rf, ok := ret.Get(0).(func(component.Namespace) string); ok {
		r0 = rf(ns)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(component.Namespace) error); ok {
		r1 = rf(ns)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Namespace provides a mock function with given fields: ksApp, nsName
func (_m *Component) Namespace(ksApp app.App, nsName string) (component.Namespace, error) {
	ret := _m.Called(ksApp, nsName)

	var r0 component.Namespace
	if rf, ok := ret.Get(0).(func(app.App, string) component.Namespace); ok {
		r0 = rf(ksApp, nsName)
	} else {
		r0 = ret.Get(0).(component.Namespace)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(app.App, string) error); ok {
		r1 = rf(ksApp, nsName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Namespaces provides a mock function with given fields: ksApp, envName
func (_m *Component) Namespaces(ksApp app.App, envName string) ([]component.Namespace, error) {
	ret := _m.Called(ksApp, envName)

	var r0 []component.Namespace
	if rf, ok := ret.Get(0).(func(app.App, string) []component.Namespace); ok {
		r0 = rf(ksApp, envName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]component.Namespace)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(app.App, string) error); ok {
		r1 = rf(ksApp, envName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}