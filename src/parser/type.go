package parser

import "github.com/ark-lang/ark/src/util"

type Type interface {
	TypeName() string
	LevelsOfIndirection() int // number of pointers you have to go through to get to the actual type
	IsIntegerType() bool      // true for all int types
	IsFloatingType() bool     // true for all floating-point types
	IsSigned() bool           // true for all signed integer types
	CanCastTo(Type) bool      // true if the receiver can be typecast to the parameter
	Attrs() AttrGroup         // fetches the attributes associated with the type
	Equals(Type) bool         // compares whether two types are equal
	ActualType() Type         // returns the actual type disregarding named types
	IsVoidType() bool
}

//go:generate stringer -type=PrimitiveType
type PrimitiveType int

const (
	PRIMITIVE_s8 PrimitiveType = iota
	PRIMITIVE_s16
	PRIMITIVE_s32
	PRIMITIVE_s64
	PRIMITIVE_s128

	PRIMITIVE_u8
	PRIMITIVE_u16
	PRIMITIVE_u32
	PRIMITIVE_u64
	PRIMITIVE_u128

	PRIMITIVE_f32
	PRIMITIVE_f64
	PRIMITIVE_f128

	PRIMITIVE_rune

	PRIMITIVE_int
	PRIMITIVE_uint

	PRIMITIVE_bool
	PRIMITIVE_void
)

func (v PrimitiveType) IsVoidType() bool {
	return v == PRIMITIVE_void
}

func (v PrimitiveType) IsIntegerType() bool {
	switch v {
	case PRIMITIVE_s8, PRIMITIVE_s16, PRIMITIVE_s32, PRIMITIVE_s64, PRIMITIVE_s128,
		PRIMITIVE_u8, PRIMITIVE_u16, PRIMITIVE_u32, PRIMITIVE_u64, PRIMITIVE_u128,
		PRIMITIVE_int, PRIMITIVE_uint:
		return true
	default:
		return false
	}
}

func (v PrimitiveType) IsFloatingType() bool {
	switch v {
	case PRIMITIVE_f32, PRIMITIVE_f64, PRIMITIVE_f128:
		return true
	default:
		return false
	}
}

func (v PrimitiveType) IsSigned() bool {
	switch v {
	case PRIMITIVE_s8, PRIMITIVE_s16, PRIMITIVE_s32, PRIMITIVE_s64, PRIMITIVE_s128, PRIMITIVE_int:
		return true
	default:
		return false
	}
}

func (v PrimitiveType) TypeName() string {
	return v.String()[10:]
}

func (v PrimitiveType) LevelsOfIndirection() int {
	return 0
}

func (v PrimitiveType) CanCastTo(t Type) bool {
	return (v.IsIntegerType() || v.IsFloatingType() || v == PRIMITIVE_rune) &&
		(t.IsFloatingType() || t.IsIntegerType() || t == PRIMITIVE_rune)
}

func (v PrimitiveType) Attrs() AttrGroup {
	return nil
}

func (v PrimitiveType) Equals(t Type) bool {
	other, ok := t.(PrimitiveType)
	if !ok {
		return false
	}

	return v == other
}

func (v PrimitiveType) ActualType() Type {
	return v
}

// StructType

type StructType struct {
	Variables []*VariableDecl
	attrs     AttrGroup
}

func (v StructType) String() string {
	result := "(" + util.Blue("StructType") + ": "
	for _, attr := range v.attrs {
		result += attr.String() + " "
	}
	result += "\n"
	for _, decl := range v.Variables {
		result += "\t" + decl.String() + "\n"
	}
	return result + ")"
}

func (v StructType) TypeName() string {
	res := "struct {"

	for i, variable := range v.Variables {
		res += variable.Variable.Name + ": " + variable.Variable.Type.TypeName()

		if i < len(v.Variables)-1 {
			res += ", "
		}
	}

	return res + "}"
}

func (v StructType) IsSigned() bool {
	return false
}

func (v StructType) LevelsOfIndirection() int {
	return 0
}

func (v StructType) IsIntegerType() bool {
	return false
}

func (v StructType) IsVoidType() bool {
	return false
}

func (v StructType) IsFloatingType() bool {
	return false
}

func (v StructType) CanCastTo(t Type) bool {
	return false
}

func (v StructType) GetVariableDecl(s string) *VariableDecl {
	for _, decl := range v.Variables {
		if decl.Variable.Name == s {
			return decl
		}
	}
	return nil
}

func (v StructType) addVariableDecl(decl *VariableDecl) StructType {
	v.Variables = append(v.Variables, decl)
	decl.Variable.ParentStruct = v
	decl.Variable.FromStruct = true
	return v
}

