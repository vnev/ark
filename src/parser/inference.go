package parser

import (
	"fmt"
	"os"

	"github.com/ark-lang/ark/src/util/log"

	"github.com/ark-lang/ark/src/util"
)

// IMPORTANT NOTE for setTypeHint():
// When implementing this function for an Expr, only set the Expr's Type if
// you are on a lowest-level Expr, ie. a literal. That means, if you Expr
// contains a pointer to another Expr(s), simple pass the type hint along to that
// Expr(s) then return.

type TypeInferer struct {
	Submodule     *Submodule
	functionStack []*Function
	function      *Function // the function we're in, or nil if we aren't
	shouldExit    bool
}

func (v *TypeInferer) pushFunction(f *Function) {
	v.functionStack = append(v.functionStack, f)
	v.function = f
}

func (v *TypeInferer) popFunction() {
	if len(v.functionStack) == 0 {
		panic("tried to pop empty function stack")
	}

	v.functionStack = v.functionStack[:len(v.functionStack)-1]
	if len(v.functionStack) == 0 {
		v.function = nil
	} else {
		v.function = v.functionStack[len(v.functionStack)-1]
	}
}

func (v *TypeInferer) err(thing Locatable, err string, stuff ...interface{}) {
	pos := thing.Pos()

	log.Error("semantic", util.TEXT_RED+util.TEXT_BOLD+"error:"+util.TEXT_RESET+" [%s:%d:%d] %s\n",
		pos.Filename, pos.Line, pos.Char, fmt.Sprintf(err, stuff...))

	log.Error("semantic", v.Submodule.File.MarkPos(pos))

	v.shouldExit = true
}

func (v *TypeInferer) warn(thing Locatable, err string, stuff ...interface{}) {
	pos := thing.Pos()

	log.Warning("semantic", util.TEXT_YELLOW+util.TEXT_BOLD+"warning:"+util.TEXT_RESET+" [%s:%d:%d] %s\n",
		pos.Filename, pos.Line, pos.Char, fmt.Sprintf(err, stuff...))

	log.Warning("semantic", v.Submodule.File.MarkPos(pos))
}

func (v *TypeInferer) Infer() {
	v.shouldExit = false

	for _, node := range v.Submodule.Nodes {
		node.infer(v)
	}

	if v.shouldExit {
		os.Exit(util.EXIT_FAILURE_SEMANTIC)
	}
}

func (v *Block) infer(s *TypeInferer) {
	for _, n := range v.Nodes {
		n.infer(s)
	}
}

func (v *Function) infer(s *TypeInferer) {
	s.pushFunction(v)
	if v.Body != nil {
		v.Body.infer(s)
	}
	s.popFunction()
}

func (v EnumType) infer(s *TypeInferer) {
	// We shouldn't need anything here
}

func (v StructType) infer(s *TypeInferer) {
	for _, decl := range v.Variables {
		decl.infer(s)
	}
}

func (v *NamedType) infer(s *TypeInferer) {
	if typ, ok := v.Type.(StructType); ok {
		typ.infer(s)
	} else if typ, ok := v.Type.(EnumType); ok {
		typ.infer(s)
	}
	// TODO add function types when done
}

func (v *Variable) infer(s *TypeInferer) {
}

/**
 * Declarations
 */

func (v *VariableDecl) infer(s *TypeInferer) {
	v.Variable.infer(s)

	if v.Assignment != nil {
		v.Assignment.setTypeHint(v.Variable.Type)
		v.Assignment.infer(s)

		if v.Variable.Type == nil {
			v.Variable.Type = v.Assignment.GetType()
			if v.Variable.Type.IsVoidType() {
				s.err(v, "variable has incomplete type `void`")
			}
		}
	}

}

func (v *TypeDecl) infer(s *TypeInferer) {
	v.NamedType.infer(s)
}

func (v *UseDirective) infer(s *TypeInferer) {
}

func (v *FunctionDecl) infer(s *TypeInferer) {
	v.Function.infer(s)
}

/*
 * Statements
 */

func (v *ReturnStat) infer(s *TypeInferer) {
	if s.function == nil {
		s.err(v, "Return statement must be in a function")
	}

	if !s.function.Type.Return.Equals(PRIMITIVE_void) {
		v.Value.setTypeHint(s.function.Type.Return)
		v.Value.infer(s)
	}
}

func (_ BreakStat) infer(s *TypeInferer) {}
func (_ NextStat) infer(s *TypeInferer)  {}

