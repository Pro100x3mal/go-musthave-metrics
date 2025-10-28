package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestIsPrimitive(t *testing.T) {
	tests := []struct {
		typ  string
		want bool
	}{
		{"int", true},
		{"int8", true},
		{"int16", true},
		{"int32", true},
		{"int64", true},
		{"uint", true},
		{"uint8", true},
		{"uint16", true},
		{"uint32", true},
		{"uint64", true},
		{"uintptr", true},
		{"float32", true},
		{"float64", true},
		{"complex64", true},
		{"complex128", true},
		{"byte", true},
		{"rune", true},
		{"string", true},
		{"bool", true},
		{"MyStruct", false},
		{"CustomType", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			if got := isPrimitive(tt.typ); got != tt.want {
				t.Errorf("isPrimitive(%q) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestZeroValue(t *testing.T) {
	tests := []struct {
		typ  string
		want string
	}{
		{"string", `""`},
		{"bool", "false"},
		{"byte", "0"},
		{"rune", "0"},
		{"int", "0"},
		{"int32", "0"},
		{"uint64", "0"},
		{"float32", "0"},
		{"complex128", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			if got := zeroValue(tt.typ); got != tt.want {
				t.Errorf("zeroValue(%q) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestGenResetForExpr(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "primitive int",
			code: "var x int",
			want: "r.field = 0\n",
		},
		{
			name: "primitive string",
			code: "var x string",
			want: "" + `r.field = ""` + "\n",
		},
		{
			name: "primitive bool",
			code: "var x bool",
			want: "r.field = false\n",
		},
		{
			name: "primitive float64",
			code: "var x float64",
			want: "r.field = 0\n",
		},
		{
			name: "pointer to primitive int",
			code: "var x *int",
			want: "if r.field != nil {\n*r.field = 0\n}\n",
		},
		{
			name: "pointer to primitive string",
			code: "var x *string",
			want: `if r.field != nil {` + "\n" + `*r.field = ""` + "\n}\n",
		},
		{
			name: "slice of primitives",
			code: "var x []int",
			want: "r.field = r.field[:0]\n",
		},
		{
			name: "slice of structs",
			code: "var x []MyStruct",
			want: "r.field = r.field[:0]\n",
		},
		{
			name: "slice of pointers",
			code: "var x []*MyStruct",
			want: "r.field = r.field[:0]\n",
		},
		{
			name: "map with primitive values",
			code: "var x map[string]int",
			want: "clear(r.field)\n",
		},
		{
			name: "map with struct values",
			code: "var x map[string]MyStruct",
			want: "clear(r.field)\n",
		},
		{
			name: "struct value",
			code: "var x MyStruct",
			want: "if resetter, ok := interface{}(&r.field).(interface{ Reset() }); ok {\nresetter.Reset()\n}\n",
		},
		{
			name: "pointer to struct",
			code: "var x *MyStruct",
			want: "if resetter, ok := r.field.(interface{ Reset() }); ok && r.field != nil {\nresetter.Reset()\n}\n",
		},
		{
			name: "pointer to slice",
			code: "var x *[]int",
			want: "if r.field != nil {\n*r.field = (*r.field)[:0]\n}\n",
		},
		{
			name: "pointer to map",
			code: "var x *map[string]int",
			want: "if r.field != nil {\nclear(*r.field)\n}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "", "package test\n"+tt.code, 0)
			if err != nil {
				t.Fatalf("failed to parse code: %v", err)
			}

			var expr ast.Expr
			ast.Inspect(file, func(n ast.Node) bool {
				if vs, ok := n.(*ast.ValueSpec); ok && len(vs.Names) > 0 {
					expr = vs.Type
					return false
				}
				return true
			})

			if expr == nil {
				t.Fatal("failed to extract expression from code")
			}

			got := genResetForExpr("r.field", expr)
			if got != tt.want {
				t.Errorf("genResetForExpr() = %q, want %q", got, tt.want)
			}
		})
	}
}