func (v StructType) VariableIndex(d *Variable) int {
	for i, decl := range v.Variables {
		if decl.Variable == d {
			return i
		}
	}
	return -1
}

func (v StructType) Attrs() AttrGroup {
	return v.attrs
}

func (v StructType) Equals(t Type) bool {
	other, ok := t.(StructType)
	if !ok {
		return false
	}

	if !v.Attrs().Equals(other.Attrs()) {
		return false
	}

	if len(v.Variables) != len(other.Variables) {
		return false
	}

	for idx, _ := range v.Variables {
		variable, otherVariable := v.Variables[idx].Variable, other.Variables[idx].Variable
		if variable.Name != otherVariable.Name {
			return false
		}
		if !variable.Type.Equals(otherVariable.Type) {
			return false
		}
	}

	return true
}

func (v StructType) ActualType() Type {
	return v
}

// NamedType

type NamedType struct {
	Name         string
	Type         Type
	Parameters   []ParameterType
	ParentModule *Module
	Methods      []*Function
}

func (v *NamedType) addMethod(fn *Function) {
	v.Methods = append(v.Methods, fn)
}

func (v *NamedType) GetMethod(name string) *Function {
	for _, fn := range v.Methods {
		if fn.Name == name {
			return fn
		}
	}

	return nil
}

func (v *NamedType) ActualType() Type {
	return v.Type.ActualType()
}

func (v *NamedType) String() string {
	res := "(" + util.Blue("NamedType") + ": " + v.Name
	if len(v.Parameters) > 0 {
		res += "<"
		for idx, param := range v.Parameters {
			res += param.TypeName()
			if idx < len(v.Parameters)-1 {
				res += ", "
			}
		}
		res += ">"
	}
	return res + " = " + v.Type.TypeName() + ")"
}

func (v *NamedType) TypeName() string {
	return v.Name
}
func (v *NamedType) IsSigned() bool {
	return v.Type.IsSigned()
}

func (v *NamedType) LevelsOfIndirection() int {
	return v.Type.LevelsOfIndirection()
}

func (v *NamedType) IsVoidType() bool {
	return v.Type.IsVoidType()
}

func (v *NamedType) IsIntegerType() bool {
	return v.Type.IsIntegerType()
}

func (v *NamedType) IsFloatingType() bool {
	return v.Type.IsFloatingType()
}

func (v *NamedType) CanCastTo(t Type) bool {
	return v.ActualType().CanCastTo(t)
}

func (v *NamedType) Attrs() AttrGroup {
	return v.Type.Attrs()
}

func (v *NamedType) Equals(t Type) bool {
	other, ok := t.(*NamedType)
	if !ok {
		return false
	}

	if v.ParentModule != other.ParentModule {
		return false
	}

	if v.Name != other.Name {
		return false
	}

	return true
}

// ArrayType

type ArrayType struct {
	MemberType Type
	attrs      AttrGroup
}

// IMPORTANT:
// Using this function is no longer important, just make sure to use
// .Equals() to compare two types.
func ArrayOf(t Type) ArrayType {
	return ArrayType{MemberType: t}
}

func (v ArrayType) String() string {
	result := "(" + util.Blue("ArrayType") + ": "
	for _, attr := range v.attrs {
		result += attr.String() + " "
	}
	return result + v.TypeName() + ")" //+ util.Magenta(" <"+v.MangledName(MANGLE_ARK_UNSTABLE)+"> ") + ")"
}

func (v ArrayType) TypeName() string {
	return "[]" + v.MemberType.TypeName()
}

func (v ArrayType) IsSigned() bool {
	return false
}

func (v ArrayType) LevelsOfIndirection() int {
	return 0
}

func (v ArrayType) IsVoidType() bool {
	return false
}

func (v ArrayType) IsIntegerType() bool {
	return false
}

func (v ArrayType) IsFloatingType() bool {
	return false
}

func (v ArrayType) CanCastTo(t Type) bool {
	return t.ActualType().Equals(v)
}

func (v ArrayType) Attrs() AttrGroup {
	return v.attrs
}

func (v ArrayType) Equals(t Type) bool {
	other, ok := t.(ArrayType)
	if !ok {
		return false
	}

	if !v.Attrs().Equals(other.Attrs()) {
		return false
	}

	if !v.MemberType.Equals(other.MemberType) {
		return false
	}

	return true
}

func (v ArrayType) ActualType() Type {
	return v
}

// Constant Reference

type ConstantReferenceType struct {
	Referrer Type
}

func constantReferenceTo(t Type) ConstantReferenceType {
	return ConstantReferenceType{Referrer: t}
}