func (v *IfStat) infer(s *TypeInferer) {
	for _, expr := range v.Exprs {
		expr.setTypeHint(PRIMITIVE_bool)
		expr.infer(s)
	}

	for _, body := range v.Bodies {
		body.infer(s)
	}

	if v.Else != nil {
		v.Else.infer(s)
	}
}

// BlockStat

func (v *BlockStat) infer(s *TypeInferer) {
	v.Block.infer(s)
}

// CallStat

func (v *CallStat) infer(s *TypeInferer) {
	v.Call.infer(s)
}

// DeferStat

func (v *DeferStat) infer(s *TypeInferer) {
	v.Call.infer(s)
}

// AssignStat

func (v *AssignStat) infer(s *TypeInferer) {
	v.Assignment.setTypeHint(v.Access.GetType())
	v.Assignment.infer(s)
	v.Access.infer(s)
}

// BinopAssignStat

func (v *BinopAssignStat) infer(s *TypeInferer) {
	v.Assignment.setTypeHint(v.Access.GetType())
	v.Assignment.infer(s)
	v.Access.infer(s)
}

// LoopStat

func (v *LoopStat) infer(s *TypeInferer) {
	v.Body.infer(s)

	switch v.LoopType {
	case LOOP_TYPE_INFINITE:
	case LOOP_TYPE_CONDITIONAL:
		v.Condition.setTypeHint(PRIMITIVE_bool)
		v.Condition.infer(s)
	default:
		panic("invalid loop type")
	}
}

// MatchStat

func (v *MatchStat) infer(s *TypeInferer) {
	v.Target.infer(s)

	for pattern, stmt := range v.Branches {
		pattern.infer(s)
		stmt.infer(s)
	}
}

// DefaultStat

func (v *DefaultStat) infer(s *TypeInferer) {
	v.Target.infer(s)
}

/*
 * Expressions
 */

// UnaryExpr

func (v *UnaryExpr) infer(s *TypeInferer) {
	v.Expr.infer(s)

	if v.Expr.GetType() == nil {
		return // come back on second inference
	}

	switch v.Op {
	case UNOP_LOG_NOT:
		if v.Expr.GetType() == PRIMITIVE_bool {
			v.Type = PRIMITIVE_bool
		}
	case UNOP_BIT_NOT:
		if v.Expr.GetType().IsIntegerType() || v.Expr.GetType().IsFloatingType() {
			v.Type = v.Expr.GetType()
		}
	case UNOP_NEGATIVE:
		if v.Expr.GetType().IsIntegerType() || v.Expr.GetType().IsFloatingType() {
			v.Type = v.Expr.GetType()
		}
	default:
		panic("unknown unary op")
	}
}

func (v *UnaryExpr) setTypeHint(t Type) {
	switch v.Op {
	case UNOP_LOG_NOT:
		v.Expr.setTypeHint(PRIMITIVE_bool)
	case UNOP_BIT_NOT, UNOP_NEGATIVE:
		v.Expr.setTypeHint(t)
	default:
		panic("unknown unary op")
	}
}

// BinaryExpr

func (v *BinaryExpr) infer(s *TypeInferer) {

	switch v.Op {
	case BINOP_EQ, BINOP_NOT_EQ:
		v.Lhand.infer(s)
		v.Rhand.setTypeHint(v.Lhand.GetType())
		v.Rhand.infer(s)
		v.Type = PRIMITIVE_bool

	case BINOP_ADD, BINOP_SUB, BINOP_MUL, BINOP_DIV, BINOP_MOD,
		BINOP_GREATER, BINOP_LESS, BINOP_GREATER_EQ, BINOP_LESS_EQ,
		BINOP_BIT_AND, BINOP_BIT_OR, BINOP_BIT_XOR:
		v.Lhand.infer(s)
		v.Rhand.setTypeHint(v.Lhand.GetType())
		v.Rhand.infer(s)

		switch v.Op.Category() {
		case OP_ARITHMETIC:
			v.Type = v.Lhand.GetType()
		case OP_COMPARISON:
			v.Type = PRIMITIVE_bool
		default:
			s.err(v, "invalid operands specified `%s`", v.Op.String())
		}

	case BINOP_BIT_LEFT, BINOP_BIT_RIGHT:
		v.Lhand.infer(s)
		v.Rhand.infer(s)
		v.Type = v.Lhand.GetType()

	case BINOP_LOG_AND, BINOP_LOG_OR:
		v.Lhand.infer(s)
		v.Rhand.infer(s)
		v.Type = PRIMITIVE_bool

	default:
		panic("unimplemented bin operation")
	}
}

