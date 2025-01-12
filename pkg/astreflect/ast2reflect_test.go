package astreflect

// // TestStruct is used to test method reflection
// type TestStruct struct{}

// func (t *TestStruct) Method(x int) string { return "" }

// func TestAST2Reflect(t *testing.T) {
// 	tests := []struct {
// 		name          string
// 		input         types.Type
// 		expected      reflect.Type
// 		shouldError   bool
// 		skipRoundtrip bool // Some types can't be perfectly roundtripped
// 	}{
// 		{
// 			name:          "nil input",
// 			input:         nil,
// 			expected:      reflect.TypeOf((*interface{})(nil)).Elem(),
// 			skipRoundtrip: true, // nil -> interface{} -> interface{} is expected
// 		},
// 		{
// 			name:     "bool",
// 			input:    types.Typ[types.Bool],
// 			expected: reflect.TypeOf(bool(false)),
// 		},
// 		{
// 			name:     "int",
// 			input:    types.Typ[types.Int],
// 			expected: reflect.TypeOf(int(0)),
// 		},
// 		{
// 			name:     "string",
// 			input:    types.Typ[types.String],
// 			expected: reflect.TypeOf(string("")),
// 		},
// 		{
// 			name:     "float64",
// 			input:    types.Typ[types.Float64],
// 			expected: reflect.TypeOf(float64(0)),
// 		},
// 		{
// 			name:          "unsafe pointer",
// 			input:         types.Typ[types.UnsafePointer],
// 			expected:      reflect.TypeOf(unsafe.Pointer(nil)),
// 			skipRoundtrip: true, // unsafe.Pointer has special handling
// 		},
// 		{
// 			name:          "uintptr",
// 			input:         types.Typ[types.Uintptr],
// 			expected:      reflect.TypeOf(uintptr(0)),
// 			skipRoundtrip: true, // uintptr has special handling
// 		},
// 		{
// 			name:     "slice",
// 			input:    types.NewSlice(types.Typ[types.Int]),
// 			expected: reflect.TypeOf([]int{}),
// 		},
// 		{
// 			name:     "array",
// 			input:    types.NewArray(types.Typ[types.String], 3),
// 			expected: reflect.TypeOf([3]string{}),
// 		},
// 		{
// 			name:     "map",
// 			input:    types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
// 			expected: reflect.TypeOf(map[string]int{}),
// 		},
// 		{
// 			name:     "pointer",
// 			input:    types.NewPointer(types.Typ[types.String]),
// 			expected: reflect.TypeOf((*string)(nil)),
// 		},
// 		{
// 			name: "struct",
// 			input: types.NewStruct([]*types.Var{
// 				types.NewVar(0, nil, "Field1", types.Typ[types.Int]),
// 				types.NewVar(0, nil, "Field2", types.Typ[types.String]),
// 			}, []string{
// 				`json:"field1"`,
// 				`json:"field2"`,
// 			}),
// 			expected: reflect.TypeOf(struct {
// 				Field1 int    `json:"field1"`
// 				Field2 string `json:"field2"`
// 			}{}),
// 			skipRoundtrip: true, // Anonymous struct types can't be perfectly roundtripped
// 		},
// 		{
// 			name:     "empty interface",
// 			input:    types.NewInterfaceType(nil, nil),
// 			expected: reflect.TypeOf((*interface{})(nil)).Elem(),
// 		},
// 		{
// 			name: "non-empty interface",
// 			input: types.NewInterfaceType([]*types.Func{
// 				types.NewFunc(0, nil, "Method", types.NewSignature(nil, nil, nil, false)),
// 			}, nil),
// 			expected:      reflect.TypeOf((*interface{})(nil)).Elem(),
// 			skipRoundtrip: true, // Non-empty interfaces are converted to empty interfaces
// 		},
// 		{
// 			name: "simple function",
// 			input: types.NewSignature(nil,
// 				types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])),
// 				types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.String])),
// 				false),
// 			expected:      reflect.TypeOf(func(int) string { return "" }),
// 			skipRoundtrip: true, // Function types lose parameter names
// 		},
// 		{
// 			name: "variadic function",
// 			input: types.NewSignature(nil,
// 				types.NewTuple(types.NewVar(0, nil, "", types.NewSlice(types.Typ[types.Int]))),
// 				types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.String])),
// 				true),
// 			expected:      reflect.TypeOf(func(args ...int) string { return "" }),
// 			skipRoundtrip: true, // Function types lose parameter names and variadic info
// 		},
// 		{
// 			name: "method with receiver",
// 			input: types.NewSignature(
// 				types.NewVar(0, nil, "", types.NewPointer(types.NewStruct(nil, nil))),
// 				types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])),
// 				types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.String])),
// 				false),
// 			expected:      reflect.TypeOf(func(int) string { return "" }),
// 			skipRoundtrip: true, // Methods can't be perfectly roundtripped
// 		},
// 		{
// 			name:          "bidirectional channel",
// 			input:         types.NewChan(types.SendRecv, types.Typ[types.Int]),
// 			expected:      reflect.TypeOf(make(chan int)),
// 			skipRoundtrip: true, // Channel types lose directionality info
// 		},
// 		{
// 			name:          "send-only channel",
// 			input:         types.NewChan(types.SendOnly, types.Typ[types.Int]),
// 			expected:      reflect.TypeOf(make(chan<- int)),
// 			skipRoundtrip: true, // Channel types lose directionality info
// 		},
// 		{
// 			name:          "receive-only channel",
// 			input:         types.NewChan(types.RecvOnly, types.Typ[types.Int]),
// 			expected:      reflect.TypeOf(make(<-chan int)),
// 			skipRoundtrip: true, // Channel types lose directionality info
// 		},
// 		{
// 			name: "nested types",
// 			input: types.NewSlice(types.NewMap(types.Typ[types.String],
// 				types.NewPointer(types.NewStruct([]*types.Var{
// 					types.NewVar(0, nil, "Field", types.Typ[types.Int]),
// 				}, []string{""})))),
// 			expected: reflect.TypeOf([]map[string]*struct {
// 				Field int
// 			}{}),
// 			skipRoundtrip: true, // Contains anonymous struct
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			result, err := AST2Reflect(tt.input)
// 			if tt.shouldError {
// 				assert.Error(t, err, "should return error")
// 				return
// 			}