func (v ConstantReferenceType) TypeName() string {
	return "&" + v.Referrer.TypeName()
}

func (v ConstantReferenceType) LevelsOfIndirection() int {
	return v.Referrer.LevelsOfIndirection() + 1
}

func (v ConstantReferenceType) IsIntegerType() bool {
	return true
}

func (v ConstantReferenceType) IsFloatingType() bool {
	return false
}

func (v ConstantReferenceType) IsVoidType() bool {
	return false
}

func (v ConstantReferenceType) CanCastTo(t Type) bool {
	return t.IsIntegerType()
}

func (v ConstantReferenceType) Attrs() AttrGroup {
	return nil
}

func (v ConstantReferenceType) IsSigned() bool {
	return false
}

func (v ConstantReferenceType) Equals(t Type) bool {
	ref, ok := t.(ConstantReferenceType)
	if ok {
		return v.Referrer.Equals(ref.Referrer)
	}

	ptr, ok := t.(PointerType)
	if ok {
		return v.Referrer.Equals(ptr.Addressee)
	}

	return false
}

func (v ConstantReferenceType) ActualType() Type {
	return v
}

// Mutable Reference
type MutableReferenceType struct {
	Referrer Type
}

func mutableReferenceTo(t Type) MutableReferenceType {
	return MutableReferenceType{Referrer: t}
}

func (v MutableReferenceType) TypeName() string {
	return "&mut " + v.Referrer.TypeName()
}

func (v MutableReferenceType) LevelsOfIndirection() int {
	return v.Referrer.LevelsOfIndirection() + 1
}

func (v MutableReferenceType) IsIntegerType() bool {
	return true
}

func (v MutableReferenceType) IsFloatingType() bool {
	return false
}

func (v MutableReferenceType) IsVoidType() bool {
	return false
}

func (v MutableReferenceType) CanCastTo(t Type) bool {
	return t.IsIntegerType()
}

func (v MutableReferenceType) Attrs() AttrGroup {
	return nil
}

func (v MutableReferenceType) IsSigned() bool {
	return false
}

func (v MutableReferenceType) Equals(t Type) bool {
	ref, ok := t.(MutableReferenceType)
	if ok {
		return v.Referrer.Equals(ref.Referrer)
	}

	ptr, ok := t.(PointerType)
	if ok {
		return v.Referrer.Equals(ptr.Addressee)
	}

	return false
}

func (v MutableReferenceType) ActualType() Type {
	return v
}

// PointerType

type PointerType struct {
	Addressee Type
}

// IMPORTANT:
// Using this function is no longer important, just make sure to use
// .Equals() to compare two types.
func PointerTo(t Type) PointerType {
	return PointerType{Addressee: t}
}

func (v PointerType) TypeName() string {
	return "^" + v.Addressee.TypeName()
}

func (v PointerType) LevelsOfIndirection() int {
	return v.Addressee.LevelsOfIndirection() + 1
}

func (v PointerType) IsIntegerType() bool {
	return true
}

func (v PointerType) IsFloatingType() bool {
	return false
}

func (v PointerType) IsVoidType() bool {
	return false
}

func (v PointerType) CanCastTo(t Type) bool {
	if t.IsIntegerType() {
		return true
	}
	return false
}

func (v PointerType) Attrs() AttrGroup {
	return nil
}

func (v PointerType) IsSigned() bool {
	return false
}

func (v PointerType) Equals(t Type) bool {
	other, ok := t.(PointerType)
	if !ok {
		return false
	}

	return v.Addressee.Equals(other.Addressee)
}

func (v PointerType) ActualType() Type {
	return v
}

// TupleType

func tupleOf(types ...Type) Type {
	if len(types) == 1 {
		return types[0]
	}
	return TupleType{Members: types}
}

type TupleType struct {
	Members []Type
}

func (v TupleType) String() string {
	result := "(" + util.Blue("TupleType") + ": "
	for _, mem := range v.Members {
		result += "\t" + mem.TypeName() + "\n"
	}
	return result + ")"
}

func (v TupleType) TypeName() string {
	result := "("
	for idx, mem := range v.Members {
		result += mem.TypeName()

		// if we are not at the last component
		if idx < len(v.Members)-1 {
			result += ", "
		}
	}
	result += ")"
	return result
}

func (v TupleType) IsSigned() bool {
	return false
}

func (v TupleType) LevelsOfIndirection() int {
	return 0
}

func (v TupleType) IsIntegerType() bool {
	return false
}

func (v TupleType) IsFloatingType() bool {
	return false
}

func (v TupleType) IsVoidType() bool {
	return false
}