func (v *BinaryExpr) setTypeHint(t Type) {
	switch v.Op.Category() {
	case OP_ARITHMETIC, OP_BITWISE:
		if v.Op == BINOP_BIT_LEFT || v.Op == BINOP_BIT_RIGHT {
			v.Rhand.setTypeHint(nil)
			v.Lhand.setTypeHint(t)
			return
		}
		if t == nil {
			if v.Lhand.GetType() == nil && v.Rhand.GetType() != nil {
				v.Lhand.setTypeHint(v.Rhand.GetType())
				return
			} else if v.Rhand.GetType() == nil && v.Lhand.GetType() != nil {
				v.Rhand.setTypeHint(v.Lhand.GetType())
				return
			}
		}
		v.Lhand.setTypeHint(t)
		v.Rhand.setTypeHint(t)
	case OP_COMPARISON:
		if v.Lhand.GetType() == nil && v.Rhand.GetType() != nil {
			v.Lhand.setTypeHint(v.Rhand.GetType())
		} else if v.Rhand.GetType() == nil && v.Lhand.GetType() != nil {
			v.Rhand.setTypeHint(v.Lhand.GetType())
		} else {
			v.Lhand.setTypeHint(nil)
			v.Rhand.setTypeHint(nil)
		}
	case OP_LOGICAL:
		v.Lhand.setTypeHint(PRIMITIVE_bool)
		v.Rhand.setTypeHint(PRIMITIVE_bool)
	default:
		panic("missing opcategory")
	}
}

// NumericLiteral

func (v *NumericLiteral) infer(s *TypeInferer) {}

func (v *NumericLiteral) setTypeHint(t Type) {
	var actual Type
	if t != nil {
		actual = t.ActualType()
	}

	if v.IsFloat {
		switch actual {
		case PRIMITIVE_f32, PRIMITIVE_f64, PRIMITIVE_f128:
			v.Type = t

		default:
			v.Type = PRIMITIVE_f64
		}
	} else {
		switch actual {
		case PRIMITIVE_int, PRIMITIVE_uint,
			PRIMITIVE_s8, PRIMITIVE_s16, PRIMITIVE_s32, PRIMITIVE_s64, PRIMITIVE_s128,
			PRIMITIVE_u8, PRIMITIVE_u16, PRIMITIVE_u32, PRIMITIVE_u64, PRIMITIVE_u128,
			PRIMITIVE_f32, PRIMITIVE_f64, PRIMITIVE_f128:
			v.Type = t

		default:
			v.Type = PRIMITIVE_int
		}
	}
}

// StringLiteral

func (v *StringLiteral) infer(s *TypeInferer) {
	if v.Type == nil {
		if v.IsCString {
			v.Type = PointerTo(PRIMITIVE_u8)
		} else {
			v.Type = stringType
		}
	}
}

func (v *StringLiteral) setTypeHint(t Type) {

}

// RuneLiteral

func (v *RuneLiteral) infer(s *TypeInferer) {}
func (v *RuneLiteral) setTypeHint(t Type)   {}

// BoolLiteral

func (v *BoolLiteral) infer(s *TypeInferer) {}
func (v *BoolLiteral) setTypeHint(t Type)   {}

// ArrayLiteral

func (v *CompositeLiteral) infer(s *TypeInferer) {
	if array, ok := v.Type.ActualType().(ArrayType); ok {
		for _, val := range v.Values {
			val.setTypeHint(array.MemberType)
			val.infer(s)
		}
	} else if struc, ok := v.Type.ActualType().(StructType); ok {
		for i, val := range v.Values {
			field := v.Fields[i]

			if decl := struc.GetVariableDecl(field); decl != nil {
				val.setTypeHint(decl.Variable.Type)
			} else {
				val.setTypeHint(nil)
			}
			val.infer(s)
		}
	} else {
		s.err(v, "Invalid composite literal, expected array or structure")
	}

	/*var memType Type // type of each member of the array

	if v.Type == nil {
		memType = nil
	} else {
		arrayType, ok := v.Type.ActualType().(ArrayType)
		if !ok {
			s.err(v, "Invalid type")
		}
		memType = arrayType.MemberType
	}

	for _, mem := range v.Members {
		mem.setTypeHint(memType)
	}

	for _, mem := range v.Members {
		mem.infer(s)
	}

	if v.Type == nil {
		for _, mem := range v.Members {
			if mem.GetType() != nil {
				memType = mem.GetType()
				break
			}
		}

		if memType == nil {
			s.err(v, "Couldn't infer type of array members") // don't think this can ever happen
		}
		v.Type = ArrayOf(memType)
	}*/
}

