// 版权 @2022 凹语言 作者。保留所有权利。

package wat

/**************************************
instConst:
**************************************/
type instConst struct {
	anInstruction
	typ     ValueType
	literal string
}

func NewInstConst(typ ValueType, literal string) *instConst {
	return &instConst{typ: typ, literal: literal}
}
func (i *instConst) Format(indent string) string {
	return indent + i.typ.Name() + ".const " + i.literal
}
