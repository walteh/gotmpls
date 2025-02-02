// Code generated by mockery v2.51.0. DO NOT EDIT.

package mockery

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	finder "github.com/walteh/gotmpls/pkg/finder"
)

// MockTemplateFinder_finder is an autogenerated mock type for the TemplateFinder type
type MockTemplateFinder_finder struct {
	mock.Mock
}

type MockTemplateFinder_finder_Expecter struct {
	mock *mock.Mock
}

func (_m *MockTemplateFinder_finder) EXPECT() *MockTemplateFinder_finder_Expecter {
	return &MockTemplateFinder_finder_Expecter{mock: &_m.Mock}
}

// FindTemplates provides a mock function with given fields: ctx, dir, extensions
func (_m *MockTemplateFinder_finder) FindTemplates(ctx context.Context, dir string, extensions []string) ([]finder.FileInfo, error) {
	ret := _m.Called(ctx, dir, extensions)

	if len(ret) == 0 {
		panic("no return value specified for FindTemplates")
	}

	var r0 []finder.FileInfo
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []string) ([]finder.FileInfo, error)); ok {
		return rf(ctx, dir, extensions)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, []string) []finder.FileInfo); ok {
		r0 = rf(ctx, dir, extensions)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]finder.FileInfo)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, []string) error); ok {
		r1 = rf(ctx, dir, extensions)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockTemplateFinder_finder_FindTemplates_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'FindTemplates'
type MockTemplateFinder_finder_FindTemplates_Call struct {
	*mock.Call
}

// FindTemplates is a helper method to define mock.On call
//   - ctx context.Context
//   - dir string
//   - extensions []string
func (_e *MockTemplateFinder_finder_Expecter) FindTemplates(ctx interface{}, dir interface{}, extensions interface{}) *MockTemplateFinder_finder_FindTemplates_Call {
	return &MockTemplateFinder_finder_FindTemplates_Call{Call: _e.mock.On("FindTemplates", ctx, dir, extensions)}
}

func (_c *MockTemplateFinder_finder_FindTemplates_Call) Run(run func(ctx context.Context, dir string, extensions []string)) *MockTemplateFinder_finder_FindTemplates_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].([]string))
	})
	return _c
}

func (_c *MockTemplateFinder_finder_FindTemplates_Call) Return(_a0 []finder.FileInfo, _a1 error) *MockTemplateFinder_finder_FindTemplates_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockTemplateFinder_finder_FindTemplates_Call) RunAndReturn(run func(context.Context, string, []string) ([]finder.FileInfo, error)) *MockTemplateFinder_finder_FindTemplates_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockTemplateFinder_finder creates a new instance of MockTemplateFinder_finder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockTemplateFinder_finder(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockTemplateFinder_finder {
	mock := &MockTemplateFinder_finder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