func (v *CompositeLiteral) setTypeHint(t Type) {
	if t == nil {
		return
	}

	switch t.ActualType().(type) {
	case StructType, ArrayType:
		v.Type = t
	}
}

// CastExpr

func (v *CastExpr) infer(s *TypeInferer) {
	v.Expr.setTypeHint(nil)
	v.Expr.infer(s)
}

func (v *CastExpr) setTypeHint(t Type) {}

// CallExpr

func (v *CallExpr) infer(s *TypeInferer) {

	if sae, ok := v.Function.(*StructAccessExpr); ok {
		sae.Struct.infer(s)
		fn := TypeWithoutPointers(sae.Struct.GetType()).(*NamedType).GetMethod(sae.Member)
		v.Function = &FunctionAccessExpr{Function: fn}
		if v.Function == nil {
			//s.err(v, "Cannot resolve method `%s` of type `%s`", sae.Member, TypeWithoutPointers(sae.Struct.GetType()).TypeName())
		}
	} else {
		v.Function.infer(s)
	}

	if v.Function != nil {
		if _, ok := v.Function.GetType().(FunctionType); !ok {
			s.err(v, "Cannot call non-function") // TODO better error message
			return;
		}

		if v.Function.GetType().(FunctionType).Receiver != nil {
			recType := v.Function.GetType().(FunctionType).Receiver
			accessType := v.ReceiverAccess.GetType()

			if accessType.LevelsOfIndirection() == recType.LevelsOfIndirection()+1 {
				if ref, ok := v.ReceiverAccess.GetType().(ConstantReferenceType); ok {
					v.ReceiverAccess = &DerefAccessExpr{
						Type: ref.Referrer,
						Expr: v.ReceiverAccess,
					}
				}
				if ref, ok := v.ReceiverAccess.GetType().(MutableReferenceType); ok {
					v.ReceiverAccess = &DerefAccessExpr{
						Type: ref.Referrer,
						Expr: v.ReceiverAccess,
					}
				}
				if ptr, ok := v.ReceiverAccess.GetType().(PointerType); ok {
					v.ReceiverAccess = &DerefAccessExpr{
						Type: ptr.Addressee,
						Expr: v.ReceiverAccess,
					}
				}
			}
		}

		// attributes defaults
		for i, arg := range v.Arguments {
			if i >= len(v.Function.GetType().(FunctionType).Parameters) { // we have a variadic arg
				arg.setTypeHint(nil)
			} else {
				arg.setTypeHint(v.Function.GetType().(FunctionType).Parameters[i])
			}
		}
	}

	for _, arg := range v.Arguments {
		arg.infer(s)
	}
}

func (v *CallExpr) setTypeHint(t Type) {}

// VariableAccessExpr
func (v *VariableAccessExpr) infer(s *TypeInferer) {

}

func (v *VariableAccessExpr) setTypeHint(t Type) {}

// FunctionAccessExpr
func (_ FunctionAccessExpr) infer(s *TypeInferer) {}
func (_ FunctionAccessExpr) setTypeHint(t Type)   {}

// StructAccessExpr
func (v *StructAccessExpr) infer(s *TypeInferer) {
	v.Struct.infer(s)

	if v.Struct.GetType() == nil {
		s.err(v, "Type of access expression was nil")
	}

	typ := v.Struct.GetType().ActualType()
	if typ.LevelsOfIndirection() == 1 {
		typ = typ.(PointerType).Addressee.ActualType()
		v.Struct = &DerefAccessExpr{
			Type: typ,
			Expr: v.Struct,
		}
	}

	structType, ok := typ.(StructType)
	if !ok {
		s.err(v, "Cannot access member of type `%s`", v.Struct.GetType().TypeName())
	}

	// TODO check no mod access
	decl := structType.GetVariableDecl(v.Member)
	if decl == nil {
		s.err(v, "Struct `%s` does not contain member `%s`", structType.TypeName(), v.Member)
	} else {
		v.Variable = decl.Variable
	}
}

