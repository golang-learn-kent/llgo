// Copyright 2013 The llgo Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package llgo

import (
	"code.google.com/p/go.tools/go/ssa"
	"code.google.com/p/go.tools/go/types"
	"github.com/axw/gollvm/llvm"
)

// createThunk creates a thunk from a
// given function and arguments, suitable for use with
// "defer" and "go".
func (fr *frame) createThunk(call *ssa.CallCommon) (thunk llvm.Value, arg llvm.Value) {
	var nonconstargs []llvm.Value
	var nonconsttypes []*types.Var
	var args []llvm.Value
	packArg := func(arg *LLVMValue) {
		if arg.LLVMValue().IsAConstant().C != nil {
			args = append(args, arg.LLVMValue())
		} else {
			args = append(args, llvm.Value{nil})
			nonconstargs = append(nonconstargs, arg.LLVMValue())
			nonconsttypes = append(nonconsttypes, types.NewField(0, nil, "", arg.Type(), true))
		}
	}

	_, isbuiltin := call.Value.(*ssa.Builtin)
	if !isbuiltin {
		packArg(fr.value(call.Value))
	}

	for _, arg := range call.Args {
		packArg(fr.value(arg))
	}

	structtype := types.NewStruct(nonconsttypes, nil)
	argstruct := fr.createTypeMalloc(structtype)
	for i, arg := range args {
		argptr := fr.builder.CreateStructGEP(argstruct, i, "")
		fr.builder.CreateStore(arg, argptr)
	}
	i8ptr := llvm.PointerType(llvm.Int8Type(), 0)
	thunkfntype := llvm.FunctionType(llvm.VoidType(), []llvm.Type{i8ptr}, false)
	thunkfn := llvm.AddFunction(fr.module.Module, "", thunkfntype)
	thunkfn.SetLinkage(llvm.InternalLinkage)
	thunk = fr.builder.CreateBitCast(thunkfn, i8ptr, "")
	arg = fr.builder.CreateBitCast(argstruct, i8ptr, "")
	return
	/*







	// Check if function pointer or context pointer is global/null.
	fnval := fn.LLVMValue()
	fnptr := fnval
	var nctx int
	var fnctx llvm.Value
	var fnctxindex uint64
	var globalfn bool
	if fnptr.Type().TypeKind() == llvm.StructTypeKind {
		fnptr = c.builder.CreateExtractValue(fnval, 0, "")
		fnctx = c.builder.CreateExtractValue(fnval, 1, "")
		globalfn = !fnptr.IsAFunction().IsNil()
		if !globalfn {
			nctx++
		}
		if !fnctx.IsNull() {
			fnctxindex = uint64(nctx)
			nctx++
		}
	} else {
		// We've got a raw global function pointer. Convert to <ptr,ctx>.
		fnval = llvm.ConstNull(c.types.ToLLVM(fn.Type()))
		fnval = llvm.ConstInsertValue(fnval, fnptr, []uint32{0})
		fn = c.NewValue(fnval, fn.Type())
		fnctx = llvm.ConstExtractValue(fnval, []uint32{1})
		globalfn = true
	}

	i8ptr := llvm.PointerType(llvm.Int8Type(), 0)
	llvmargs := make([]llvm.Value, len(args)+nctx)
	llvmargtypes := make([]llvm.Type, len(args)+nctx)
	for i, arg := range args {
		llvmargs[i+nctx] = arg.LLVMValue()
		llvmargtypes[i+nctx] = llvmargs[i+nctx].Type()
	}
	if !globalfn {
		llvmargtypes[0] = fnptr.Type()
		llvmargs[0] = fnptr
	}
	if !fnctx.IsNull() {
		llvmargtypes[fnctxindex] = fnctx.Type()
		llvmargs[fnctxindex] = fnctx
	}

	// TODO(axw) investigate an option for go statements
	// to allocate argument structure on the stack in the
	// initiator, and block until the spawned goroutine
	// has loaded the arguments from it.
	structtyp := llvm.StructType(llvmargtypes, false)
	argstruct := c.createTypeMalloc(structtyp)

	// Create a function that will take a pointer to a structure of the type
	// defined above, or no parameters if there are none to pass.
	fntype := llvm.FunctionType(llvm.VoidType(), []llvm.Type{argstruct.Type()}, false)
	indirectfn := llvm.AddFunction(c.module.Module, "", fntype)
	i8argstruct := c.builder.CreateBitCast(argstruct, i8ptr, "")
	currblock := c.builder.GetInsertBlock()
	c.builder.SetInsertPointAtEnd(llvm.AddBasicBlock(indirectfn, "entry"))
	argstruct = indirectfn.Param(0)
	newargs := make([]*LLVMValue, len(args))
	for i := range llvmargs[nctx:] {
		argptr := c.builder.CreateGEP(argstruct, []llvm.Value{
			llvm.ConstInt(llvm.Int32Type(), 0, false),
			llvm.ConstInt(llvm.Int32Type(), uint64(i+nctx), false)}, "")
		newargs[i] = c.NewValue(c.builder.CreateLoad(argptr, ""), args[i].Type())
	}

	// Unless we've got a global function, extract the
	// function pointer from the context.
	if !globalfn {
		fnval = llvm.Undef(fnval.Type())
		fnptrptr := c.builder.CreateGEP(argstruct, []llvm.Value{
			llvm.ConstInt(llvm.Int32Type(), 0, false),
			llvm.ConstInt(llvm.Int32Type(), 0, false)}, "")
		fnptr = c.builder.CreateLoad(fnptrptr, "")
		fnval = c.builder.CreateInsertValue(fnval, fnptr, 0, "")
	}
	if !fnctx.IsNull() {
		fnctxptr := c.builder.CreateGEP(argstruct, []llvm.Value{
			llvm.ConstInt(llvm.Int32Type(), 0, false),
			llvm.ConstInt(llvm.Int32Type(), fnctxindex, false)}, "")
		fnctx = c.builder.CreateLoad(fnctxptr, "")
		fnval = c.builder.CreateInsertValue(fnval, fnctx, 1, "")
		fn = c.NewValue(fnval, fn.Type())
	}
	c.createCall(fn, newargs)

	// Indirect function calls' return values are always ignored.
	c.builder.CreateRetVoid()
	c.builder.SetInsertPointAtEnd(currblock)

	fnval = llvm.Undef(c.types.ToLLVM(nilarytyp))
	indirectfn = c.builder.CreateBitCast(indirectfn, fnval.Type().StructElementTypes()[0], "")
	fnval = c.builder.CreateInsertValue(fnval, indirectfn, 0, "")
	fnval = c.builder.CreateInsertValue(fnval, i8argstruct, 1, "")
	fn = c.NewValue(fnval, nilarytyp)
	return fn
	*/
}