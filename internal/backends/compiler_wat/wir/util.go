// 版权 @2022 凹语言 作者。保留所有权利。

package wir

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"wa-lang.org/wa/internal/logger"
	"wa-lang.org/wa/internal/ssa"
	"wa-lang.org/wa/internal/types"
)

/*
func (m *Module) GenValueType(from types.Type) ValueType {
	switch t := from.(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.Bool, types.UntypedBool, types.Int, types.UntypedInt:
			return m.I32

		case types.Int32:
			if t.Name() == "rune" {
				return m.RUNE
			} else {
				return m.I32
			}

		case types.Uint32, types.Uintptr:
			return m.U32

		case types.Int64:
			return m.I64

		case types.Uint64:
			return m.U64

		case types.Float32, types.UntypedFloat:
			return m.F32

		case types.Float64:
			return m.F64

		case types.Int8:
			return m.I8

		case types.Uint8:
			return m.U8

		case types.Int16:
			return m.I16

		case types.Uint16:
			return m.U16

		case types.String:
			return m.STRING

		default:
			logger.Fatalf("Unknown type:%s", t)
			return nil
		}

	case *types.Tuple:
		switch t.Len() {
		case 0:
			return m.VOID

		case 1:
			return m.GenValueType(t.At(0).Type())

		default:
			var feilds []ValueType
			for i := 0; i < t.Len(); i++ {
				feilds = append(feilds, m.GenValueType(t.At(i).Type()))
			}
			return m.GenValueType_Tuple(feilds)
		}

	case *types.Pointer:
		tRef := m.GenValueType_Ref(m.GenValueType(t.Elem()))
		{
			methodset := m.ssaProg.MethodSets.MethodSet(t)
			for i := 0; i < methodset.Len(); i++ {
				sel := methodset.At(i)
				method := m.ssaProg.MethodValue(sel)

				var mtype FnType
				mtype.Name, _ = GetFnMangleName(method)
				mtype.FnSig = m.GenFnSig(method.Signature)

				tRef.AddMethodEntry(mtype)
			}
		}
		return tRef

	case *types.Named:
		switch ut := t.Underlying().(type) {
		case *types.Struct:
			var fs []Field
			for i := 0; i < ut.NumFields(); i++ {
				f := ut.Field(i)
				wtyp := m.GenValueType(f.Type())
				if f.Embedded() {
					fs = append(fs, NewField("$"+wtyp.Name(), wtyp))
				} else {
					fs = append(fs, NewField(GenSymbolName(f.Name()), wtyp))
				}
			}
			pkg_name, _ := GetPkgMangleName(t.Obj().Pkg().Path())
			obj_name := GenSymbolName(t.Obj().Name())
			tStruct := m.GenValueType_Struct(pkg_name+"."+obj_name, fs)
			{
				methodset := m.ssaProg.MethodSets.MethodSet(t)
				for i := 0; i < methodset.Len(); i++ {
					sel := methodset.At(i)
					method := m.ssaProg.MethodValue(sel)

					var mtype FnType
					mtype.Name, _ = GetFnMangleName(method)
					mtype.FnSig = m.GenFnSig(method.Signature)

					tStruct.AddMethodEntry(mtype)
				}
			}

			return tStruct

		case *types.Interface:
			if ut.NumMethods() == 0 {
				return m.GenValueType(ut)
			}
			pkg_name, _ := GetPkgMangleName(t.Obj().Pkg().Path())
			obj_name := GenSymbolName(t.Obj().Name())
			return m.GenValueType_Interface(pkg_name+"."+obj_name, ut)

		case *types.Signature:
			return m.GenValueType_Closure(ut)

		default:
			logger.Fatalf("Todo:%T", ut)
		}

	case *types.Array:
		return m.GenValueType_Array(m.GenValueType(t.Elem()), int(t.Len()))

	case *types.Slice:
		return m.GenValueType_Slice(m.GenValueType(t.Elem()))

	case *types.Signature:
		return m.GenValueType_Closure(t)

	case *types.Interface:
		if t.NumMethods() != 0 {
			panic("NumMethods of interface{} != 0")
		}
		return m.GenValueType_Interface("interface", t)

	default:
		logger.Fatalf("Todo:%T", t)
	}

	return nil
} //*/

func IsNumber(v Value) bool {
	switch v.Type().(type) {
	case *tI8, *tU8, *tI16, *tU16, *tI32, *tU32, *tI64, *tU64, *tF32, *tF64, *tBool:
		return true
	}

	return false
}

func GetFnMangleName(v interface{}) (internal string, external string) {
	switch f := v.(type) {
	case *ssa.Function:
		if recv := f.Signature.Recv(); recv != nil {
			internal, external = GetPkgMangleName(recv.Pkg().Path())

			internal += "."
			external += "."
			switch rt := recv.Type().(type) {
			case *types.Named:
				internal += GenSymbolName(rt.Obj().Name())
				external += GenSymbolName(rt.Obj().Name())

			case *types.Pointer:
				btype, ok := rt.Elem().(*types.Named)
				if !ok {
					panic("Unreachable")
				}
				internal += GenSymbolName(btype.Obj().Name())
				external += GenSymbolName(btype.Obj().Name())

			default:
				panic("Unreachable")
			}
		} else {
			if f.Pkg != nil {
				internal, external = GetPkgMangleName(f.Pkg.Pkg.Path())
			}
		}
		internal += "."
		external += "."
		internal += GenSymbolName(f.Name())
		external += GenSymbolName(f.Name())

	case *types.Func:
		internal, external = GetPkgMangleName(f.Pkg().Path())
		sig := f.Type().(*types.Signature)
		if recv := sig.Recv(); recv != nil {
			internal += "."
			external += "."
			switch rt := recv.Type().(type) {
			case *types.Named:
				internal += GenSymbolName(rt.Obj().Name())
				external += GenSymbolName(rt.Obj().Name())

			case *types.Pointer:
				btype, ok := rt.Elem().(*types.Named)
				if !ok {
					panic("Unreachable")
				}
				internal += GenSymbolName(btype.Obj().Name())
				external += GenSymbolName(btype.Obj().Name())

			default:
				panic("Unreachable")
			}
		}
		internal += "."
		external += "."
		internal += GenSymbolName(f.Name())
		external += GenSymbolName(f.Name())
	}

	return internal, external
}

func GetPkgMangleName(pkg_path string) (string, string) {
	var symbol_name, exp_name string
	for i := strings.IndexAny(pkg_path, "/\\"); i != -1; i = strings.IndexAny(pkg_path, "/\\") {
		p := pkg_path[:i]
		pkg_path = pkg_path[i+1:]

		exp_name += p
		exp_name += "$"

		symbol_name += GenSymbolName(p)
		symbol_name += "$"
	}
	exp_name += GenSymbolName(pkg_path)
	symbol_name += GenSymbolName(pkg_path)
	return symbol_name, exp_name
}

func GenSymbolName(src string) string {
	if len(src) == utf8.RuneCountInString(src) {
		return src
	}

	s := "$0x"
	for i := 0; i < len(src); i++ {
		s += strconv.FormatUint(uint64(src[i]), 16)
	}
	return s
}

func ExtractField(x Value, field_name string) Value {
	switch x := x.(type) {
	case *aStruct:
		return x.Extract(field_name)

	case *aRef:
		return x.Extract(field_name)

	case *aClosure:
		return x.Extract(field_name)

	default:
		logger.Fatalf("Todo:%T", x)
	}

	return nil
}