func (v *StructAccessExpr) setTypeHint(t Type) {}

// ArrayAccessExpr
func (v *ArrayAccessExpr) infer(s *TypeInferer) {
	v.Array.infer(s)

	v.Subscript.setTypeHint(PRIMITIVE_int)
	v.Subscript.infer(s)
}

func (v *ArrayAccessExpr) setTypeHint(t Type) {}

// TupleAccessExpr
func (v *TupleAccessExpr) infer(s *TypeInferer) {
	v.Tuple.infer(s)
}

func (v *TupleAccessExpr) setTypeHint(t Type) {}

// DerefAccessExpr

func (v *DerefAccessExpr) infer(s *TypeInferer) {
	v.Expr.infer(s)
	if pointerType, ok := v.Expr.GetType().(PointerType); ok {
		v.Type = pointerType.Addressee
	}

	if mutRef, ok := v.Expr.GetType().(MutableReferenceType); ok {
		v.Type = mutRef.Referrer
	}

	if constRef, ok := v.Expr.GetType().(ConstantReferenceType); ok {
		v.Type = constRef.Referrer
	}
}

func (v *DerefAccessExpr) setTypeHint(t Type) {
	v.Expr.setTypeHint(PointerTo(t))
}

// AddressOfExpr

func (v *AddressOfExpr) infer(s *TypeInferer) {
	v.Access.infer(s)
}

func (v *AddressOfExpr) setTypeHint(t Type) {
}

// ArrayLenExpr

func (v *ArrayLenExpr) infer(s *TypeInferer) {
	if v.Expr != nil {
		v.Expr.setTypeHint(nil)
		v.Expr.infer(s)
	}
}

func (v *ArrayLenExpr) setTypeHint(t Type) {}

// LambdaExpr

func (v *LambdaExpr) infer(s *TypeInferer) {
	v.Function.infer(s)
}

func (v LambdaExpr) setTypeHint(t Type) {}

// SizeofExpr

func (v *SizeofExpr) infer(s *TypeInferer) {
	if v.Expr != nil {
		v.Expr.setTypeHint(nil)
		v.Expr.infer(s)
	}
}

func (v *SizeofExpr) setTypeHint(t Type) {
}

// TupleLiteral

func (v *TupleLiteral) infer(s *TypeInferer) {
	var memberTypes []Type

	if v.Type != nil {
		tupleType, ok := v.Type.ActualType().(TupleType)
		if ok {
			memberTypes = tupleType.Members
		}
	}

	if len(v.Members) == len(memberTypes) {
		for idx, mem := range memberTypes {
			v.Members[idx].setTypeHint(mem)
		}
	} else {
		for _, mem := range v.Members {
			mem.setTypeHint(nil)
		}
	}

	for _, mem := range v.Members {
		mem.infer(s)
	}

	if v.Type == nil {
		var members []Type
		for _, mem := range v.Members {
			if mem.GetType() == nil {
				s.err(mem, "Couldn't infer type of tuple component")
			}
			members = append(members, mem.GetType())
		}

		v.Type = tupleOf(members...)
	}
}

func (v *TupleLiteral) setTypeHint(t Type) {
	if t == nil {
		return
	}

	_, ok := t.ActualType().(TupleType)
	if ok {
		v.Type = t
	}
}

// EnumLiteral
func (v *EnumLiteral) infer(s *TypeInferer) {
	if enumType, ok := v.Type.ActualType().(EnumType); ok {
		memIdx := enumType.MemberIndex(v.Member)

		if memIdx < 0 || memIdx >= len(enumType.Members) {
			return
		}

		memType := enumType.Members[memIdx].Type
		if structType, ok := memType.(StructType); ok && v.CompositeLiteral != nil {
			v.CompositeLiteral.setTypeHint(structType)
			v.CompositeLiteral.infer(s)
		} else if tupleType, ok := memType.(TupleType); ok && v.TupleLiteral != nil {
			v.TupleLiteral.setTypeHint(tupleType)
			v.TupleLiteral.infer(s)
		}
	}
}

func (v *EnumLiteral) setTypeHint(t Type) {
}

// DefaultMatchBranch

func (v *DefaultMatchBranch) infer(s *TypeInferer) {
}

func (v *DefaultMatchBranch) setTypeHint(t Type) {
}

// DefaultExpr

func (v *DefaultExpr) infer(s *TypeInferer) {
}

func (v *DefaultExpr) setTypeHint(t Type) {
}