// 			require.NoError(t, err, "should not return error")
// 			assert.Equal(t, tt.expected, result, "types should match")

// 			// Test roundtrip conversion
// 			if !tt.shouldError && !tt.skipRoundtrip {
// 				roundtrip := Reflect2AST(result)
// 				assert.True(t, types.Identical(tt.input, roundtrip),
// 					"roundtrip conversion should yield identical type")
// 			}
// 		})
// 	}
// }

// func TestAST2Reflect_ErrorCases(t *testing.T) {
// 	// Create a custom type that implements types.Type but will cause errors
// 	type errorType struct {
// 		types.Type
// 		name string
// 	}

// 	tests := []struct {
// 		name        string
// 		input       types.Type
// 		errorString string
// 	}{
// 		{
// 			name:        "unsupported type",
// 			input:       &errorType{name: "custom"},
// 			errorString: "unsupported type: *astreflect.errorType",
// 		},
// 		{
// 			name: "invalid array element",
// 			input: types.NewArray(&errorType{
// 				name: "invalid element",
// 			}, 1),
// 			errorString: "converting array element type",
// 		},
// 		{
// 			name: "invalid slice element",
// 			input: types.NewSlice(&errorType{
// 				name: "invalid element",
// 			}),
// 			errorString: "converting slice element type",
// 		},
// 		{
// 			name: "invalid map key",
// 			input: types.NewMap(&errorType{
// 				name: "invalid key",
// 			}, types.Typ[types.Int]),
// 			errorString: "converting map key type",
// 		},
// 		{
// 			name: "invalid map value",
// 			input: types.NewMap(types.Typ[types.String], &errorType{
// 				name: "invalid value",
// 			}),
// 			errorString: "converting map value type",
// 		},
// 		{
// 			name: "invalid pointer element",
// 			input: types.NewPointer(&errorType{
// 				name: "invalid element",
// 			}),
// 			errorString: "converting pointer element type",
// 		},
// 		{
// 			name: "invalid struct field",
// 			input: types.NewStruct([]*types.Var{
// 				types.NewVar(0, nil, "Field", &errorType{
// 					name: "invalid field",
// 				}),
// 			}, []string{""}),
// 			errorString: `converting field "Field" type`,
// 		},
// 		{
// 			name: "invalid channel element",
// 			input: types.NewChan(types.SendRecv, &errorType{
// 				name: "invalid element",
// 			}),
// 			errorString: "converting channel element type",
// 		},
// 		{
// 			name: "invalid function parameter",
// 			input: types.NewSignature(nil,
// 				types.NewTuple(types.NewVar(0, nil, "", &errorType{name: "invalid param"})),
// 				nil,
// 				false),
// 			errorString: "converting parameter 0 type",
// 		},
// 		{
// 			name: "invalid function result",
// 			input: types.NewSignature(nil,
// 				nil,
// 				types.NewTuple(types.NewVar(0, nil, "", &errorType{name: "invalid result"})),
// 				false),
// 			errorString: "converting result 0 type",
// 		},
// 		{
// 			name: "invalid function receiver",
// 			input: types.NewSignature(
// 				types.NewVar(0, nil, "", &errorType{name: "invalid receiver"}),
// 				nil,
// 				nil,
// 				false),
// 			errorString: "converting receiver type",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			_, err := AST2Reflect(tt.input)
// 			require.Error(t, err, "should return error")
// 			assert.Contains(t, err.Error(), tt.errorString,
// 				"error should contain expected string")
// 		})
// 	}
// }