func (v TupleType) CanCastTo(t Type) bool {
	return v.Equals(t.ActualType())
}

func (v TupleType) addMember(decl Type) {
	v.Members = append(v.Members, decl)
}

func (v TupleType) Attrs() AttrGroup {
	return nil
}

func (v TupleType) Equals(t Type) bool {
	other, ok := t.(TupleType)
	if !ok {
		return false
	}

	if len(v.Members) != len(other.Members) {
		return false
	}

	for idx, mem := range v.Members {
		if !mem.Equals(other.Members[idx]) {
			return false
		}
	}

	return true
}

func (v TupleType) ActualType() Type {
	return v
}

// InterfaceType

type InterfaceType struct {
	Functions []*Function
	attrs     AttrGroup
}

func (v InterfaceType) String() string {
	result := "(" + util.Blue("InterfaceType") + ": "
	for _, function := range v.Functions {
		result += "\t" + function.String() + "\n"
	}
	result += "}"
	return result + ")"
}

func (v InterfaceType) TypeName() string {
	result := "interface {\n"
	for _, function := range v.Functions {
		result += "\t" + function.String() + "\n"
	}
	result += "}"
	return result
}

func (v InterfaceType) IsSigned() bool {
	return false
}

func (v InterfaceType) LevelsOfIndirection() int {
	return 0
}

func (v InterfaceType) IsIntegerType() bool {
	return false
}

func (v InterfaceType) IsFloatingType() bool {
	return false
}

func (v InterfaceType) IsVoidType() bool {
	return false
}

func (v InterfaceType) CanCastTo(t Type) bool {
	return v.Equals(t.ActualType())
}

func (v InterfaceType) addFunction(fn *Function) InterfaceType {
	v.Functions = append(v.Functions, fn)
	return v
}

func (v InterfaceType) MatchesType(t Type) {

}

func (v InterfaceType) Attrs() AttrGroup {
	return nil
}

func (v InterfaceType) Equals(t Type) bool {
	other, ok := t.(InterfaceType)
	if !ok {
		return false
	}

	if len(v.Functions) != len(other.Functions) {
		return false
	}

	for idx, mem := range v.Functions {
		if mem != other.Functions[idx] {
			return false
		}
	}

	return true
}

func (v InterfaceType) ActualType() Type {
	return v
}

// EnumType
type EnumType struct {
	Simple  bool
	Members []EnumTypeMember
	attrs   AttrGroup
}

type EnumTypeMember struct {
	Name string
	Type Type
	Tag  int
}

func (v EnumType) GetMember(name string) (EnumTypeMember, bool) {
	for _, member := range v.Members {
		if member.Name == name {
			return member, true
		}
	}
	return EnumTypeMember{}, false
}

func (v EnumType) String() string {
	result := "(" + util.Blue("EnumType") + ": "
	for _, attr := range v.attrs {
		result += attr.String() + " "
	}

	result += "\n"

	for _, mem := range v.Members {
		result += "\t" + mem.Name + ": " + mem.Type.TypeName() + "\n"
	}
	return result + ")"
}

func (v EnumType) TypeName() string {
	res := "enum {"

	for idx, mem := range v.Members {
		res += mem.Name + ": " + mem.Type.TypeName()
		if idx < len(v.Members)-1 {
			res += ", "
		}
	}

	return res + "}"
}

func (v EnumType) IsSigned() bool {
	return false
}

func (v EnumType) LevelsOfIndirection() int {
	return 0
}

func (v EnumType) IsIntegerType() bool {
	return v.Simple
}

func (v EnumType) IsFloatingType() bool {
	return false
}

func (v EnumType) IsVoidType() bool {
	return false
}

func (v EnumType) CanCastTo(t Type) bool {
	return v.Simple && t.IsIntegerType()
}

func (v EnumType) MemberIndex(name string) int {
	for idx, member := range v.Members {
		if member.Name == name {
			return idx
		}
	}
	return -1
}

func (v EnumType) Attrs() AttrGroup {
	return v.attrs
}

func (v EnumType) Equals(t Type) bool {
	other, ok := t.(EnumType)
	if !ok {
		return false
	}

	if !v.Attrs().Equals(other.Attrs()) {
		return false
	}

	if len(v.Members) != len(other.Members) {
		return false
	}

	for idx, member := range v.Members {
		otherMember := other.Members[idx]

		if member.Name != otherMember.Name {
			return false
		}
		if !member.Type.Equals(otherMember.Type) {
			return false
		}
		if member.Tag != otherMember.Tag {
			return false
		}
	}

	return true
}

func (v EnumType) ActualType() Type {
	return v
}

