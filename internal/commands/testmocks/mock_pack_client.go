// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/buildpacks/pack/internal/commands (interfaces: PackClient)

// Package testmocks is a generated GoMock package.
package testmocks

import (
	context "context"
	pack "github.com/buildpacks/pack"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockPackClient is a mock of PackClient interface
type MockPackClient struct {
	ctrl     *gomock.Controller
	recorder *MockPackClientMockRecorder
}

// MockPackClientMockRecorder is the mock recorder for MockPackClient
type MockPackClientMockRecorder struct {
	mock *MockPackClient
}

// NewMockPackClient creates a new mock instance
func NewMockPackClient(ctrl *gomock.Controller) *MockPackClient {
	mock := &MockPackClient{ctrl: ctrl}
	mock.recorder = &MockPackClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPackClient) EXPECT() *MockPackClientMockRecorder {
	return m.recorder
}

// Build mocks base method
func (m *MockPackClient) Build(arg0 context.Context, arg1 pack.BuildOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Build", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Build indicates an expected call of Build
func (mr *MockPackClientMockRecorder) Build(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Build", reflect.TypeOf((*MockPackClient)(nil).Build), arg0, arg1)
}

// CreateBuilder mocks base method
func (m *MockPackClient) CreateBuilder(arg0 context.Context, arg1 pack.CreateBuilderOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateBuilder", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateBuilder indicates an expected call of CreateBuilder
func (mr *MockPackClientMockRecorder) CreateBuilder(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateBuilder", reflect.TypeOf((*MockPackClient)(nil).CreateBuilder), arg0, arg1)
}

// InspectBuilder mocks base method
func (m *MockPackClient) InspectBuilder(arg0 string, arg1 bool, arg2 ...pack.BuilderInspectionModifier) (*pack.BuilderInfo, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "InspectBuilder", varargs...)
	ret0, _ := ret[0].(*pack.BuilderInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// InspectBuilder indicates an expected call of InspectBuilder
func (mr *MockPackClientMockRecorder) InspectBuilder(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InspectBuilder", reflect.TypeOf((*MockPackClient)(nil).InspectBuilder), varargs...)
}

// InspectBuildpack mocks base method
func (m *MockPackClient) InspectBuildpack(arg0 pack.InspectBuildpackOptions) (*pack.BuildpackInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InspectBuildpack", arg0)
	ret0, _ := ret[0].(*pack.BuildpackInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// InspectBuildpack indicates an expected call of InspectBuildpack
func (mr *MockPackClientMockRecorder) InspectBuildpack(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InspectBuildpack", reflect.TypeOf((*MockPackClient)(nil).InspectBuildpack), arg0)
}

// InspectImage mocks base method
func (m *MockPackClient) InspectImage(arg0 string, arg1 bool) (*pack.ImageInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InspectImage", arg0, arg1)
	ret0, _ := ret[0].(*pack.ImageInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// InspectImage indicates an expected call of InspectImage
func (mr *MockPackClientMockRecorder) InspectImage(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InspectImage", reflect.TypeOf((*MockPackClient)(nil).InspectImage), arg0, arg1)
}

// PackageBuildpack mocks base method
func (m *MockPackClient) PackageBuildpack(arg0 context.Context, arg1 pack.PackageBuildpackOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PackageBuildpack", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// PackageBuildpack indicates an expected call of PackageBuildpack
func (mr *MockPackClientMockRecorder) PackageBuildpack(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PackageBuildpack", reflect.TypeOf((*MockPackClient)(nil).PackageBuildpack), arg0, arg1)
}

// Rebase mocks base method
func (m *MockPackClient) Rebase(arg0 context.Context, arg1 pack.RebaseOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Rebase", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Rebase indicates an expected call of Rebase
func (mr *MockPackClientMockRecorder) Rebase(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Rebase", reflect.TypeOf((*MockPackClient)(nil).Rebase), arg0, arg1)
}

// RegisterBuildpack mocks base method
func (m *MockPackClient) RegisterBuildpack(arg0 context.Context, arg1 pack.RegisterBuildpackOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RegisterBuildpack", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// RegisterBuildpack indicates an expected call of RegisterBuildpack
func (mr *MockPackClientMockRecorder) RegisterBuildpack(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterBuildpack", reflect.TypeOf((*MockPackClient)(nil).RegisterBuildpack), arg0, arg1)
}

// YankBuildpack mocks base method
func (m *MockPackClient) YankBuildpack(arg0 pack.YankBuildpackOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "YankBuildpack", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// YankBuildpack indicates an expected call of YankBuildpack
func (mr *MockPackClientMockRecorder) YankBuildpack(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "YankBuildpack", reflect.TypeOf((*MockPackClient)(nil).YankBuildpack), arg0)
}
