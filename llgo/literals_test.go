package main

import (
	"testing"
)

func TestLiteralSlice(t *testing.T)   { checkOutputEqual(t, "literals/slice.go") }
func TestLiteralStruct(t *testing.T)  { checkOutputEqual(t, "literals/struct.go") }
func TestLiteralFuncton(t *testing.T) { checkOutputEqual(t, "literals/func.go") }