// FunctionType
type FunctionType struct {
	attrs AttrGroup

	Parameters []Type
	Return     Type
	IsVariadic bool

	Receiver Type // non-nil if non-static method
}

func (v FunctionType) String() string {
	result := "(" + util.Blue("FunctionType") + ": "
	return result + ")"
}

func (v FunctionType) TypeName() string {
	res := ""

	for _, attr := range v.attrs {
		res += "[" + attr.Key
		if attr.Value == "" {
			res += "]"
		} else {
			res += "=\"" + attr.Value + "\"] "
		}
	}

	res += "func("

	for idx, para := range v.Parameters {
		res += para.TypeName()
		if idx < len(v.Parameters)-1 {
			res += ", "
		}
	}

	res += ")"

	if v.Return != nil {
		res += " -> " + v.Return.TypeName()
	}

	return res
}

func (v FunctionType) IsSigned() bool {
	return false
}

func (v FunctionType) LevelsOfIndirection() int {
	return 0
}

func (v FunctionType) IsIntegerType() bool {
	return false
}

func (v FunctionType) IsFloatingType() bool {
	return false
}

func (v FunctionType) IsVoidType() bool {
	return false
}

func (v FunctionType) CanCastTo(t Type) bool {
	return false
}

func (v FunctionType) Attrs() AttrGroup {
	return v.attrs
}

func (v FunctionType) Equals(t Type) bool {
	other, ok := t.(FunctionType)
	if !ok {
		return false
	}

	if !v.Attrs().Equals(other.Attrs()) {
		return false
	}

	if v.IsVariadic != other.IsVariadic {
		return false
	}

	if !v.Return.Equals(other.Return) {
		return false
	}

	if len(v.Parameters) != len(other.Parameters) {
		return false
	}

	for i, par := range v.Parameters {
		if !par.Equals(other.Parameters[i]) {
			return false
		}
	}

	if (v.Receiver == nil) != (other.Receiver == nil) {
		return false
	}

	if v.Receiver != nil && !v.Receiver.Equals(other.Receiver) {
		return false
	}

	return true
}

func (v FunctionType) ActualType() Type {
	return v
}

// MetaType
type metaType struct {
}

func (v metaType) IsSigned() bool {
	panic("IsSigned() invalid on metaType")
}

func (v metaType) LevelsOfIndirection() int {
	panic("LevelsOfIndirection() invalid on metaType")
}

func (v metaType) IsIntegerType() bool {
	panic("IsIntegerType() invalid on metaType")
}

func (v metaType) IsFloatingType() bool {
	panic("IsFloatingType() invalid on metaType")
}

func (v metaType) IsVoidType() bool {
	panic("IsVoidType() invalid on metaType")
}

func (v metaType) CanCastTo(t Type) bool {
	panic("CanCastTo() invalid on metaType")
}

func (v metaType) Attrs() AttrGroup {
	panic("Attrs() invalid on metaType")
}

func (v metaType) Equals(t Type) bool {
	panic("Equals() invalid on metaType")
}

// ParameterType
type ParameterType struct {
	metaType
	Name string
}

func (v ParameterType) String() string {
	return "(" + util.Blue("ParameterType") + ": " + v.Name + ")"
}

func (v ParameterType) TypeName() string {
	return v.Name
}

func (v ParameterType) ActualType() Type {
	return v
}

// SubstitutionType
type SubstitutionType struct {
	metaType
	Name string
	Type Type
}

func (v SubstitutionType) String() string {
	return "(" + util.Blue("SubstitutionType") + ": " + v.Name + " = " + v.Type.TypeName() + ")"
}

func (v SubstitutionType) TypeName() string {
	return v.Name
}

func (v SubstitutionType) ActualType() Type {
	return v
}

// UnresolvedType
type UnresolvedType struct {
	metaType
	Name       UnresolvedName
	Parameters []Type
}

func (v UnresolvedType) String() string {
	res := "(" + util.Blue("UnresolvedType") + ": " + v.Name.String()
	if len(v.Parameters) > 0 {
		res += "<"
		for idx, param := range v.Parameters {
			res += param.TypeName()
			if idx < len(v.Parameters)-1 {
				res += ", "
			}
		}
		res += "> "
	}
	return res + ")"
}

func (v UnresolvedType) TypeName() string {
	return "unresolved(" + v.Name.String() + ")"
}

func (v UnresolvedType) ActualType() Type {
	return v
}

func TypeWithoutPointers(t Type) Type {
	if ptr, ok := t.(PointerType); ok {
		return TypeWithoutPointers(ptr.Addressee)
	}

	return t
}
